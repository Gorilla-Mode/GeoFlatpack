package main

import (
	"GeoFlatpack/convert"
	"GeoFlatpack/validate"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/airbusgeo/godal"
)

func main() {
	inputFile := flag.String("i", "", "Path to the input file")
	outputDir := flag.String("o", ".", "Path to the output directory")
	formatFlag := flag.String("f", string(validate.FormatMapLibre), "Output format: maplibre or sld")

	flag.Usage = func() {
		_, err := fmt.Fprintln(os.Stderr, "Usage gfp -i <input file> -f <stylesheet format> -o <output directory>")
		if err != nil {
			return
		}

		flag.PrintDefaults()
	}
	flag.Parse()

	if *inputFile == "" || flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}

	if err := validate.Gml(*inputFile); err != nil {
		_, err := fmt.Fprintln(os.Stderr, "gfp:", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}

	if err := validate.Format(validate.StyleFormat(*formatFlag)); err != nil {
		_, err := fmt.Fprintln(os.Stderr, "gfp:", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}

	name := strings.TrimSuffix(filepath.Base(*inputFile), filepath.Ext(*inputFile))
	output := filepath.Join(*outputDir, name+".fgb")
	fmt.Printf("Converting %s to %s\n", *inputFile, output)

	fgb, err := convert.GmlToFgb(*inputFile)
	if err != nil {
		_, err := fmt.Fprintln(os.Stderr, "gfp: failed to convert GML to FlatGeobuf:", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
	defer func(fgb *godal.Dataset, opts ...godal.CloseOption) {
		_ = fgb.Close(opts...)
	}(fgb)

	err = convert.WriteFgb(fgb, output)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "gfp: failed to write FlatGeobuf:", err)
		os.Exit(1)
	}
}
