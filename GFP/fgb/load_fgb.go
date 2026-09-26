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

type RawFgb struct {
	header   *flat.Header
	Features []flat.Feature
}

type Fgb struct {
	Header   *flat.Header
	Features []Feature
}

type Feature struct {
	Raw        flat.Feature
	Properties map[string]any
}

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

		if rawFgb.Features[i].PropertiesLength() > 0 {
			var schema flatgeobuf.Schema = rawFgb.header
			if rawFgb.Features[i].ColumnsLength() > 0 {
				schema = &rawFgb.Features[i]
			}

			values, err := flatgeobuf.NewPropReader(
				bytes.NewReader(rawFgb.Features[i].PropertiesBytes()),
			).ReadSchema(schema)
			if err != nil {
				return nil, fmt.Errorf("feature %d properties: %w", i, err)
			}

			for _, value := range values {
				props[string(value.Col.Name())] = value.Value
			}
		}

		fgb.Features = append(fgb.Features, Feature{
			Raw:        rawFgb.Features[i],
			Properties: props,
		})
	}

	return fgb, nil
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

func InspectFgb(fgb *Fgb) string {
	var b strings.Builder
	b.WriteString(PrintHeader(fgb.Header))

	for i := range fgb.Features {
		feature := &fgb.Features[i]
		_, _ = fmt.Fprintf(&b, "\nFeature %d\n", i+1)

		if geometry := feature.Raw.Geometry(&flat.Geometry{}); geometry != nil {
			_, _ = fmt.Fprintf(&b, "\tGeometry Type: %s\n", geometry.Type())

			for j := 0; j+1 < geometry.XyLength(); j += 2 {
				_, _ = fmt.Fprintf(&b, "\tX: %g Y: %g\n",
					geometry.Xy(j), geometry.Xy(j+1))
			}
		}

		names := make([]string, 0, len(feature.Properties))
		for name := range feature.Properties {
			names = append(names, name)
		}

		for _, name := range names {
			_, _ = fmt.Fprintf(&b, "\t%s: %v\n",
				name, feature.Properties[name])
		}
	}

	return b.String()
}
