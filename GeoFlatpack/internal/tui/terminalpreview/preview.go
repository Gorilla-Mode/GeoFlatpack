// Package terminalpreview owns terminal graphics and asynchronous map previews.
package terminalpreview

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"strings"
	"time"
	"unicode"

	"GeoFlatpack/fgb"
	"GeoFlatpack/internal/preview"
	"GeoFlatpack/internal/tui/editor"
	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/style/maplibre"
	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

type Renderer interface {
	Render(context.Context, preview.Request) (preview.Result, error)
	Close() error
}

type State struct {
	ctx                                 context.Context
	cancel                              context.CancelFunc
	renderer                            Renderer
	phase                               int // querying graphics, querying virtual placement, supported, unsupported
	base, nextID, imageID               int
	owned                               map[int]bool
	cellWidth, cellHeight               int
	generation                          int
	request                             preview.Request
	target, signature, message, warning string
	content                             string
	renderCancel                        context.CancelFunc
	suspended                           bool
	sample                              bool
	representatives                     map[representativeKey]representativeChoice
	images                              map[string]cachedPreview
	imageSize                           [4]int // pane columns/rows and terminal cell pixels
}

type cachedPreview struct {
	id                          int
	signature, content, warning string
}

type representativeKey struct {
	data       *fgb.Fgb
	field      string
	group      maplibre.StyleGroup
	conditions string
}

type representativeChoice struct {
	index int
	err   error
}

type previewProbeTimeoutMsg struct{ phase int }
type previewRenderMsg struct{ generation int }
type previewDoneMsg struct {
	generation int
	result     preview.Result
	err        error
}

type previewUploadedMsg struct {
	generation, id                      int
	target, signature, content, warning string
}

func New() *State {
	return NewWithRenderer(nil)
}

// NewWithRenderer injects the browser renderer; nil creates it on first use.
func NewWithRenderer(renderer Renderer) *State {
	ctx, cancel := context.WithCancel(context.Background())
	var seed [2]byte
	_, _ = rand.Read(seed[:])

	return &State{ctx: ctx, cancel: cancel, renderer: renderer, base: 0x100000 + (int(seed[0]) << 8) + int(seed[1]), owned: make(map[int]bool), cellWidth: 8, cellHeight: 16, message: "Checking terminal graphics support…"}
}

func (s *State) Init() tea.Cmd {
	opts := &kitty.Options{Action: kitty.Query, ID: s.base, Format: kitty.RGB, ImageWidth: 1, ImageHeight: 1, Transmission: kitty.Direct}

	// A 1×1 black RGB pixel, followed by cell/window pixel-size queries.
	query := ansi.KittyGraphics([]byte("AAAA"), opts.Options()...)

	return probeCommand(query+ansi.WindowOp(16)+ansi.WindowOp(14), 0)
}

// Start the deadline after dispatching the query, not while Init is scheduling it.
// A deadline only changes the status text; a matching late reply is still valid.
func probeCommand(query string, phase int) tea.Cmd {
	return tea.Sequence(tea.Raw(query), func() tea.Msg {
		return tea.Tick(time.Second, func(time.Time) tea.Msg { return previewProbeTimeoutMsg{phase} })()
	})
}

func probeName(phase int) string {
	if phase == 0 {
		return "Kitty graphics query"
	}
	return "virtual placement probe"
}

func safeProbeReply(payload []byte) string {
	text := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			return -1
		}
		return r
	}, string(payload))
	return string([]rune(text)[:min(160, len([]rune(text)))])
}

func deletePreviewImage(id int) string {
	opts := &kitty.Options{Action: kitty.Delete, ID: id, Delete: kitty.DeleteID, DeleteResources: true, Quiet: 2}

	return ansi.KittyGraphics(nil, opts.Options()...)
}

func (s *State) CleanupImages() string {
	var b strings.Builder
	for id := range s.owned {
		b.WriteString(deletePreviewImage(id))
	}

	return b.String()
}

func (s *State) Close() error {
	if s == nil {
		return nil
	}

	s.cancel()
	if s.renderCancel != nil {
		s.renderCancel()
	}

	if s.renderer != nil {
		return s.renderer.Close()
	}

	return nil
}

func (s *State) Input(msg tea.Msg, dimensions Dimensions) tea.Cmd {
	if s == nil || s.ctx.Err() != nil {
		return nil
	}

	switch msg := msg.(type) {
	case uv.CellSizeEvent:
		if msg.Width > 0 && msg.Height > 0 {
			s.cellWidth, s.cellHeight = msg.Width, msg.Height
		}
	case uv.PixelSizeEvent:
		if msg.Width > 0 && msg.Height > 0 && dimensions.TerminalWidth > 0 && dimensions.TerminalHeight > 0 {
			s.cellWidth, s.cellHeight = max(1, msg.Width/dimensions.TerminalWidth), max(1, msg.Height/dimensions.TerminalHeight)
		}
	case uv.KittyGraphicsEvent:
		if s.phase >= 2 || msg.Options.ID != s.base+s.phase {
			return nil
		}

		if string(msg.Payload) != "OK" {
			s.message = "Preview unsupported: terminal rejected " + probeName(s.phase) + ": " + safeProbeReply(msg.Payload)
			s.phase = 3
			if s.owned[msg.Options.ID] {
				delete(s.owned, msg.Options.ID)
				return tea.Raw(deletePreviewImage(msg.Options.ID))
			}
			return nil
		}

		if s.phase == 0 {
			s.phase = 1
			s.message = "Checking virtual placement support…"
			s.nextID++ // Reserve an ID distinct from the query and future previews.
			id := s.base + s.nextID
			var b bytes.Buffer
			if err := kitty.EncodeGraphics(&b, image.NewRGBA(image.Rect(0, 0, 1, 1)), &kitty.Options{Action: kitty.TransmitAndPut, ID: id, Format: kitty.PNG, Transmission: kitty.Direct, VirtualPlacement: true, Columns: 1, Rows: 1}); err != nil {
				s.phase, s.message = 3, "Preview unavailable: could not encode virtual placement probe"
				return nil
			}
			s.owned[id] = true
			return probeCommand(b.String(), s.phase)
		}

		if s.phase == 1 {
			s.phase, s.message = 2, "Loading preview…"
			delete(s.owned, msg.Options.ID)

			return tea.Raw(deletePreviewImage(msg.Options.ID))
		}
	case previewProbeTimeoutMsg:
		if s.phase < 2 && msg.phase == s.phase {
			s.message = "Preview unavailable: No response to " + probeName(s.phase) + "; still listening."
		}
	case previewRenderMsg:
		if msg.generation != s.generation || s.suspended || s.phase != 2 {
			return nil
		}

		if s.renderer == nil {
			s.renderer = preview.NewService()
		}

		ctx, cancel := context.WithCancel(s.ctx)
		s.renderCancel = cancel
		request, renderer, generation := s.request, s.renderer, s.generation

		return func() tea.Msg {
			result, err := renderer.Render(ctx, request)

			return previewDoneMsg{generation: generation, result: result, err: err}
		}
	case previewDoneMsg:
		if msg.generation != s.generation || s.suspended {
			return nil
		}

		if msg.err != nil {
			s.message, s.content, s.warning = msg.err.Error(), "", ""

			return nil
		}

		img, _, err := image.Decode(bytes.NewReader(msg.result.PNG))
		if err != nil {
			s.message, s.content, s.warning = "Preview unavailable: invalid rendered image", "", ""

			return nil
		}

		s.nextID++
		id := s.base + s.nextID
		r := dimensions.Region
		var b bytes.Buffer
		if err := kitty.EncodeGraphics(&b, img, &kitty.Options{Action: kitty.TransmitAndPut, ID: id, Format: kitty.PNG, Transmission: kitty.Direct, VirtualPlacement: true, Columns: r.Width, Rows: r.Height, Quiet: 2, Chunk: true}); err != nil {
			s.message = "Preview unavailable: " + err.Error()

			return nil
		}

		s.owned[id] = true
		ready := previewUploadedMsg{generation: s.generation, id: id, target: s.target, signature: s.signature, content: previewPlaceholders(id, r.Width, r.Height), warning: msg.result.Warning}

		return tea.Sequence(tea.Raw(b.String()), func() tea.Msg { return ready })
	case previewUploadedMsg:
		if msg.generation != s.generation || s.suspended {
			delete(s.owned, msg.id)

			return tea.Raw(deletePreviewImage(msg.id))
		}

		if s.images == nil {
			s.images = make(map[string]cachedPreview)
		}

		oldID := s.images[msg.target].id
		s.images[msg.target] = cachedPreview{id: msg.id, signature: msg.signature, content: msg.content, warning: msg.warning}
		s.imageID, s.content, s.warning, s.message = msg.id, msg.content, msg.warning, ""
		if oldID > 0 && oldID != msg.id {
			delete(s.owned, oldID)

			return tea.Raw(deletePreviewImage(oldID))
		}
	}

	return nil
}

func previewPlaceholders(id, width, height int) string {
	var b strings.Builder
	for y := 0; y < height; y++ {
		if y > 0 {
			b.WriteByte('\n')
		}

		fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm%c%c%c", id>>16&255, id>>8&255, id&255, kitty.Placeholder, kitty.Diacritic(y), kitty.Diacritic(0))
		b.WriteString(strings.Repeat(string(kitty.Placeholder), max(0, width-1)))
		b.WriteString("\x1b[39m")
	}

	return b.String()
}

func (s *State) Request(target editor.Target, dimensions Dimensions) (preview.Request, string, string, error) {
	request := preview.Request{Input: target.Input, Icons: target.Icons, Group: target.Group, Basemap: preview.Basemap, Sample: s.sample}
	if target.Group == nil {
		return request, "", "", fmt.Errorf("Choose a feature to preview")
	}

	group := *target.Group
	stack := request.Input.Styles[group]
	if request.Input.Conditional != nil {
		stack = request.Input.Conditional.Stack(group)
	}
	layer := request.Input
	key := representativeKey{data: layer.Data, field: request.Input.CategoryField, group: group, conditions: layer.Conditional.DefinitionKey()}
	if s.representatives == nil {
		s.representatives = make(map[representativeKey]representativeChoice)
	}

	choice, cached := s.representatives[key]
	if !cached {
		if layer.Conditional != nil {
			choice.index, choice.err = preview.ConditionalRepresentativeIndex(layer.Data, layer.Conditional, group)
		} else {
			choice.index, choice.err = preview.RepresentativeIndex(layer.Data, request.Input.CategoryField, group)
		}
		s.representatives[key] = choice
	}

	if choice.err != nil {
		return request, "", "", choice.err
	}

	r := dimensions.Region
	if r.Width < 8 || r.Height < 3 || r.Height > 256 {
		return request, "", "", fmt.Errorf("Expand terminal for preview")
	}

	request.Width, request.Height = min(4096, r.Width*s.cellWidth), min(4096, r.Height*s.cellHeight)
	targetKey := fmt.Sprintf("%d/%d/%d/%s/%s/%d/%t", target.Layer, target.Category, target.Feature, group.RuleID, group.GeometryType, choice.index, s.sample)
	if target.CollectionKey != "" {
		targetKey = fmt.Sprintf("%d/%s/%s/%s/%d/%t", target.Layer, target.CollectionKey, group.RuleID, group.GeometryType, choice.index, s.sample)
	}
	raw, err := json.Marshal([]any{targetKey, request.Width, request.Height, r.Width, r.Height, string(group.GeometryType), group.Category.String(), group.Category.FilterValue(), stack, key.conditions})
	if err != nil {
		return request, "", "", err
	}

	return request, targetKey, fmt.Sprintf("%x", sha256.Sum256(raw)), nil
}

func (s *State) Refresh(snapshot editor.Target, snapshotErr error, dimensions Dimensions, visible bool) tea.Cmd {
	if s == nil || s.phase != 2 {
		return nil
	}

	if !visible {
		if !s.suspended {
			s.generation++
			if s.renderCancel != nil {
				s.renderCancel()
			}

			s.suspended = true
		}

		return nil
	}

	resumed := s.suspended
	s.suspended = false
	r := dimensions.Region
	size := [4]int{r.Width, r.Height, s.cellWidth, s.cellHeight}
	var cleanup string
	if size != s.imageSize {
		// Placements and screenshots depend on both cell and pixel dimensions.
		// Invalidate pending uploads as well as resident images on a resize.
		s.generation++
		if s.renderCancel != nil {
			s.renderCancel()
		}

		cleanup = s.CleanupImages()
		clear(s.owned)
		clear(s.images)
		s.imageID, s.signature, s.content, s.warning = 0, "", "", ""
		s.imageSize = size
	}

	withCleanup := func(cmd tea.Cmd) tea.Cmd {
		if cleanup == "" {
			return cmd
		}

		return tea.Batch(tea.Raw(cleanup), cmd)
	}

	var request preview.Request
	var target, signature string
	err := snapshotErr
	if err == nil {
		request, target, signature, err = s.Request(snapshot, dimensions)
	}

	if err != nil {
		if s.signature != "" {
			s.generation++
			if s.renderCancel != nil {
				s.renderCancel()
			}
		}

		s.signature, s.content, s.message, s.warning = "", "", err.Error(), ""

		return withCleanup(nil)
	}

	if signature == s.signature && !resumed {
		return withCleanup(nil)
	}

	if s.renderCancel != nil {
		s.renderCancel()
	}

	s.generation++
	if image, ok := s.images[target]; ok && image.signature == signature {
		s.request, s.signature, s.target = request, signature, target
		s.imageID, s.content, s.warning, s.message = image.id, image.content, image.warning, ""

		return withCleanup(nil)
	}

	if target != s.target || resumed || request.Width != s.request.Width || request.Height != s.request.Height {
		s.content, s.warning = "", ""
	}

	s.request, s.signature, s.target, s.message = request, signature, target, "Loading preview…"
	generation := s.generation

	return withCleanup(tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return previewRenderMsg{generation} }))
}

func (s *State) Presentation(opts *panel.ScaffoldOptions) {
	if s != nil {
		opts.PreviewSample = s.sample
		opts.PreviewContent, opts.PreviewWarning = s.message, s.warning
		if s.content != "" && !s.suspended {
			opts.PreviewContent, opts.PreviewImage = s.content, true
		}
	}
}

// Dimensions separates terminal geometry from rendering and editor state.
type Dimensions struct {
	TerminalWidth, TerminalHeight int
	Region                        panel.PickerRect
}

func (s *State) Supported() bool { return s != nil && s.phase == 2 }

func (s *State) ToggleSample() { s.sample = !s.sample }

func (s *State) Quit() tea.Cmd {
	if s == nil {
		return tea.Quit
	}

	s.cancel()

	return tea.Sequence(tea.Raw(s.CleanupImages()), tea.Quit)
}
