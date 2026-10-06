package preview

import (
	"fmt"
	"math"
	"strings"

	"GeoFlatpack/fgb"
	"GeoFlatpack/style/maplibre"
	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
)

// RepresentativeIndex uses loaded source order, before serialization sorts the
// spatial index. A multipart geometry remains a single original feature.
func RepresentativeIndex(data *fgb.Fgb, field string, group maplibre.StyleGroup) (int, error) {
	if data != nil && data.Header != nil {
		for i := range data.Features {
			feature := &data.Features[i]
			geometry := feature.Raw.Geometry(&flat.Geometry{})
			if geometry == nil {
				continue
			}
			typ := geometry.Type()
			if typ == flat.GeometryTypeUnknown {
				typ = data.Header.GeometryType()
			}
			if maplibre.GeometryType(strings.TrimPrefix(typ.String(), "Multi")) != group.GeometryType {
				continue
			}
			if field != "" {
				category, err := maplibre.NewCategoryValue(feature.Properties[field])
				if err != nil {
					return -1, err
				}
				if category != group.Category {
					continue
				}
			}
			if _, present := geometryBounds(geometry); present {
				return i, nil
			}
		}
	}
	return -1, fmt.Errorf("No geometry to preview")
}

func geometryBounds(geometry *flat.Geometry) ([4]float64, bool) {
	bounds := [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	var visit func(*flat.Geometry)
	visit = func(g *flat.Geometry) {
		for i := 0; i+1 < g.XyLength(); i += 2 {
			x, y := g.Xy(i), g.Xy(i+1)
			if math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) {
				continue
			}
			bounds[0], bounds[1] = min(bounds[0], x), min(bounds[1], y)
			bounds[2], bounds[3] = max(bounds[2], x), max(bounds[3], y)
		}
		for i := 0; i < g.PartsLength(); i++ {
			var part flat.Geometry
			if g.Parts(&part, i) {
				visit(&part)
			}
		}
	}
	visit(geometry)
	return bounds, !math.IsInf(bounds[0], 1)
}

func representativeData(data *fgb.Fgb, index int) (*fgb.Fgb, error) {
	// Clone the FlatBuffer before changing its count and extent. Feature buffers
	// and properties stay read-only; companion generation allocates its own data.
	table := data.Header.Table()
	header := &flat.Header{}
	header.Init(append([]byte(nil), table.Bytes...), table.Pos)
	if !header.MutateFeaturesCount(1) {
		return nil, fmt.Errorf("Preview unavailable: source header has no feature count")
	}
	geometry := data.Features[index].Raw.Geometry(&flat.Geometry{})
	if bounds, present := geometryBounds(geometry); present && header.EnvelopeLength() >= 4 {
		for i, value := range bounds {
			header.MutateEnvelope(i, value)
		}
	}
	return &fgb.Fgb{Header: header, Features: []fgb.Feature{data.Features[index]}}, nil
}
