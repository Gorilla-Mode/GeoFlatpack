package tui

import (
	"GeoFlatpack/internal/app"
	"errors"
	"io"

	tea "charm.land/bubbletea/v2"
)

// Run prepares input behind the loading screen on explicit streams and owns the
// session until exit. Errors are returned after Bubble Tea restores the terminal
// and preparation has finished and released its resources.
func Run(opts app.Options, in io.Reader, out io.Writer) error {
	return runModel(NewModel(opts), in, out)
}

func runModel(model *Model, in io.Reader, out io.Writer, options ...tea.ProgramOption) (err error) {
	defer func() {
		err = errors.Join(err, model.writeOperation.finish(), model.preparation.finish())
	}()

	options = append(options, tea.WithInput(in), tea.WithOutput(out))
	_, err = tea.NewProgram(model, options...).Run()

	return err
}
