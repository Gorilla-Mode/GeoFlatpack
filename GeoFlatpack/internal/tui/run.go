package tui

import (
	"GeoFlatpack/internal/app"
	"io"

	tea "charm.land/bubbletea/v2"
)

// Run starts the scaffold on explicit streams. Bubble Tea restores the terminal
// and leaves the alternate screen before Run returns, including on errors.
func Run(opts app.Options, in io.Reader, out io.Writer) error {
	_, err := tea.NewProgram(NewModel(opts), tea.WithInput(in), tea.WithOutput(out)).Run()
	return err
}
