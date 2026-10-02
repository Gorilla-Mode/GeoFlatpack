package fgb

import (
	"errors"
	"fmt"
	"io"

	"github.com/gogama/flatgeobuf/flatgeobuf"
)

func LoadFgb(src io.Reader) (loadedFGB *Fgb, err error) {
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

	raw := RawFgb{
		header:   header,
		Features: features,
	}
	return buildFGB(raw)
}

func buildFGB(rawFgb RawFgb) (*Fgb, error) {
	fgb := &Fgb{
		Header:   rawFgb.header,
		Features: make([]Feature, 0, len(rawFgb.Features)),
	}

	for i := range rawFgb.Features {
		props := make(map[string]any)

		props, f, err := getPropertiesFGB(rawFgb, i, props)
		if err != nil {
			return f, err
		}

		fgb.Features = append(fgb.Features, Feature{
			Raw:        rawFgb.Features[i],
			Properties: props,
		})
	}

	return fgb, nil
}

func getPropertiesFGB(rawFgb RawFgb, i int, props map[string]any) (map[string]any, *Fgb, error) {
	if rawFgb.Features[i].PropertiesLength() > 0 {
		var schema flatgeobuf.Schema = rawFgb.header
		if rawFgb.Features[i].ColumnsLength() > 0 {
			schema = &rawFgb.Features[i]
		}

		values, err := readProperties(rawFgb.Features[i].PropertiesBytes(), schema)
		if err != nil {
			return nil, nil, fmt.Errorf("feature %d properties: %w", i+1, err)
		}
		props = values
	}
	return props, nil, nil
}
