package fgb

import (
	"errors"
	"fmt"
	"io"

	"github.com/gogama/flatgeobuf/flatgeobuf"
	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
)

type FGB struct {
	header   *flat.Header
	Features []flat.Feature
}

func LoadFgb(src io.Reader) (loadedFGB *FGB, err error) {
	reader := flatgeobuf.NewFileReader(src)
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			loadedFGB = nil
			err = errors.Join(err, closeErr)
		}
	}()

	header, err := reader.Header()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	features, err := reader.DataRem()
	if err != nil {
		return nil, fmt.Errorf("read features: %w", err)
	}

	return &FGB{
		header:   header,
		Features: features,
	}, nil
}

func PrintHeader(header *flat.Header) {
	h := header
	fmt.Println("FGB Header:")
	fmt.Printf("\tName: %s\n", h.Name())
	fmt.Printf("\tGeometry type: %s\n", h.GeometryType())
	fmt.Printf("\tFeature count: %d\n", h.FeaturesCount())
	fmt.Printf("\tHas Z: %t\n", h.HasZ())
	fmt.Printf("\tIndex node size: %d\n", h.IndexNodeSize())

	if h.EnvelopeLength() == 4 {
		fmt.Printf("\n\tBounds: (%g, %g) to (%g, %g)\n",
			h.Envelope(0), h.Envelope(1),
			h.Envelope(2), h.Envelope(3))
	}

	var crs flat.Crs
	if h.Crs(&crs) != nil {
		fmt.Printf("\tCRS: %s:%d\n\n", crs.Org(), crs.Code())
	}

	for i := 0; i < h.ColumnsLength(); i++ {
		var column flat.Column
		if h.Columns(&column, i) {
			fmt.Printf("\tField: %s (%s)\n", column.Name(), column.Type())
		}
	}
}

func InspectFgb(fgb *FGB) {
	PrintHeader(fgb.header)

	for i := range fgb.Features {
		geometry := fgb.Features[i].Geometry(&flat.Geometry{})
		geometryLength := geometry.XyLength()

		fmt.Println("\nFeature", i+1, "\nGeometry Type:", geometry.Type())
		for j := 0; j+1 < geometryLength; j++ {
			x := geometry.Xy(j)
			y := geometry.Xy(j + 1)

			fmt.Println("\tX:", x, "Y:", y)
		}
	}
}
