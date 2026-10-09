package workflow

import (
	"sync"

	"GeoFlatpack/internal/app"
	tea "charm.land/bubbletea/v2"
)

type WriteDoneMsg struct {
	Operation *WriteOperation
	Err       error
}

// Like preparation, an operation owns queued/running work until cleanup. The
// result remains available even if the program stops receiving messages.
type WriteOperation struct {
	mu                sync.Mutex
	started, stopping bool
	done              chan struct{}
	err               error
}

func (w *WriteOperation) Command(session Session, selections []app.StyleSelection) tea.Cmd {
	return func() tea.Msg {
		w.mu.Lock()
		if w.started || w.stopping {
			w.mu.Unlock()

			return nil
		}

		w.started = true
		w.mu.Unlock()
		defer close(w.done)
		w.err = session.Write(selections)

		return WriteDoneMsg{Operation: w, Err: w.err}
	}
}

func (w *WriteOperation) Finish() error {
	if w == nil {
		return nil
	}

	w.mu.Lock()
	w.stopping = true
	started := w.started
	w.mu.Unlock()
	if !started {
		return nil
	}

	<-w.done

	return w.err
}
