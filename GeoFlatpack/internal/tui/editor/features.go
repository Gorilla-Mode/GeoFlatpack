package editor

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

func (m *Editor) currentFeatures() *featureState {
	categories := m.currentCategories()
	if categories == nil || categories.active < 0 || categories.active >= len(categories.features) {
		return nil
	}

	return &categories.features[categories.active]
}

func (s *featureState) choose() {
	s.chosen = s.selected
	if s.items[s.chosen].Status == panel.ListUnopened {
		s.items[s.chosen].Status = panel.ListIncomplete
	}
}

// RefreshReadiness preserves unopened targets while deriving completed states
// from their full stack, independently of the current navigation selection.
func (s *featureState) RefreshReadiness() bool {
	complete := len(s.items) > 0 && len(s.items) == len(s.styling)
	for i := range s.items {
		status := panel.ListUnopened
		if i < len(s.styling) {
			status = s.styling[i].status()
		}

		if status == panel.ListUnopened && s.items[i].Status != panel.ListUnopened {
			status = panel.ListIncomplete
		}

		s.items[i].Status = status
		complete = complete && status == panel.ListComplete
	}

	return complete
}

func (m *Editor) RefreshReadiness() {
	for i := range m.categories {
		categories := &m.categories[i]
		complete := false
		for j := range categories.features {
			ready := categories.features[j].RefreshReadiness()
			if j == categories.active {
				complete = ready
			}
		}

		if i >= len(m.layers) {
			continue
		}

		if complete {
			m.layers[i].Status = panel.ListComplete
		} else if categories.active >= 0 || m.layers[i].Status != panel.ListUnopened {
			m.layers[i].Status = panel.ListIncomplete
		}
	}
}
