package convert

import (
	"fmt"
	"os"

	"github.com/airbusgeo/godal"
)

func GmlToFgb(input string, output string) error {
	godal.RegisterAll()

	src, err := godal.Open(input,
		godal.VectorOnly(),
		godal.DriverOpenOption("WRITE_GFS=NO"),
	)

	if err != nil {
		return fmt.Errorf("failed to open input GML file: %v", err)
	}

	defer func(src *godal.Dataset, opts ...godal.CloseOption) {
		err := src.Close(opts...)
		if err != nil {
			_, err := fmt.Fprintf(os.Stderr, "failed to close input GML file: %v\n", err)
			if err != nil {
				return
			}
		}
	}(src)

	dst, err := src.VectorTranslate(output, []string{"-f", "FlatGeobuf"})
	if err != nil {
		return fmt.Errorf("failed to translate GML to FlatGeobuf: %v", err)
	}

	return dst.Close()
}
