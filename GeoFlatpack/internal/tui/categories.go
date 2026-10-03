package tui

import (
	"GeoFlatpack/fgb"
	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/style/maplibre"
	"fmt"
)

// categoryState belongs to one source layer. Cursor and active choice are
// independent: browsing a field never commits it as the styling category.
type categoryState struct {
	items        []panel.ListItem
	fields       []string
	selected     int
	firstVisible int
	active       int
	emptyText    string
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
	state.add("", "No category field", len(groups), examples)
	for _, field := range fields {
		if field.Unavailable != "" {
			continue
		}
		groups, err := maplibre.CollectStyleGroups(data, field.Name)
		if err != nil {
			continue
		}
		state.add(field.Name, field.Name, len(groups), field.Examples)
	}
	return state
}

func (s *categoryState) add(field, name string, count int, examples []string) {
	detail := fmt.Sprintf("%d items to style", count)
	if count == 1 {
		detail = "1 item to style"
	}
	s.fields = append(s.fields, field)
	s.items = append(s.items, panel.ListItem{Name: name, Detail: detail, Children: append([]string(nil), examples[:min(3, len(examples))]...)})
}

func (m *Model) currentCategories() *categoryState {
	if m.selectedLayer < 0 || m.selectedLayer >= len(m.categories) {
		return nil
	}
	return &m.categories[m.selectedLayer]
}

func (s *categoryState) activate() {
	if s.active >= 0 {
		s.items[s.active].Status = panel.ListUnopened
	}
	s.active = s.selected
	s.items[s.active].Status = panel.ListActive
}
