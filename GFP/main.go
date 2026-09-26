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
	//region Flag parsing

	inputFile := flag.String("i", "", "Path to the input file")
	outputDir := flag.String("o", "", "Path to the output directory")
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
	output := *outputDir

	info, err := os.Stat(output)
	if err != nil && !os.IsNotExist(err) {
		_, _ = fmt.Fprintln(os.Stderr, "gfp:", err)
	}

	if output == "." || strings.HasSuffix(output, string(os.PathSeparator)) ||
		(err == nil && info.IsDir()) {
		output = filepath.Join(output, name+".fgb")
	} else if filepath.Ext(output) == "" {
		output += ".fgb"
	}

	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "gfp:", err)
	}

	if *verbose {
		fmt.Println("gfp: converting GML to FlatGeobuf...")
	}

	//endregion

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

	if *verbose {
		fmt.Println("gfp: successfully converted GML to FlatGeobuf in vsimem")
		fmt.Println("\ngfp: exposing reader...")
	}

	src, err := memoryFGB.OpenReader()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "gfp: failed to open FlatGeobuf:", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Println("gfp: reader exposed successfully")
		fmt.Println("\ngfp: loading FlatGeobuf...")
	}

	LoadedFgb, err := fgb.LoadFgb(src)
	if err != nil {
		return
	}

	if *verbose {
		fmt.Println("gfp: successfully loaded FlatGeobuf into memory")
		fmt.Println("\ngfp: inspecting FlatGeobuf...")
		s := strings.TrimSuffix(fgb.InspectFgb(LoadedFgb), "\n")
		fmt.Printf("\t%s\n", strings.ReplaceAll(s, "\n", "\n\t"))
		fmt.Println("\ngfp: writing FlatGeobuf to", output, "...")
	}

	err = convert.WriteFgb(memoryFGB, output)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "gfp: failed to write FlatGeobuf:", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Println("gfp: FlatGeobuf written to", output)
	}
}
