package fgb

import (
	"errors"
	"fmt"
	"io"

	"github.com/gogama/flatgeobuf/flatgeobuf"
	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
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
		props, err := readFeatureProperties(&rawFgb.Features[i], rawFgb.header)
		if err != nil {
			return nil, fmt.Errorf("feature %d properties: %w", i+1, err)
		}

		fgb.Features = append(fgb.Features, Feature{
			Raw:        rawFgb.Features[i],
			Properties: props,
		})
	}

	return fgb, nil
}

func readFeatureProperties(feature *flat.Feature, schema flatgeobuf.Schema) (map[string]any, error) {
	if feature.PropertiesLength() == 0 {
		return make(map[string]any), nil
	}

	if feature.ColumnsLength() > 0 {
		schema = feature
	}

	return readProperties(feature.PropertiesBytes(), schema)
}
