package convert

import (
	"fmt"
	"os"

	"github.com/airbusgeo/godal"
)

// GmlToFgb loads a GML file into an in-memory GDAL vector dataset.
// The caller is responsible for closing the returned dataset.
func GmlToFgb(input string) (*godal.Dataset, error) {
	godal.RegisterAll()

	src, err := godal.Open(input,
		godal.VectorOnly(),
		godal.DriverOpenOption("WRITE_GFS=NO"),
	)

	if err != nil {
		return nil, err
	}

	defer func(src *godal.Dataset, opts ...godal.CloseOption) {
		err := src.Close(opts...)
		if err != nil {
			_, _ = fmt.Fprint(os.Stderr, "Error closing source dataset:", err)
		}
	}(src)

	return src.VectorTranslate("", nil, godal.Memory)
}

func WriteFgb(fgb *godal.Dataset, output string) error {
	dst, err := fgb.VectorTranslate(output, []string{"-f", "FlatGeobuf"})
	if err != nil {
		return err
	}

	return dst.Close()
}
