package cli

import (
	"GeoFlatpack/internal/app"
	"errors"
	"fmt"
	"io"
	"path/filepath"
)

// Run processes every layer using the existing prompts and a shared scanner.
func Run(opts app.Options, in io.Reader, out io.Writer) (err error) {
	session, err := app.Prepare(opts, out)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, session.Close())
	}()

	var selections []app.StyleSelection
	if opts.WriteStyle {
		scanner := NewScanner(in)
		layers := session.Layers()
		selections = make([]app.StyleSelection, len(layers))
		for i, layer := range layers {
			if len(layer.Data.Features) == 0 {
				continue
			}
			if _, err := fmt.Fprintf(out, "\nLayer %q (%s)\n", layer.Name, filepath.Base(layer.OutputPath)); err != nil {
				return fmt.Errorf("layer %q: %w", layer.Name, err)
			}
			field, styles, err := PromptStyleWithScanner(layer.Data, scanner, out, StyleOptions{Icons: session.Icons(), WriteFGB: opts.WriteFGB})
			if err != nil {
				return fmt.Errorf("layer %q: %w", layer.Name, err)
			}
			selections[i] = app.StyleSelection{CategoryField: field, Styles: styles}
		}
	}
	return session.Write(selections)
}
