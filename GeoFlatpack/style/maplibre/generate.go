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
				Type:  RenderBackground,
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
		if err := validateStyleSource(input.SourceID, i, mapStyle.Sources); err != nil {
			return nil, err
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
				stack, err := resolveGroupStack(input.Styles, group, id)
				if err != nil {
					return nil, err
				}

				hasLine := false
				for _, layerStyle := range stack {
					if layerStyle.Type == RenderLine {
						hasLine = true
					}
				}

				for k, layerStyle := range stack {
					layer, err := buildRenderLayer(input, group, id, k, layerStyle, hasLine, icons)
					if err != nil {
						return nil, err
					}

					if layerStyle.Type == RenderSymbol {
						selectedIcons[layerStyle.IconName] = icons[layerStyle.IconName].Svg
					}

					placeLayer(layer, &mapStyle.Layers, &symbols[i])
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

func validateStyleSource(sourceID string, index int, sources map[string]map[string]any) error {
	if sourceID == "" {
		return fmt.Errorf("layer %d: empty source ID", index+1)
	}

	if _, exists := sources[sourceID]; exists {
		return fmt.Errorf("duplicate source ID %q", sourceID)
	}

	return nil
}

func resolveGroupStack(styles map[StyleGroup][]RenderLayerStyle, group StyleGroup, id string) ([]RenderLayerStyle, error) {
	stack, configured := styles[group]
	if !configured {
		stack = []RenderLayerStyle{{Type: RenderTypes[group.GeometryType][0]}}
	}

	if len(stack) == 0 {
		return nil, fmt.Errorf("group %q: empty style stack", id)
	}

	return stack, nil
}

func buildRenderLayer(input LayerStyle, group StyleGroup, id string, index int, layerStyle RenderLayerStyle, hasLine bool, icons map[string]svg.Svg) (StyleLayer, error) {
	if layerStyle.Type != RenderSymbol && !slices.Contains(RenderTypes[group.GeometryType], layerStyle.Type) {
		return StyleLayer{}, fmt.Errorf("group %q: unsupported render type %q for %s", id, layerStyle.Type, group.GeometryType)
	}

	layerID := id
	if index > 0 {
		layerID = fmt.Sprintf("%s-%d", id, index)
	}

	var layer StyleLayer
	switch layerStyle.Type {
	case RenderSymbol:
		var err error
		layer, err = buildSymbolLayer(input, group, id, layerID, layerStyle, icons)
		if err != nil {
			return StyleLayer{}, err
		}

	case RenderCircle:
		layer = PointLayer(layerID, input.SourceID, input.CategoryField, group, layerStyle.Paint)

	case RenderLine:
		layer = LineLayer(layerID, input.SourceID, input.CategoryField, group, layerStyle.Paint)

	case RenderFill:
		layer = PolygonLayer(layerID, input.SourceID, input.CategoryField, group, layerStyle.Paint)
		if _, explicit := layerStyle.Paint["fill-outline-color"]; hasLine && !explicit {
			layer.Paint["fill-outline-color"] = "transparent"
		}
	}

	layer.Filter = buildVertexFilter(layer.Filter, group.GeometryType, layerStyle.Type, input.VertexMarker)

	return layer, nil
}

func buildSymbolLayer(input LayerStyle, group StyleGroup, id, layerID string, layerStyle RenderLayerStyle, icons map[string]svg.Svg) (StyleLayer, error) {
	if _, exists := icons[layerStyle.IconName]; !exists {
		return StyleLayer{}, fmt.Errorf("group %q: unknown SVG icon %q", id, layerStyle.IconName)
	}

	if group.GeometryType != Point && input.VertexMarker == "" {
		return StyleLayer{}, fmt.Errorf("group %q: SVG icons require stored vertex companions", id)
	}

	pointGroup := group
	pointGroup.GeometryType = Point
	layer := newLayer(RenderSymbol, layerID, input.SourceID, input.CategoryField, pointGroup, Paint{}, layerStyle.Paint)
	layer.Layout = buildSymbolLayout(layerStyle)

	return layer, nil
}

func buildSymbolLayout(layerStyle RenderLayerStyle) map[string]any {
	layout := map[string]any{"icon-size": 0.5, "icon-allow-overlap": true}
	maps.Copy(layout, layerStyle.Layout)
	layout["icon-image"] = layerStyle.IconName
	layout["symbol-placement"] = "point"

	return layout
}

func buildVertexFilter(filter []any, geometry GeometryType, renderType RenderType, marker string) []any {
	if marker == "" {
		return filter
	}

	if geometry == Point {
		return []any{"all", filter, []any{"!", []any{"has", marker}}}
	}

	if renderType == RenderSymbol {
		return []any{"all", filter, []any{"==", []any{"get", marker}, string(geometry)}}
	}

	return filter
}

func placeLayer(layer StyleLayer, base, symbols *[]StyleLayer) {
	if layer.Type == RenderSymbol {
		*symbols = append(*symbols, layer)

		return
	}

	*base = append(*base, layer)
}

func styleName(filename string) string {
	name := filepath.Base(filename)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if name == "" {
		return "features"
	}

	return name
}
