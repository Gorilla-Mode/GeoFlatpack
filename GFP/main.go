package main

import (
	"GeoFlatpack/convert"
	"GeoFlatpack/fgb"
	"GeoFlatpack/validate"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	inputFile := flag.String("i", "", "Path to the input file")
	outputDir := flag.String("o", ".", "Path to the output directory")
	formatFlag := flag.String("f", string(validate.FormatMapLibre), "Output format: maplibre or sld")
	verbose := flag.Bool("v", false, "Verbose output")

	flag.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, "Usage gfp -i <input file> -f <stylesheet format> -o <output directory>")

		flag.PrintDefaults()
	}
	flag.Parse()

	if *inputFile == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}

	if err := validate.Gml(*inputFile); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "gfp:", err)

		os.Exit(1)
	}

	if err := validate.Format(validate.StyleFormat(*formatFlag)); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "gfp:", err)

		os.Exit(1)
	}

	name := strings.TrimSuffix(filepath.Base(*inputFile), filepath.Ext(*inputFile))
	output := filepath.Join(*outputDir, name+".fgb")
	fmt.Printf("Converting %s to %s\n", *inputFile, output)

	memoryFGB, err := convert.GmlToFgb(*inputFile)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "gfp: failed to convert GML to FlatGeobuf:", err)

		os.Exit(1)
	}

	defer func(fgb *convert.MemoryFGB) {
		err := fgb.Close()
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "gfp: failed to close FlatGeobuf:", err)
		}
	}(memoryFGB)

	src, err := memoryFGB.OpenReader()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "gfp: failed to open FlatGeobuf:", err)
		os.Exit(1)
	}

	LoadedFgb, err := fgb.LoadFgb(src)
	if err != nil {
		return
	}

	if *verbose {
		fgb.InspectFgb(LoadedFgb)
	}

	err = convert.WriteFgb(memoryFGB, output)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "gfp: failed to write FlatGeobuf:", err)
		os.Exit(1)
	}
}
