package tui

import (
	"GeoFlatpack/internal/app"
	"GeoFlatpack/style/maplibre/svg"
	"errors"
	"io"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

type loadedSession interface {
	Layers() []app.Layer
	Icons() map[string]svg.Svg
	Close() error
}

type prepareFunc func(app.Options, io.Writer) (loadedSession, error)

func prepareInput(opts app.Options, out io.Writer) (loadedSession, error) {
	session, err := app.Prepare(opts, out)
	if session == nil {
		return nil, err
	}
	return session, err
}

type preparedMsg struct {
	session loadedSession
	elapsed time.Duration
	err     error
}

// preparation retains the result even if Bubble Tea stops receiving messages.
// The lock prevents a queued command from starting after Run begins cleanup.
type preparation struct {
	mu       sync.Mutex
	started  bool
	stopping bool
	done     chan struct{}
	result   preparedMsg
	prepare  prepareFunc
}

func newPreparation(prepare prepareFunc) *preparation {
	return &preparation{done: make(chan struct{}), prepare: prepare}
}

func (p *preparation) command(opts app.Options) tea.Cmd {
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
		p.result = preparedMsg{session: session, elapsed: time.Since(start), err: err}
		return p.result
	}
}

// finish is called by Run after terminal restoration. Never close a session
// while preparation is using it, including when the program was killed.
func (p *preparation) finish() error {
	p.mu.Lock()

	p.stopping = true
	started := p.started
	p.mu.Unlock()
	if !started {
		return nil
	}

	<-p.done
	err := p.result.err
	if p.result.session != nil {
		err = errors.Join(err, p.result.session.Close())
	}

	return err
}
