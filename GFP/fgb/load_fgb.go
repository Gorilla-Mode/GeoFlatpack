package fgb

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

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

func PrintHeader(header *flat.Header) string {
	var b strings.Builder

	h := header
	_, _ = fmt.Fprintf(&b, "FGB Header:\n")
	_, _ = fmt.Fprintf(&b, "\tName: %s\n", h.Name())
	_, _ = fmt.Fprintf(&b, "\tGeometry type: %s\n", h.GeometryType())
	_, _ = fmt.Fprintf(&b, "\tFeature count: %d\n", h.FeaturesCount())
	_, _ = fmt.Fprintf(&b, "\tHas Z: %t\n", h.HasZ())
	_, _ = fmt.Fprintf(&b, "\tIndex node size: %d\n", h.IndexNodeSize())

	if h.EnvelopeLength() == 4 {
		_, _ = fmt.Fprintf(&b, "\n\tBounds: (%g, %g) to (%g, %g)\n",
			h.Envelope(0), h.Envelope(1),
			h.Envelope(2), h.Envelope(3))
	}

	var crs flat.Crs
	if h.Crs(&crs) != nil {
		_, _ = fmt.Fprintf(&b, "\tCRS: %s:%d\n\n", crs.Org(), crs.Code())
	}

	for i := 0; i < h.ColumnsLength(); i++ {
		var column flat.Column
		if h.Columns(&column, i) {
			_, _ = fmt.Fprintf(&b, "\tField: %s (%s)\n", column.Name(), column.Type())
		}
	}

	return b.String()
}

func InspectFgb(fgb *FGB) string {
	var b strings.Builder

	b.WriteString(PrintHeader(fgb.header))

	for i := range fgb.Features {
		geometry := fgb.Features[i].Geometry(&flat.Geometry{})
		if geometry == nil {
			continue
		}

		_, _ = fmt.Fprintf(&b, "\nFeature %d\n\tGeometry Type: %s\n",
			i+1, geometry.Type())

		props, err := FeatureProperties(fgb, &fgb.Features[i])
		if err != nil {
			return ""
		}

		_, _ = fmt.Fprintf(&b, "\tkind: %v\n\tname: %v\n\theight: %v\n",
			props["kind"], props["name"], props["height_m"])

		for j := 0; j+1 < geometry.XyLength(); j += 2 {
			_, _ = fmt.Fprintf(&b, "\tX: %g Y: %g\n",
				geometry.Xy(j), geometry.Xy(j+1))
		}
	}

	return b.String()
}

func FeatureProperties(fgb *FGB, feature *flat.Feature) (map[string]any, error) {
	properties := make(map[string]any)

	if feature.PropertiesLength() == 0 {
		return properties, nil
	}

	schema := flatgeobuf.Schema(fgb.header)
	if feature.ColumnsLength() > 0 {
		schema = feature
	}

	reader := flatgeobuf.NewPropReader(
		bytes.NewReader(feature.PropertiesBytes()),
	)
	values, err := reader.ReadSchema(schema)
	if err != nil {
		return nil, fmt.Errorf("decode FGB properties: %w", err)
	}

	for _, property := range values {
		properties[string(property.Col.Name())] = property.Value
	}
	return properties, nil
}