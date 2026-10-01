package maplibre

import (
	"GeoFlatpack/fgb"
	"GeoFlatpack/style"
	"fmt"
	"image/color"
	"path/filepath"
	"strings"
)

func NewMapLibreStyle(filename, sourceName string, backgroundPaint style.Hex) (*StyleHeader, string) {
	name := styleName(filename)
	if sourceName == "" {
		sourceName = name
	}

	styleHeader := &StyleHeader{
		Version: 8,
		Name:    name,
		Sources: map[string]map[string]any{
			sourceName: {
				"type": "geojson",
				"data": map[string]any{
					"type":     "FeatureCollection",
					"features": []any{},
				},
			},
		},
		Layers: []StyleLayer{
			{
				ID:    "background",
				Type:  "background",
				Paint: Paint{"background-color": backgroundPaint},
			},
		},
	}
	return styleHeader, sourceName
}

// LayerStyle describes one independently parsed FGB and its styling choices.
type LayerStyle struct {
	Data          *fgb.Fgb
	SourceID      string
	CategoryField string
	Paints        map[StyleGroup]Paint
}

func BuildMapLibreStyle(
	data *fgb.Fgb,
	filename, sourceName, field string,
	paints map[StyleGroup]Paint,
) (*StyleHeader, error) {
	if sourceName == "" {
		sourceName = styleName(filename)
	}
	return BuildMapLibreCollectionStyle(filename, []LayerStyle{{
		Data: data, SourceID: sourceName, CategoryField: field, Paints: paints,
	}})
}

// BuildMapLibreCollectionStyle draws polygons, lines, then points, retaining
// input order within each geometry type. Empty inputs still contribute sources.
func BuildMapLibreCollectionStyle(filename string, inputs []LayerStyle) (*StyleHeader, error) {
	backgroundPaint := style.ToHex(color.RGBA{R: 30, G: 30, B: 30, A: 255})
	mapStyle, _ := NewMapLibreStyle(filename, "", backgroundPaint)
	mapStyle.Sources = make(map[string]map[string]any, len(inputs))
	groups := make([][]StyleGroup, len(inputs))

	for i, input := range inputs {
		if input.SourceID == "" {
			return nil, fmt.Errorf("layer %d: empty source ID", i+1)
		}

		if _, exists := mapStyle.Sources[input.SourceID]; exists {
			return nil, fmt.Errorf("duplicate source ID %q", input.SourceID)
		}

		mapStyle.Sources[input.SourceID] = map[string]any{
			"type": "geojson",
			"data": map[string]any{"type": "FeatureCollection", "features": []any{}},
		}

		var err error
		groups[i], err = CollectStyleGroups(input.Data, input.CategoryField)

		if err != nil {
			return nil, fmt.Errorf("source %q: %w", input.SourceID, err)
		}
	}

	for _, geometry := range []GeometryType{Polygon, Line, Point} {
		for i, input := range inputs {
			for j, group := range groups[i] {
				if group.GeometryType != geometry {
					continue
				}

				id := fmt.Sprintf("%s-%s-%d", input.SourceID, geometry, j)
				paint := input.Paints[group]
				var layer StyleLayer
				
				switch geometry {
				case Point:
					layer = PointLayer(id, input.SourceID, input.CategoryField, group, paint)
				case Line:
					layer = LineLayer(id, input.SourceID, input.CategoryField, group, paint)
				case Polygon:
					layer = PolygonLayer(id, input.SourceID, input.CategoryField, group, paint)
				}
				mapStyle.Layers = append(mapStyle.Layers, layer)
			}
		}
	}
	return mapStyle, nil
}

func styleName(filename string) string {
	name := filepath.Base(filename)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if name == "" {
		return "features"
	}
	return name
}
