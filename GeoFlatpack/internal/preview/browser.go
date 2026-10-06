package preview

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/network"
	cdpruntime "github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

//go:embed assets
var assets embed.FS

type Result struct {
	PNG     []byte
	Warning string
}

// Service serializes requests and owns the browser, its temporary profile, and
// the local asset server. Canceling a render does not stop the persistent browser.
type Service struct {
	mu                         sync.Mutex
	ctx                        context.Context
	cancel                     context.CancelFunc
	browser                    context.Context
	stopBrowser, stopAllocator context.CancelFunc
	server                     *http.Server
	url                        string
	data                       dataCache
	diagnostics                chan string
}

func NewService() *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{ctx: ctx, cancel: cancel, diagnostics: make(chan string, 8)}
}

func FindBrowser() (string, error) {
	if path := os.Getenv("GFP_PREVIEW_BROWSER"); path != "" {
		if resolved, err := exec.LookPath(path); err == nil {
			return resolved, nil
		}
		return "", fmt.Errorf("Preview unavailable: GFP_PREVIEW_BROWSER is not an executable")
	}
	candidates := []string{"google-chrome", "chromium", "chromium-browser", "brave-browser", "chrome", "brave"}
	if runtime.GOOS == "darwin" {
		candidates = append(candidates, "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Chromium.app/Contents/MacOS/Chromium", "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser")
	} else if runtime.GOOS == "windows" {
		for _, base := range []string{os.Getenv("PROGRAMFILES"), os.Getenv("PROGRAMFILES(X86)"), os.Getenv("LOCALAPPDATA")} {
			if base != "" {
				candidates = append(candidates, filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"), filepath.Join(base, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"))
			}
		}
	}
	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("Preview unavailable: install Chrome or Chromium, or set GFP_PREVIEW_BROWSER")
}

func (s *Service) start(ctx context.Context) error {
	path, err := FindBrowser()
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	files, err := fs.Sub(assets, "assets")
	if err != nil {
		listener.Close()
		return err
	}
	s.server = &http.Server{Handler: http.FileServer(http.FS(files)), ReadHeaderTimeout: 5 * time.Second}
	s.url = "http://" + listener.Addr().String()
	go func() { _ = s.server.Serve(listener) }()
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.ExecPath(path), chromedp.Flag("use-angle", "swiftshader"), chromedp.Flag("enable-unsafe-swiftshader", true))
	allocator, stopAllocator := chromedp.NewExecAllocator(s.ctx, opts...)
	s.stopAllocator = stopAllocator
	s.browser, s.stopBrowser = chromedp.NewContext(allocator)
	chromedp.ListenTarget(s.browser, func(event any) {
		if exception, ok := event.(*cdpruntime.EventExceptionThrown); ok {
			message := exception.ExceptionDetails.Text
			if exception.ExceptionDetails.Exception != nil {
				message = exception.ExceptionDetails.Exception.Description
			}
			select {
			case s.diagnostics <- message:
			default:
			}
		}
	})
	// The first Run must use the browser lifetime context. Bound startup with
	// the request cancellation without attaching that lifetime to this render.
	stop := context.AfterFunc(ctx, s.stopBrowser)
	err = chromedp.Run(s.browser, chromedp.Navigate(s.url))
	stop()
	if err != nil {
		s.stopBrowser()
		s.stopAllocator()
		s.browser = nil
		_ = s.server.Close()
		return fmt.Errorf("Preview browser: %w", err)
	}
	return nil
}

func (s *Service) Render(ctx context.Context, request Request) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ctx.Err(); err != nil {
		return Result{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if request.Width <= 0 || request.Height <= 0 || request.Width > 4096 || request.Height > 4096 {
		return Result{}, fmt.Errorf("Preview unavailable: invalid image size")
	}
	doc, err := prepareCached(ctx, request, &s.data)
	if err != nil {
		return Result{}, err
	}
	if s.browser == nil {
		if err := s.start(ctx); err != nil {
			return Result{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	renderCtx, stopRender := context.WithCancel(s.browser)
	defer stopRender()
	stop := context.AfterFunc(ctx, stopRender)
	defer stop()
	var basemapFailed atomic.Bool
	remoteRequests := make(map[network.RequestID]bool)
	chromedp.ListenTarget(renderCtx, func(event any) {
		if request.Basemap == "" {
			return
		}
		remote := func(url string) bool { return strings.HasPrefix(url, "http") && !strings.HasPrefix(url, s.url+"/") }
		switch event := event.(type) {
		case *network.EventRequestWillBeSent:
			remoteRequests[event.RequestID] = remote(event.Request.URL)
		case *network.EventResponseReceived:
			if event.Response.Status >= 400 && remote(event.Response.URL) {
				basemapFailed.Store(true)
			}
		case *network.EventLoadingFailed:
			if !event.Canceled && remoteRequests[event.RequestID] {
				basemapFailed.Store(true)
			}
		}
	})
	raw, err := json.Marshal(doc)
	if err != nil {
		return Result{}, err
	}
	var status struct{ Warning, Error string }
	var png []byte
	err = chromedp.Run(renderCtx,
		network.Enable(),
		chromedp.EmulateViewport(int64(request.Width), int64(request.Height)),
		chromedp.Evaluate("window.renderPreview("+string(raw)+")", &status, func(p *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams { return p.WithAwaitPromise(true) }),
	)
	if err != nil {
		select {
		case diagnostic := <-s.diagnostics:
			return Result{}, fmt.Errorf("Preview unavailable: %s", diagnostic)
		default:
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return Result{}, fmt.Errorf("Preview unavailable: rendering timed out")
		}
		return Result{}, fmt.Errorf("Preview render: %w", err)
	}
	if status.Error != "" {
		return Result{}, fmt.Errorf("Preview unavailable: %s", status.Error)
	}
	if basemapFailed.Load() && status.Warning == "" {
		if err := chromedp.Run(renderCtx, chromedp.Evaluate("window.clearPreviewBasemap()", nil, func(p *cdpruntime.EvaluateParams) *cdpruntime.EvaluateParams { return p.WithAwaitPromise(true) })); err != nil {
			return Result{}, err
		}
		status.Warning = "Basemap unavailable"
	}
	if err := chromedp.Run(renderCtx, chromedp.FullScreenshot(&png, 100)); err != nil {
		return Result{}, err
	}
	return Result{PNG: png, Warning: status.Warning}, nil
}

func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	s.cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopBrowser != nil {
		s.stopBrowser()
	}
	if s.stopAllocator != nil {
		s.stopAllocator()
	}
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}
