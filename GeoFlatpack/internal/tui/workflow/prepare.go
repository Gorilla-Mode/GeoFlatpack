package workflow

import (
	"GeoFlatpack/internal/app"
	"GeoFlatpack/style/maplibre/svg"
	"errors"
	"io"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

type Session interface {
	Layers() []app.Layer
	Icons() map[string]svg.Svg
	Write([]app.StyleSelection) error
	Close() error
}

type PrepareFunc func(app.Options, io.Writer) (Session, error)

func PrepareInput(opts app.Options, out io.Writer) (Session, error) {
	session, err := app.Prepare(opts, out)
	if session == nil {
		return nil, err
	}

	return session, err
}

type PreparedMsg struct {
	Session Session
	Elapsed time.Duration
	Err     error
}

// Preparation retains the result even if Bubble Tea stops receiving messages.
// The lock prevents a queued command from starting after Run begins cleanup.
type Preparation struct {
	mu       sync.Mutex
	started  bool
	stopping bool
	done     chan struct{}
	result   PreparedMsg
	prepare  PrepareFunc
}

func NewPreparation(prepare PrepareFunc) *Preparation {
	return &Preparation{done: make(chan struct{}), prepare: prepare}
}

func (p *Preparation) Command(opts app.Options) tea.Cmd {
	return func() tea.Msg {
		p.mu.Lock()
		if p.stopping || p.started {
			p.mu.Unlock()

			return nil
		}

		p.started = true
		p.mu.Unlock()
		defer close(p.done)

		start := time.Now()
		session, err := p.prepare(opts, io.Discard)
		p.result = PreparedMsg{Session: session, Elapsed: time.Since(start), Err: err}

		return p.result
	}
}

// Finish is called by Run after terminal restoration. Never close a session
// while preparation is using it, including when the program was killed.
func (p *Preparation) Finish() error {
	p.mu.Lock()

	p.stopping = true
	started := p.started
	p.mu.Unlock()
	if !started {
		return nil
	}

	<-p.done
	err := p.result.Err
	if p.result.Session != nil {
		err = errors.Join(err, p.result.Session.Close())
	}

	return err
}
