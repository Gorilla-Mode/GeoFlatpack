package maplibre

import (
	"GeoFlatpack/style"
	"path/filepath"
	"strings"
)

func NewMapLibreStyle(filename, sourceName string, hex style.Hex) (*StyleHeader, string) {
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
				Paint: Paint{"background-color": hex},
			},
		},
	}
	return styleHeader, sourceName
}
