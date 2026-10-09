package editor

import (
	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/style/maplibre"
	"GeoFlatpack/style/maplibre/svg"
	"fmt"
)

// Target is an independent style snapshot; source geometry and icons are
// immutable session inputs. Browsing does not change the committed edit target.
type Target struct {
	Input                    maplibre.LayerStyle
	Icons                    map[string]svg.Svg
	Group                    *maplibre.StyleGroup
	Layer, Category, Feature int
}

func (m *Editor) PreviewTarget() (Target, error) {
	layers := m.sources
	if m.selectedLayer < 0 || m.selectedLayer >= len(layers) {
		return Target{}, fmt.Errorf("No layers loaded")
	}

	layer := layers[m.selectedLayer]
	request := Target{Input: maplibre.LayerStyle{Data: layer.Data, SourceID: layer.SourceID, Styles: make(map[maplibre.StyleGroup][]maplibre.RenderLayerStyle)}, Icons: m.icons}
	if request.Input.SourceID == "" {
		request.Input.SourceID = "preview-source"
	}

	categories, features := m.currentCategories(), m.currentFeatures()
	if categories == nil || features == nil {
		return request, fmt.Errorf("Choose a category to preview")
	}

	if len(features.groups) == 0 {
		return request, fmt.Errorf("No geometry to preview")
	}

	groupIndex := features.chosen
	if m.activePane == panel.FeaturesPane || ((m.activePane == panel.LayerPane || m.activePane == panel.CategoryPane) && (groupIndex < 0 || groupIndex >= len(features.groups))) {
		groupIndex = features.selected
	}

	if groupIndex < 0 || groupIndex >= len(features.groups) {
		return request, fmt.Errorf("Choose a feature to preview")
	}

	categoryIndex := categories.active
	request.Input.CategoryField = categories.fields[categoryIndex]
	group := features.groups[groupIndex]
	request.Group = &group
	request.Layer, request.Category, request.Feature = m.selectedLayer, categoryIndex, groupIndex
	var stack []maplibre.RenderLayerStyle
	for _, layer := range features.styling[groupIndex].layers {
		if layer.style.Type == maplibre.RenderSymbol && layer.style.IconName == "" {
			continue
		}

		render, err := layer.renderSnapshot(true)
		if err != nil {
			return request, err
		}

		stack = append(stack, render)
	}

	if len(stack) > 0 {
		request.Input.Styles[group] = stack
	}

	return request, nil
}
