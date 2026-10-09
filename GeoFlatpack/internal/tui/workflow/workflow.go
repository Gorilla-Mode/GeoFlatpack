// Package workflow owns the prepared session and synchronized write lifecycle.
package workflow

import (
	"errors"
	"fmt"
	"path/filepath"

	"GeoFlatpack/internal/app"
	tea "charm.land/bubbletea/v2"
)

type Workflow struct {
	preparation *Preparation
	session     Session
	operation   *WriteOperation
	busy        bool
	status      string
	err         error
}

func New(prepare PrepareFunc) *Workflow {
	return &Workflow{preparation: NewPreparation(prepare)}
}

func (w *Workflow) Prepare(opts app.Options) tea.Cmd { return w.preparation.Command(opts) }

func (w *Workflow) Prepared(msg PreparedMsg) { w.session = msg.Session }

func (w *Workflow) Session() Session { return w.session }

func (w *Workflow) Busy() bool { return w.busy }

func (w *Workflow) Status() string { return w.status }

func (w *Workflow) Err() error { return w.err }

func (w *Workflow) Start(selections []app.StyleSelection, err error) tea.Cmd {
	if w.busy {
		return nil
	}

	if err != nil {
		w.err, w.status = err, err.Error()

		return nil
	}

	w.operation = &WriteOperation{done: make(chan struct{})}
	w.busy, w.err, w.status = true, nil, "Writing files…"

	return w.operation.Command(w.session, selections)
}

// Complete rejects results from an older operation, including a retry.
func (w *Workflow) Complete(msg WriteDoneMsg) bool {
	if msg.Operation == nil || msg.Operation != w.operation {
		return false
	}

	w.busy, w.err = false, msg.Err
	directory := "."
	if w.session != nil {
		if layers := w.session.Layers(); len(layers) > 0 {
			directory = filepath.Dir(layers[0].OutputPath)
		}
	}

	w.status = fmt.Sprintf("Written files to %q", directory)
	if msg.Err != nil {
		w.status = "Write failed: " + msg.Err.Error()
	}

	return true
}

func (w *Workflow) Quitting() { w.status = "Finishing write before exiting…" }

// Finish stops queued writes and waits for running work before closing input.
func (w *Workflow) Finish() error {
	writeErr := w.operation.Finish()

	return errors.Join(writeErr, w.preparation.Finish())
}
