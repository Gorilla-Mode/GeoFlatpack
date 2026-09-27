package main

import (
	"GeoFlatpack/convert"
	"GeoFlatpack/fgb"
	"GeoFlatpack/internal/cli"
	"GeoFlatpack/style/maplibre"
	"GeoFlatpack/validate"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	//region Flag parsing

	inputFile := flag.String("i", "", "Path to the input file")
	outputDir := flag.String("o", "./", "Path to the output directory")
	formatFlag := flag.String("f", string(validate.FormatMapLibre), "Output format: maplibre or sld")
	verbose := flag.Bool("v", false, "Verbose output")
	writeFgb := flag.Bool("write-fgb", true, "Write the FlatGeobuf output file")
	writeStyle := flag.Bool("write-style", true, "Write the generated stylesheet output file")
	skipFailures := flag.Bool("skip-failures", false, "Skip feature conversion failures; skipped failures can produce incomplete output")
	forceEPSG4326 := flag.Bool("force-epsg:4326", true, "Reproject coordinates and CRS metadata to EPSG:4326 regardless of stylesheet output; use --force-epsg:4326=false to preserve the original CRS. Regenerate existing projected FGB files for the web app")

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

	if *writeFgb || *writeStyle {
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "gfp:", err)
		}
	}

	if *verbose {
		fmt.Println("gfp: converting GML to FlatGeobuf...")
	}

	//endregion

	memoryFGB, err := convert.GmlToFgb(*inputFile, *forceEPSG4326, *skipFailures)
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
		log.Fatal("gfp: failed to load FlatGeobuf: ", err)
	}

	if *verbose {
		fmt.Println("gfp: successfully loaded FlatGeobuf into memory")
		fmt.Println("\ngfp: inspecting FlatGeobuf...")
		s := strings.TrimSuffix(fgb.InspectFgb(LoadedFgb), "\n")
		fmt.Printf("\t%s\n", strings.ReplaceAll(s, "\n", "\n\t"))
	}

	if *writeFgb {
		if *verbose {
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

	if *writeStyle {
		field, paints, err := cli.PromptStyle(LoadedFgb, os.Stdin, os.Stderr)
		if err != nil {
			log.Fatal(err)
		}

		style, err := maplibre.BuildMapLibreStyle(
			LoadedFgb, filepath.Base(output), "", field, paints,
		)

		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "gfp: failed to build MapLibre style:", err)
			os.Exit(1)
		}

		styleJSON, err := json.MarshalIndent(style, "", "  ")
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "gfp: failed to marshal MapLibre style:", err)
			os.Exit(1)
		}

		styleOutput := strings.TrimSuffix(output, filepath.Ext(output)) + ".gen.maplibre.json"
		if err := os.WriteFile(styleOutput, styleJSON, 0644); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "gfp: failed to write MapLibre style:", err)
			os.Exit(1)
		}

		if *verbose {
			fmt.Println("gfp: MapLibre style written to", styleOutput)
		}
	}
}
