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
	name := filepath.Base(filename)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	if name == "" {
		name = "features"
	}
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

func BuildMapLibreStyle(
	fgb *fgb.Fgb,
	filename, sourceName, field string,
	paints map[StyleGroup]Paint,
) (*StyleHeader, error) {
	groups, err := CollectStyleGroups(fgb, field)
	if err != nil {
		return nil, err
	}

	backgroundPaint := style.ToHex(color.RGBA{R: 30, G: 30, B: 30, A: 255})
	mapStyle, source := NewMapLibreStyle(filename, sourceName, backgroundPaint)

	for i, group := range groups {
		id := fmt.Sprintf("%s-%s-%d", source, group.GeometryType, i)
		paint := paints[group]

		var layer StyleLayer
		switch group.GeometryType {
		case Point:
			layer = PointLayer(id, source, field, group, paint)
		case Line:
			layer = LineLayer(id, source, field, group, paint)
		case Polygon:
			layer = PolygonLayer(id, source, field, group, paint)
		}
		mapStyle.Layers = append(mapStyle.Layers, layer)
	}

	return mapStyle, nil
}
