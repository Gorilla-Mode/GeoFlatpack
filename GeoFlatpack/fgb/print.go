package fgb

import (
	"fmt"
	"strings"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
)

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

		printGeometry(feature, b)

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

func printGeometry(feature *Feature, b strings.Builder) {
	if geometry := feature.Raw.Geometry(&flat.Geometry{}); geometry != nil {
		_, _ = fmt.Fprintf(&b, "\tGeometry Type: %s\n", geometry.Type())

		for j := 0; j+1 < geometry.XyLength(); j += 2 {
			_, _ = fmt.Fprintf(&b, "\tX: %g Y: %g\n",
				geometry.Xy(j), geometry.Xy(j+1))
		}
	}
}
