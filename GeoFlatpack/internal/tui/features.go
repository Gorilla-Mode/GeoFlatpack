package tui

import (
	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/style/maplibre"
)

// Each category choice retains its styling groups and independent navigation.
type featureState struct {
	items        []panel.ListItem
	groups       []maplibre.StyleGroup
	styling      []stylingState
	selected     int
	firstVisible int
	chosen       int
}

func newFeatureState(groups []maplibre.StyleGroup) featureState {
	state := featureState{groups: groups, chosen: -1}
	for _, group := range groups {
		state.styling = append(state.styling, stylingState{active: -1})
		state.items = append(state.items, panel.ListItem{
			Name: group.Category.String(), Children: []string{"Type: " + string(group.GeometryType)},
		})
	}
	return state
}

func (m *Model) currentFeatures() *featureState {
	categories := m.currentCategories()
	if categories == nil || categories.active < 0 || categories.active >= len(categories.features) {
		return nil
	}
	return &categories.features[categories.active]
}

func (s *featureState) choose() {
	s.chosen = s.selected
	s.items[s.chosen].Status = panel.ListIncomplete
}
