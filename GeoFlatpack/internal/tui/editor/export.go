package editor

import (
	"GeoFlatpack/internal/app"
	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/style/maplibre"
	"fmt"
)

func (m *Editor) Ready() bool {
	if !m.loaded || len(m.layers) == 0 {
		return false
	}

	for _, layer := range m.layers {
		if layer.Status != panel.ListComplete {
			return false
		}
	}

	return true
}

// Snapshot source/category/group order independently of visible cursors. Stack
// order is insertion order, which the shared writer translates to draw order.
func (m *Editor) Selections() ([]app.StyleSelection, error) {
	if !m.loaded || len(m.layers) != len(m.categories) || len(m.categories) != len(m.sources) {
		return nil, fmt.Errorf("style selections unavailable")
	}

	selections := make([]app.StyleSelection, len(m.categories))
	for i, categories := range m.categories {
		if categories.active < 0 || categories.active >= len(categories.features) || categories.active >= len(categories.fields) {
			return nil, fmt.Errorf("layer %q has no active category", m.layers[i].Name)
		}

		features := categories.features[categories.active]
		if len(features.groups) != len(features.styling) {
			return nil, fmt.Errorf("layer %q has incomplete styling groups", m.layers[i].Name)
		}

		selection := app.StyleSelection{CategoryField: categories.fields[categories.active], Styles: make(map[maplibre.StyleGroup][]maplibre.RenderLayerStyle)}
		for j, group := range features.groups {
			state := features.styling[j]
			if state.status() != panel.ListComplete {
				return nil, fmt.Errorf("layer %q has incomplete styling", m.layers[i].Name)
			}

			stack := make([]maplibre.RenderLayerStyle, 0, len(state.layers))
			for _, layer := range state.layers {
				render, err := layer.renderSnapshot(false)
				if err != nil {
					return nil, fmt.Errorf("copy layer %q styling: %w", m.layers[i].Name, err)
				}

				stack = append(stack, render)
			}

			selection.Styles[group] = stack
		}

		selections[i] = selection
	}

	return selections, nil
}
