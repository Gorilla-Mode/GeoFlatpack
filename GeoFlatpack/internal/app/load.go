package app

import (
	"GeoFlatpack/convert"
	"GeoFlatpack/fgb"
	"fmt"
)

func loadLayers(files []*convert.MemoryFGB) ([]*fgb.Fgb, error) {
	layers := make([]*fgb.Fgb, 0, len(files))

	for _, file := range files {
		src, err := file.OpenReader()
		if err != nil {
			return nil, fmt.Errorf("open layer %q: %w", file.LayerName, err)
		}

		// LoadFgb closes the reader, including on failure.
		data, err := fgb.LoadFgb(src)
		if err != nil {
			return nil, fmt.Errorf("load layer %q: %w", file.LayerName, err)
		}

		layers = append(layers, data)
	}
	return layers, nil
}
