package maplibre

import (
	"GeoFlatpack/fgb"
	"GeoFlatpack/style"
	"GeoFlatpack/style/maplibre/svg"
	"fmt"
	"image/color"
	"maps"
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
	VertexMarker  string
}

// BuildMapLibreCollectionStyle draws polygons, lines, then points, retaining
// input order within each geometry type, then appends SVG symbols in prompting
// order (input, group, selection). Empty inputs still contribute sources.
func BuildMapLibreCollectionStyle(filename string, inputs []LayerStyle, catalogs ...map[string]svg.Svg) (*StyleHeader, error) {
	backgroundPaint := style.ToHex(color.RGBA{R: 30, G: 30, B: 30, A: 255})
	mapStyle, _ := NewMapLibreStyle(filename, "", backgroundPaint)
	mapStyle.Sources = make(map[string]map[string]any, len(inputs))
	groups := make([][]StyleGroup, len(inputs))
	var icons map[string]svg.Svg
	if len(catalogs) > 0 {
		icons = catalogs[0]
	}
	selectedIcons := make(map[string]string)
	symbols := make([][]StyleLayer, len(inputs))

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
					if layerStyle.Type != "symbol" && !slices.Contains(RenderTypes[geometry], layerStyle.Type) {
						return nil, fmt.Errorf("group %q: unsupported render type %q for %s", id, layerStyle.Type, geometry)
					}
					layerID := id
					if k > 0 {
						layerID = fmt.Sprintf("%s-%d", id, k)
					}
					var layer StyleLayer

					switch layerStyle.Type {
					case "symbol":
						icon, exists := icons[layerStyle.IconName]
						if !exists {
							return nil, fmt.Errorf("group %q: unknown SVG icon %q", id, layerStyle.IconName)
						}
						if geometry != Point && input.VertexMarker == "" {
							return nil, fmt.Errorf("group %q: SVG icons require stored vertex companions", id)
						}
						pointGroup := group
						pointGroup.GeometryType = Point
						layer = newLayer("symbol", layerID, input.SourceID, input.CategoryField, pointGroup, Paint{}, layerStyle.Paint)
						layer.Layout = map[string]any{"icon-size": 0.5, "icon-allow-overlap": true}
						maps.Copy(layer.Layout, layerStyle.Layout)
						layer.Layout["icon-image"] = layerStyle.IconName
						layer.Layout["symbol-placement"] = "point"
						if geometry != Point {
							layer.Filter = []any{"all", layer.Filter, []any{"==", []any{"get", input.VertexMarker}, string(geometry)}}
						}
						selectedIcons[layerStyle.IconName] = icon.Svg
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

					if geometry == Point && input.VertexMarker != "" {
						layer.Filter = []any{"all", layer.Filter, []any{"!", []any{"has", input.VertexMarker}}}
					}
					if layerStyle.Type == "symbol" {
						symbols[i] = append(symbols[i], layer)
					} else {
						mapStyle.Layers = append(mapStyle.Layers, layer)
					}
				}
			}
		}
	}
	for _, stack := range symbols {
		mapStyle.Layers = append(mapStyle.Layers, stack...)
	}
	if len(selectedIcons) > 0 {
		mapStyle.Metadata = map[string]any{"geoflatpack:icons": selectedIcons}
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
