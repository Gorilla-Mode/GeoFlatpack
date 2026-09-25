package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ValidateFile(file string) error {
	extenstion := filepath.Ext(file)
	if extenstion != ".gml" {
		return fmt.Errorf("input file must have a .gml extension")
	}

	return nil
}

func main() {
	inputFile := flag.String("i", "", "Path to the input file")
	outputDir := flag.String("o", "", "Path to the output directory")

	flag.Usage = func() {
		_, err := fmt.Fprintln(os.Stderr, "Usage gfp -i <input file> -o <output directory>")
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

	if err := ValidateFile(*inputFile); err != nil {
		_, err := fmt.Fprintln(os.Stderr, "gfp:", err)
		if err != nil {
			return
		}
		os.Exit(1)
	}

	name := strings.TrimSuffix(filepath.Base(*inputFile), filepath.Ext(*inputFile))
	output := filepath.Join(*outputDir, name+".gfp")
	fmt.Printf("Converting %s to %s\n", *inputFile, output)
}
