package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

	name := strings.TrimSuffix(filepath.Base(*inputFile), filepath.Ext(*inputFile))
	output := filepath.Join(*outputDir, name+".gfp")
	fmt.Printf("Converting %s to %s\n", *inputFile, output)
}
