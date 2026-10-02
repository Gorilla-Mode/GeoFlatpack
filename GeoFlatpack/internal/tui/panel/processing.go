package panel

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
	"github.com/charmbracelet/x/ansi"
)

// ProcessingState selects the processing status message to render.
type ProcessingState int

const (
	Loading ProcessingState = iota
	Complete
	Failure
)

// ProcessingOptions supplies processing content and its existing styles.
type ProcessingOptions struct {
	State        ProcessingState
	Input        string
	Quitting     bool
	Spinner      string
	Elapsed      time.Duration
	Err          error
	LayerNames   []string
	Loaded       bool
	Width        int
	TitleStyle   lipgloss.Style
	MutedStyle   lipgloss.Style
	SuccessStyle lipgloss.Style
	FailureStyle lipgloss.Style
	ElapsedStyle lipgloss.Style
}

// Processing renders status, elapsed time, and the input layer tree.
func Processing(opts ProcessingOptions) string {
	var status string
	switch opts.State {
	case Loading:
		message := "Processing input…"
		if opts.Quitting {
			message = "Finishing processing before exiting…"
		}
		status = opts.Spinner + " " + message
	case Failure:
		status = opts.FailureStyle.Render(fmt.Sprintf("Could not process input\n%s", opts.Err))
	default:
		status = opts.SuccessStyle.Render("✓ Processing complete")
	}

	filename := filepath.Base(opts.Input)
	layers := tree.Root(ansi.Wrap(filename, opts.Width, "")).RootStyle(opts.TitleStyle).
		EnumeratorStyle(opts.MutedStyle.PaddingRight(1)).
		IndenterStyle(opts.MutedStyle.PaddingRight(1))
	count := ""

	if opts.Loaded {
		for _, name := range opts.LayerNames {
			layers.Child(ansi.Wrap(name, max(1, opts.Width-4), ""))
		}

		count = opts.MutedStyle.Render(fmt.Sprintf("Loaded layers: %d", len(opts.LayerNames)))
	} else if opts.State == Loading {
		layers.Child(opts.MutedStyle.Render("Reading layers…"))
	}

	parts := []string{
		status,
		opts.ElapsedStyle.Render(fmt.Sprintf("◷ Elapsed %.1f s", opts.Elapsed.Seconds())),
		"",
		opts.MutedStyle.Render(filepath.Dir(opts.Input)),
		layers.String(),
	}

	if count != "" {
		parts = append(parts, count)
	}

	switch opts.State {
	case Complete:
		parts = append(parts, "", opts.TitleStyle.Render("↵ Continue"))
	case Loading:
	case Failure:
	}

	return strings.Join(parts, "\n")
}
