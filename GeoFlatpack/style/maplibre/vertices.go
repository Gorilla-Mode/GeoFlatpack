package maplibre

import (
	"GeoFlatpack/fgb"
	"fmt"
	"slices"
	"strings"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
)

// PrepareVertexCompanions keeps Data as the original styling input so generated
// MultiPoints never become new style groups. The returned dataset is for output.
func PrepareVertexCompanions(input *LayerStyle) (*fgb.Fgb, error) {
	if input == nil || input.Data == nil || input.Data.Header == nil {
		return nil, fmt.Errorf("missing styling input, FGB, or header")
	}
	selected := make(map[int]string)
	for i := range input.Data.Features {
		feature := &input.Data.Features[i]
		g := feature.Raw.Geometry(&flat.Geometry{})
		if g == nil {
			continue
		}
		typ := g.Type()
		if typ == flat.GeometryTypeUnknown {
			typ = input.Data.Header.GeometryType()
		}
		group := StyleGroup{GeometryType: GeometryType(strings.TrimPrefix(typ.String(), "Multi"))}
		if group.GeometryType != Line && group.GeometryType != Polygon {
			continue
		}
		if input.CategoryField != "" {
			category, err := NewCategoryValue(feature.Properties[input.CategoryField])
			if err != nil {
				return nil, fmt.Errorf("feature %d: %w", i+1, err)
			}
			group.Category = category
		}
		if slices.ContainsFunc(input.Styles[group], func(layer RenderLayerStyle) bool { return layer.Type == "symbol" }) {
			selected[i] = string(group.GeometryType)
		}
	}
	data, marker, err := fgb.WithVertexCompanions(input.Data, selected)
	if err != nil {
		return nil, err
	}
	input.VertexMarker = marker
	return data, nil
}
