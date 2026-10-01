package main

import (
	"GeoFlatpack/convert"
	"GeoFlatpack/fgb"
	"GeoFlatpack/internal/cli"
	"GeoFlatpack/style/maplibre"
	"GeoFlatpack/validate"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type options struct {
	input         string
	output        string
	format        string
	verbose       bool
	writeFGB      bool
	writeStyle    bool
	skipFailures  bool
	forceEPSG4326 bool
}

func main() {
	var opts options
	flag.StringVar(&opts.input, "i", "", "Path to the input file")
	flag.StringVar(&opts.output, "o", "./", "Output base file path or directory (one FGB per layer)")
	flag.StringVar(&opts.format, "f", string(validate.FormatMapLibre), "Output format: maplibre or sld")
	flag.BoolVar(&opts.verbose, "v", false, "Verbose output")
	flag.BoolVar(&opts.writeFGB, "write-fgb", true, "Write one FlatGeobuf output file per input layer")
	flag.BoolVar(&opts.writeStyle, "write-style", true, "Prompt for each layer and write one shared .gen.maplibre.json stylesheet")
	flag.BoolVar(&opts.skipFailures, "skip-failures", false, "Skip feature conversion failures; skipped failures can produce incomplete output")
	flag.BoolVar(&opts.forceEPSG4326, "force-epsg:4326", true, "Reproject coordinates and CRS metadata to EPSG:4326 regardless of stylesheet output; use --force-epsg:4326=false to preserve the original CRS. Regenerate existing projected FGB files for the web app")

	flag.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, "Usage gfp -i <input file> -f <stylesheet format> -o <output base or directory>")
		flag.PrintDefaults()
	}

	flag.Parse()
	if opts.input == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}

	if err := run(opts, os.Stdin, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "gfp:", err)
		os.Exit(1)
	}
}

// run returns before main exits so every dataset and virtual file is released.
func run(opts options, in io.Reader, out io.Writer) (err error) {
	if err := validate.Gml(opts.input); err != nil {
		return err
	}
	if err := validate.Format(validate.StyleFormat(opts.format)); err != nil {
		return err
	}

	output, err := resolveOutput(opts.input, opts.output)
	if err != nil {
		return err
	}
	if opts.verbose {
		_, _ = fmt.Fprintln(out, "gfp: converting every GML layer to FlatGeobuf...")
	}

	files, err := convert.GmlToFgb(opts.input, opts.forceEPSG4326, opts.skipFailures)
	if err != nil {
		return err
	}
	defer func() {
		for _, file := range files {
			err = errors.Join(err, file.Close())
		}
	}()

	// Retain all parsed layers before any styling prompts or output writes.
	layers, err := loadLayers(files)
	if err != nil {
		return err
	}
	names := make([]string, len(files))
	for i, file := range files {
		names[i] = file.LayerName
		if opts.verbose {
			_, _ = fmt.Fprintf(out, "gfp: layer %q loaded into memory\n%s\n", file.LayerName, fgb.InspectFgb(layers[i]))
		}
	}

	paths := layerOutputPaths(output, names)
	var styleJSON []byte
	if opts.writeStyle {
		scanner := cli.NewScanner(in)
		inputs := make([]maplibre.LayerStyle, len(layers))
		for i, data := range layers {
			inputs[i] = maplibre.LayerStyle{Data: data, SourceID: strings.TrimSuffix(filepath.Base(paths[i]), filepath.Ext(paths[i]))}
			if len(data.Features) == 0 {
				continue
			}
			if _, err := fmt.Fprintf(out, "\nLayer %q (%s)\n", names[i], filepath.Base(paths[i])); err != nil {
				return fmt.Errorf("layer %q: %w", names[i], err)
			}
			field, styles, err := cli.PromptStyleWithScanner(data, scanner, out)
			if err != nil {
				return fmt.Errorf("layer %q: %w", names[i], err)
			}
			inputs[i].CategoryField, inputs[i].Styles = field, styles
		}
		mapStyle, err := maplibre.BuildMapLibreCollectionStyle(filepath.Base(output), inputs)
		if err != nil {
			return fmt.Errorf("build MapLibre style: %w", err)
		}
		styleJSON, err = json.MarshalIndent(mapStyle, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal MapLibre style: %w", err)
		}
	}

	if opts.writeFGB || opts.writeStyle {
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return err
		}
	}

	if opts.writeFGB {
		for i, file := range files {
			if err := convert.WriteFgb(file, paths[i]); err != nil {
				return fmt.Errorf("write layer %q: %w", file.LayerName, err)
			}
			if opts.verbose {
				_, _ = fmt.Fprintln(out, "gfp: FlatGeobuf written to", paths[i])
			}
		}
	}

	if opts.writeStyle {
		styleOutput := strings.TrimSuffix(output, filepath.Ext(output)) + ".gen.maplibre.json"
		if err := os.WriteFile(styleOutput, styleJSON, 0o644); err != nil {
			return fmt.Errorf("write MapLibre style: %w", err)
		}
		if opts.verbose {
			_, _ = fmt.Fprintln(out, "gfp: MapLibre style written to", styleOutput)
		}
	}
	return nil
}

func loadLayers(files []*convert.MemoryFGB) ([]*fgb.Fgb, error) {
	layers := make([]*fgb.Fgb, 0, len(files))
	for _, file := range files {
		src, err := file.OpenReader()
		if err != nil {
			return nil, fmt.Errorf("open layer %q: %w", file.LayerName, err)
		}
		// LoadFgb closes the reader, including on failure.
		data, err := fgb.LoadFgb(src)
		if err != nil {
			return nil, fmt.Errorf("load layer %q: %w", file.LayerName, err)
		}
		layers = append(layers, data)
	}
	return layers, nil
}

func resolveOutput(input, output string) (string, error) {
	info, err := os.Stat(output)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if output == "." || strings.HasSuffix(output, string(os.PathSeparator)) || (err == nil && info.IsDir()) {
		name := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
		output = filepath.Join(output, name+".fgb")
	} else if filepath.Ext(output) == "" {
		output += ".fgb"
	}

	return output, nil
}

func layerOutputPaths(output string, names []string) []string {
	if len(names) == 1 {
		return []string{output}
	}

	base := strings.TrimSuffix(output, filepath.Ext(output))
	paths := make([]string, len(names))
	used := make(map[string]bool)

	for i, name := range names {
		// A conservative portable filename alphabet also prevents path traversal.
		name = strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
				return r
			}
			return '_'
		}, name)
		if name == "" {
			name = "layer"
		}
		unique := name
		for suffix := 2; used[strings.ToLower(unique)]; suffix++ {
			unique = fmt.Sprintf("%s_%d", name, suffix)
		}
		used[strings.ToLower(unique)] = true
		paths[i] = base + "." + unique + ".fgb"
	}

	return paths
}
