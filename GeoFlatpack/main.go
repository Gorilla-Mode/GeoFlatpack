package main

import (
	"GeoFlatpack/internal/app"
	"GeoFlatpack/internal/cli"
	"GeoFlatpack/internal/tui"
	"GeoFlatpack/validate"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	var opts app.Options
	var useCLI bool
	flag.BoolVar(&useCLI, "cli", false, "Use the existing CLI interface")
	flag.StringVar(&opts.Input, "i", "", "Path to the input file")
	flag.StringVar(&opts.Output, "o", "./", "Output base file path or directory (one FGB per layer)")
	flag.StringVar(&opts.Format, "f", string(validate.FormatMapLibre), "Output format: maplibre or sld")
	flag.BoolVar(&opts.Verbose, "v", false, "Verbose output")
	flag.BoolVar(&opts.WriteFGB, "write-fgb", true, "Write one FlatGeobuf output file per input layer")
	flag.BoolVar(&opts.WriteStyle, "write-style", true, "Prompt for each layer and write one shared .gen.maplibre.json stylesheet")
	flag.StringVar(&opts.SVGDir, "svg-dir", "", "Directory of SVG icons for point and stored line/polygon vertex styling")
	flag.BoolVar(&opts.SkipFailures, "skip-failures", false, "Skip feature conversion failures; skipped failures can produce incomplete output")
	flag.BoolVar(&opts.ForceEPSG4326, "force-epsg:4326", true, "Reproject coordinates and CRS metadata to EPSG:4326 regardless of stylesheet output; use --force-epsg:4326=false to preserve the original CRS. Regenerate existing projected FGB files for the web app")

	flag.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, "Usage gfp -i <input file> -f <stylesheet format> -o <output base or directory>")
		flag.PrintDefaults()
	}

	flag.Parse()
	if opts.Input == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}

	if err := run(opts, useCLI, os.Stdin, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "gfp:", err)
		os.Exit(1)
	}
}

// run returns before main exits so frontend cleanup can finish.
func run(opts app.Options, useCLI bool, in io.Reader, out io.Writer) error {
	if useCLI {
		return cli.Run(opts, in, out)
	}
	return tui.Run(opts, in, out)
}
