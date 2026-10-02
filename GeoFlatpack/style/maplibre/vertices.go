package maplibre

import (
	"GeoFlatpack/fgb"
	"fmt"
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
		group, eligible, err := vertexStyleGroup(&input.Data.Features[i], input.Data.Header.GeometryType(), input.CategoryField, i)
		if err != nil {
			return nil, err
		}

		if !eligible {
			continue
		}

		for _, layer := range input.Styles[group] {
			if layer.Type == "symbol" {
				selected[i] = string(group.GeometryType)
				break
			}
		}
	}

	data, marker, err := fgb.WithVertexCompanions(input.Data, selected)
	if err != nil {
		return nil, err
	}

	input.VertexMarker = marker

	return data, nil
}

func vertexStyleGroup(feature *fgb.Feature, fallback flat.GeometryType, field string, index int) (StyleGroup, bool, error) {
	typ, present := featureGeometryType(feature, fallback)
	if !present {
		return StyleGroup{}, false, nil
	}

	group := StyleGroup{GeometryType: GeometryType(strings.TrimPrefix(typ.String(), "Multi"))}
	if group.GeometryType != Line && group.GeometryType != Polygon {
		return StyleGroup{}, false, nil
	}

	if field != "" {
		category, err := NewCategoryValue(feature.Properties[field])
		if err != nil {
			return StyleGroup{}, false, fmt.Errorf("feature %d: %w", index+1, err)
		}

		group.Category = category
	}

	return group, true, nil
}
