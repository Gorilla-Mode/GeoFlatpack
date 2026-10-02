// Package app owns the processing workflow shared by the frontends.
package app

import (
	"GeoFlatpack/convert"
	"GeoFlatpack/fgb"
	"GeoFlatpack/style/maplibre"
	"GeoFlatpack/style/maplibre/svg"
	"GeoFlatpack/validate"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Options contains workflow settings; the caller supplies flag defaults.
type Options struct {
	Input         string
	Output        string
	Format        string
	Verbose       bool
	WriteFGB      bool
	WriteStyle    bool
	SkipFailures  bool
	ForceEPSG4326 bool
	SVGDir        string
}

// Layer describes an input layer in source order. Data is read-only to callers.
type Layer struct {
	Name       string
	SourceID   string
	OutputPath string
	Data       *fgb.Fgb
}

// StyleSelection contains the styling choices for one layer.
type StyleSelection struct {
	CategoryField string
	Styles        map[maplibre.StyleGroup][]maplibre.RenderLayerStyle
}

// Session owns converted datasets until Close. It is not safe for concurrent use.
type Session struct {
	options     Options
	diagnostics io.Writer
	output      string
	files       []*convert.MemoryFGB
	layers      []Layer
	icons       map[string]svg.Svg
	closed      bool
}

// Prepare validates and loads every layer without prompting or writing outputs.
// On failure, any acquired datasets and virtual files are released.
func Prepare(opts Options, out io.Writer) (_ *Session, err error) {
	if out == nil {
		out = io.Discard
	}
	if err := validate.Gml(opts.Input); err != nil {
		return nil, err
	}
	if err := validate.Format(validate.StyleFormat(opts.Format)); err != nil {
		return nil, err
	}
	var icons map[string]svg.Svg
	if opts.SVGDir != "" {
		icons, err = svg.ReadSvgs(opts.SVGDir)
		if err != nil {
			return nil, fmt.Errorf("read SVG directory: %w", err)
		}
	}

	output, err := resolveOutput(opts.Input, opts.Output)
	if err != nil {
		return nil, err
	}

	if opts.Verbose {
		_, _ = fmt.Fprintln(out, "gfp: converting every GML layer to FlatGeobuf...")
	}

	files, err := convert.GmlToFgb(opts.Input, opts.ForceEPSG4326, opts.SkipFailures)
	if err != nil {
		return nil, err
	}
	session := &Session{options: opts, diagnostics: out, output: output, files: files, icons: icons}
	defer func() {
		if err != nil {
			err = errors.Join(err, session.Close())
		}
	}()

	// Retain all parsed layers before any styling prompts or output writes.
	layers, err := loadLayers(files)
	if err != nil {
		return nil, err
	}

	names := make([]string, len(files))
	for i, file := range files {
		names[i] = file.LayerName

		if opts.Verbose {
			_, _ = fmt.Fprintf(out, "gfp: layer %q loaded into memory\n%s\n", file.LayerName, fgb.InspectFgb(layers[i]))
		}
	}

	paths := layerOutputPaths(output, names)
	session.layers = make([]Layer, len(layers))
	for i, data := range layers {
		session.layers[i] = Layer{
			Name:       names[i],
			SourceID:   strings.TrimSuffix(filepath.Base(paths[i]), filepath.Ext(paths[i])),
			OutputPath: paths[i],
			Data:       data,
		}
	}
	return session, nil
}

// Layers returns metadata in source order. The parsed data remains read-only.
func (s *Session) Layers() []Layer {
	return append([]Layer(nil), s.layers...)
}

// Icons exposes the icon catalog for read-only use by frontends.
func (s *Session) Icons() map[string]svg.Svg {
	return s.icons
}

// Close releases all owned datasets and virtual files. Repeated calls are safe.
func (s *Session) Close() error {
	if s == nil || s.closed {
		return nil
	}
	s.closed = true
	var err error
	for _, file := range s.files {
		err = errors.Join(err, file.Close())
	}
	s.files = nil
	return err
}

// Write prepares companions and writes enabled outputs. When styling is enabled,
// selections must contain one entry per layer in Layers order, including an
// empty selection for each empty layer. Otherwise selections are ignored.
func (s *Session) Write(selections []StyleSelection) (err error) {
	if s.closed {
		return fmt.Errorf("session is closed")
	}
	opts, out, output, files := s.options, s.diagnostics, s.output, s.files
	if opts.WriteStyle && len(selections) != len(s.layers) {
		return fmt.Errorf("expected one style selection per layer: got %d, want %d", len(selections), len(s.layers))
	}
	layers := make([]*fgb.Fgb, len(s.layers))
	names := make([]string, len(s.layers))
	paths := make([]string, len(s.layers))
	for i, layer := range s.layers {
		layers[i], names[i], paths[i] = layer.Data, layer.Name, layer.OutputPath
	}
	outputLayers := append([]*fgb.Fgb(nil), layers...)
	var styleJSON []byte
	if opts.WriteStyle {
		inputs := make([]maplibre.LayerStyle, len(layers))
		for i, layer := range s.layers {
			inputs[i] = maplibre.LayerStyle{
				Data:          layer.Data,
				SourceID:      layer.SourceID,
				CategoryField: selections[i].CategoryField,
				Styles:        selections[i].Styles,
			}
		}
		if opts.WriteFGB {
			for i := range inputs {
				outputLayers[i], err = maplibre.PrepareVertexCompanions(&inputs[i])
				if err != nil {
					return fmt.Errorf("layer %q companions: %w", names[i], err)
				}
			}
		}
		mapStyle, err := maplibre.BuildMapLibreCollectionStyle(filepath.Base(output), inputs, s.icons)
		if err != nil {
			return fmt.Errorf("build MapLibre style: %w", err)
		}

		styleJSON, err = json.MarshalIndent(mapStyle, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal MapLibre style: %w", err)
		}
	}

	if opts.WriteFGB || opts.WriteStyle {
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return err
		}
	}

	if opts.WriteFGB {
		for i, file := range files {
			var writeErr error
			if outputLayers[i] != layers[i] {
				writeErr = fgb.WriteFgb(outputLayers[i], paths[i])
			} else {
				writeErr = convert.WriteFgb(file, paths[i])
			}
			if writeErr != nil {
				return fmt.Errorf("write layer %q: %w", file.LayerName, writeErr)
			}

			if opts.Verbose {
				_, _ = fmt.Fprintln(out, "gfp: FlatGeobuf written to", paths[i])
			}
		}
	}

	if opts.WriteStyle {
		styleOutput := strings.TrimSuffix(output, filepath.Ext(output)) + ".gen.maplibre.json"
		if err := os.WriteFile(styleOutput, styleJSON, 0o644); err != nil {
			return fmt.Errorf("write MapLibre style: %w", err)
		}

		if opts.Verbose {
			_, _ = fmt.Fprintln(out, "gfp: MapLibre style written to", styleOutput)
		}
	}
	return nil
}
