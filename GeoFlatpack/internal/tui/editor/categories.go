package editor

import (
	"GeoFlatpack/fgb"
	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/style/maplibre"
	"fmt"
)

// categoryState belongs to one source layer. Cursor and active choice are
// independent: browsing a field never commits it as the styling category.
type categoryState struct {
	items            []panel.ListItem
	fields           []string
	features         []featureState
	selected         int
	firstVisible     int
	active           int
	emptyText        string
	conditional      []*conditionState
	markedOnly       bool
	collectionActive bool
	collection       *collectionState
	collections      map[string]*collectionState
}

func newCategoryState(data *fgb.Fgb) categoryState {
	state := categoryState{active: -1, emptyText: "No category fields available"}
	if data == nil || data.Header == nil || len(data.Header.Table().Bytes) == 0 {
		state.emptyText = "Category fields unavailable: missing data or header"

		return state
	}

	for i := range data.Features {
		if len(data.Features[i].Raw.Table().Bytes) == 0 {
			state.emptyText = "Category fields unavailable: missing feature data"

			return state
		}
	}

	fields, err := maplibre.DiscoverCategoryFields(data)
	if err != nil {
		state.emptyText = fmt.Sprintf("Category fields unavailable: %v", err)

		return state
	}

	groups, err := maplibre.CollectStyleGroups(data, "")
	if err != nil {
		state.emptyText = fmt.Sprintf("Category fields unavailable: %v", err)

		return state
	}

	var examples []string
	for _, group := range groups {
		examples = append(examples, string(group.GeometryType))
	}

	state.add("", "No category field", len(groups), examples, groups)
	for _, field := range fields {
		if field.Unavailable != "" {
			continue
		}

		groups, err := maplibre.CollectStyleGroups(data, field.Name)
		if err != nil {
			continue
		}

		state.add(field.Name, field.Name, field.Distinct, field.Examples, groups)
		state.conditional[len(state.conditional)-1].valueType = inferConditionType(field.Types)
	}

	return state
}

func (s *categoryState) add(field, name string, count int, examples []string, groups []maplibre.StyleGroup) {
	detail := fmt.Sprintf("%d items to style", count)
	if count == 1 {
		detail = "1 item to style"
	}

	s.fields = append(s.fields, field)
	s.features = append(s.features, newFeatureState(groups))
	s.conditional = append(s.conditional, &conditionState{geometries: append([]maplibre.StyleGroup(nil), s.features[0].groups...), valueType: "number"})
	s.items = append(s.items, panel.ListItem{Name: name, Detail: detail, Children: append([]string(nil), examples[:min(3, len(examples))]...)})
}

func (s *categoryState) visibleIndices() []int {
	var indices []int
	for i, item := range s.items {
		if !s.markedOnly || item.Conditional {
			indices = append(indices, i)
		}
	}
	return indices
}

func (s *categoryState) visibleItems() ([]panel.ListItem, int) {
	var items []panel.ListItem
	selected := 0
	for _, index := range s.visibleIndices() {
		if index == s.selected {
			selected = len(items)
		}
		item := s.items[index]
		if item.Conditional {
			count := len(s.conditional[index].rules)
			item.Detail = fmt.Sprintf("%d conditions", count)
			if count == 1 {
				item.Detail = "1 condition"
			}
		}
		items = append(items, item)
	}
	return items, selected
}

func (s *categoryState) move(delta int) {
	indices := s.visibleIndices()
	if len(indices) == 0 {
		return
	}
	_, selected := s.visibleItems()
	s.selected = indices[panel.MoveListSelection(selected, delta, len(indices))]
}

func (s *categoryState) toggleFilter() {
	s.markedOnly = !s.markedOnly
	s.firstVisible = 0
	if s.markedOnly && !s.items[s.selected].Conditional {
		indices := s.visibleIndices()
		if len(indices) > 0 {
			s.selected = indices[0]
		}
	}
}

func (m *Editor) currentCategories() *categoryState {
	if m.selectedLayer < 0 || m.selectedLayer >= len(m.categories) {
		return nil
	}

	return &m.categories[m.selectedLayer]
}

func (s *categoryState) activate() {
	s.active = s.selected
	s.collectionActive = s.items[s.selected].Conditional
	if s.collectionActive {
		s.rebuildCollection(s.selected)
	}
	s.refreshActiveIndicators()
}
