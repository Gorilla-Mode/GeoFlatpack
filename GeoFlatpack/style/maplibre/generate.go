package maplibre

import (
	"GeoFlatpack/fgb"
	"GeoFlatpack/style"
	"fmt"
	"image/color"
	"path/filepath"
	"slices"
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
	Styles        map[StyleGroup][]RenderLayerStyle
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
				stack, configured := input.Styles[group]
				if !configured {
					stack = []RenderLayerStyle{{Type: RenderTypes[geometry][0]}}
				}
				if len(stack) == 0 {
					return nil, fmt.Errorf("group %q: empty style stack", id)
				}
				hasLine := slices.ContainsFunc(stack, func(style RenderLayerStyle) bool {
					return style.Type == "line"
				})
				for k, layerStyle := range stack {
					if !slices.Contains(RenderTypes[geometry], layerStyle.Type) {
						return nil, fmt.Errorf("group %q: unsupported render type %q for %s", id, layerStyle.Type, geometry)
					}
					layerID := id
					if k > 0 {
						layerID = fmt.Sprintf("%s-%d", id, k)
					}
					var layer StyleLayer

					switch layerStyle.Type {
					case "circle":
						layer = PointLayer(layerID, input.SourceID, input.CategoryField, group, layerStyle.Paint)
					case "line":
						layer = LineLayer(layerID, input.SourceID, input.CategoryField, group, layerStyle.Paint)
					case "fill":
						layer = PolygonLayer(layerID, input.SourceID, input.CategoryField, group, layerStyle.Paint)

						if _, explicit := layerStyle.Paint["fill-outline-color"]; hasLine && !explicit {
							layer.Paint["fill-outline-color"] = "transparent"
						}
					}

					mapStyle.Layers = append(mapStyle.Layers, layer)
				}
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
