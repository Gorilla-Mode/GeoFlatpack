package maplibre

import (
	"GeoFlatpack/fgb"
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
)

func newLayer(
	layerType, id, source, field string,
	group StyleGroup,
	defaults, overrides Paint,
) StyleLayer {
	maps.Copy(defaults, overrides)

	filter := []any{"==", []any{"geometry-type"}, group.GeometryType}
	if field != "" {
		filter = []any{
			"all",
			filter,
			// A missing property evaluates to null, combining missing and null values.
			[]any{"==", []any{"get", field}, group.Category.FilterValue()},
		}
	}

	return StyleLayer{
		ID: id, Type: layerType, Source: source,
		Filter: filter, Paint: defaults,
	}
}

func PointLayer(id, source, field string, group StyleGroup, paint Paint) StyleLayer {
	return newLayer("circle", id, source, field, group, Paint{
		"circle-color":        "#f59e0b",
		"circle-radius":       6,
		"circle-stroke-color": "#78350f",
		"circle-stroke-width": 1,
	}, paint)
}

func LineLayer(id, source, field string, group StyleGroup, paint Paint) StyleLayer {
	return newLayer("line", id, source, field, group, Paint{
		"line-color": "#dc2626",
		"line-width": 3,
	}, paint)
}

func PolygonLayer(id, source, field string, group StyleGroup, paint Paint) StyleLayer {
	return newLayer("fill", id, source, field, group, Paint{
		"fill-color":         "#2563eb",
		"fill-opacity":       0.55,
		"fill-outline-color": "#1e3a8a",
	}, paint)
}

func CollectStyleGroups(fgb *fgb.Fgb, field string) ([]StyleGroup, error) {
	if fgb == nil || fgb.Header == nil {
		return nil, fmt.Errorf("missing FGB or header")
	}

	seen := make(map[StyleGroup]bool)

	for i := range fgb.Features {
		feature := &fgb.Features[i]
		geometry := feature.Raw.Geometry(&flat.Geometry{})
		if geometry == nil {
			continue
		}

		geometryType := geometry.Type()
		if geometryType == flat.GeometryTypeUnknown {
			geometryType = fgb.Header.GeometryType()
		}

		group := StyleGroup{
			GeometryType: GeometryType(strings.TrimPrefix(geometryType.String(), "Multi")),
		}
		switch group.GeometryType {
		case Point, Line, Polygon:
		default:
			return nil, fmt.Errorf(
				"feature %d: unsupported geometry %s", i+1, geometryType,
			)
		}

		if field != "" {
			category, err := NewCategoryValue(feature.Properties[field])
			if err != nil {
				return nil, fmt.Errorf("feature %d, field %q: %w", i+1, field, err)
			}
			group.Category = category
		}
		seen[group] = true
	}

	groups := make([]StyleGroup, 0, len(seen))
	for group := range seen {
		groups = append(groups, group)
	}

	// Draw polygons first, then lines, then points.
	order := map[string]int{string(Polygon): 0, string(Line): 1, string(Point): 2}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].GeometryType != groups[j].GeometryType {
			return order[string(groups[i].GeometryType)] < order[string(groups[j].GeometryType)]
		}
		return categoryLess(groups[i].Category, groups[j].Category)
	})

	return groups, nil
}
