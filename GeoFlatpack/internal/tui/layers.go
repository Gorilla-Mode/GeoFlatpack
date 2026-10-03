package tui

import (
	"GeoFlatpack/internal/app"
	"GeoFlatpack/internal/tui/panel"
	"fmt"
)

func layerItems(layers []app.Layer) []panel.ListItem {
	items := make([]panel.ListItem, len(layers))
	for i, layer := range layers {
		count := 0
		if layer.Data != nil {
			count = len(layer.Data.Features)
		}
		detail := fmt.Sprintf("%d features", count)
		if count == 1 {
			detail = "1 feature"
		}
		items[i] = panel.ListItem{Name: layer.Name, Detail: detail}
	}
	return items
}
