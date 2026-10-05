package tui

import (
	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/style/maplibre"
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

type stylingMode int

const (
	styleTypes stylingMode = iota
	styleSelection
	styleEdit
	styleIcons
)

type styleProperty struct {
	name    string
	section maplibre.StyleSection
	spec    maplibre.PropertySpec
}

type stylingCatalog struct {
	properties map[maplibre.RenderType][]styleProperty
	err        error
}

// The embedded reference and its expanded transition properties are immutable.
var cachedStylingCatalog = sync.OnceValue(func() stylingCatalog {
	catalog := stylingCatalog{properties: make(map[maplibre.RenderType][]styleProperty)}
	spec, err := maplibre.LoadSpec()
	if err != nil {
		catalog.err = err
		return catalog
	}
	for _, kind := range []maplibre.RenderType{maplibre.RenderFill, maplibre.RenderLine, maplibre.RenderCircle} {
		properties, err := spec.Properties(kind, maplibre.PaintSection)
		if err != nil {
			catalog.err = err
			return catalog
		}
		for name, property := range properties {
			catalog.properties[kind] = append(catalog.properties[kind], styleProperty{name, maplibre.PaintSection, property})
		}
		sort.Slice(catalog.properties[kind], func(i, j int) bool { return catalog.properties[kind][i].name < catalog.properties[kind][j].name })
	}
	return catalog
})

// Inclusion and validated values are drafts, separate from output maps.
type optionDraft struct {
	included      bool
	value         any
	hasValue      bool
	validationErr error
}

func (d optionDraft) status() panel.ListStatus {
	if !d.included {
		return panel.ListUnopened
	}
	if d.hasValue && d.validationErr == nil {
		if _, err := json.Marshal(d.value); err == nil {
			return panel.ListComplete
		}
	}
	return panel.ListIncomplete
}

type styleLayerState struct {
	style                  maplibre.RenderLayerStyle
	properties             []styleProperty
	options                map[string]optionDraft
	selected, firstVisible int
	includedOnly           bool
	controls               map[string]*controlState
}

type stylingState struct {
	layers                         []*styleLayerState // insertion order; SVG layers have higher draw order
	active                         int
	mode                           stylingMode
	selected, firstVisible         int
	typeSelected, typeFirstVisible int
	iconSelected, iconFirstVisible int
	stackFirstVisible              int
}

// status is shared by stack indicators and the parent feature's readiness.
func (l *styleLayerState) status() panel.ListStatus {
	included := 0
	for _, draft := range l.options {
		if !draft.included {
			continue
		}
		included++
		if draft.status() != panel.ListComplete {
			return panel.ListIncomplete
		}
	}
	if l.style.Type == maplibre.RenderSymbol {
		if l.style.IconName == "" || l.style.Layout["icon-image"] != l.style.IconName {
			return panel.ListIncomplete
		}
		return panel.ListComplete
	}
	if included == 0 {
		return panel.ListUnopened
	}
	return panel.ListComplete
}

func (s *stylingState) status() panel.ListStatus {
	if len(s.layers) == 0 {
		return panel.ListUnopened
	}
	for _, layer := range s.layers {
		if layer.status() != panel.ListComplete {
			return panel.ListIncomplete
		}
	}
	return panel.ListComplete
}

func (m *Model) currentStyling() *stylingState {
	features := m.currentFeatures()
	if features == nil || features.chosen < 0 || features.chosen >= len(features.styling) {
		return nil
	}
	return &features.styling[features.chosen]
}

func (s *stylingState) activeLayer() *styleLayerState {
	if s.active < 0 || s.active >= len(s.layers) {
		return nil
	}
	return s.layers[s.active]
}

func renderTypeLabel(kind maplibre.RenderType) string {
	if kind == maplibre.RenderSymbol {
		return "SVG icon"
	}
	return string(kind)
}

func (m *Model) styleTypes() []maplibre.RenderType {
	features := m.currentFeatures()
	if features == nil || features.chosen < 0 {
		return nil
	}
	geometry := features.groups[features.chosen].GeometryType
	types := append([]maplibre.RenderType(nil), maplibre.RenderTypes[geometry]...)
	if len(types) > 0 && m.session != nil && len(m.session.Icons()) > 0 && (geometry == maplibre.Point || m.Options.WriteFGB) {
		types = append(types, maplibre.RenderSymbol)
	}
	return types
}

func (m *Model) iconNames() []string {
	if m.session == nil {
		return nil
	}
	names := make([]string, 0, len(m.session.Icons()))
	for name := range m.session.Icons() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (m *Model) currentIcon(layer *styleLayerState) (string, bool) {
	names := m.iconNames()
	if len(names) == 0 {
		return "", false
	}
	return names[panel.MoveListSelection(layer.selected, 0, len(names))], true
}

// Add only confirmed layers; SVG Edit starts with its current catalog entry.
func (m *Model) addStyleLayer(s *stylingState, kind maplibre.RenderType, iconName string) {
	layer := &styleLayerState{
		style: maplibre.RenderLayerStyle{Type: kind}, properties: m.stylingCatalog.properties[kind],
		options: make(map[string]optionDraft),
	}
	if kind == maplibre.RenderSymbol {
		layer.style.IconName = iconName
		layer.style.Layout = map[string]any{"icon-image": iconName, "symbol-placement": "point"}
		names := m.iconNames()
		layer.selected = panel.MoveListSelection(sort.SearchStrings(names, iconName), 0, len(names))
	}
	s.layers = append(s.layers, layer)
	s.active, s.mode = len(s.layers)-1, styleEdit
	_, s.selected = s.stackItems()
}

// Display the highest drawing layer first, using the CLI's SVG-last rule.
func (s *stylingState) displayOrder() []int {
	order := make([]int, 0, len(s.layers))
	for _, symbol := range []bool{true, false} {
		for i := len(s.layers) - 1; i >= 0; i-- {
			if (s.layers[i].style.Type == maplibre.RenderSymbol) == symbol {
				order = append(order, i)
			}
		}
	}
	return order
}

func (s *stylingState) removeSelectedLayer() {
	if s.mode != styleSelection || s.selected < 0 || s.selected >= len(s.layers) {
		return // The final Add layer row is an action, not a layer.
	}
	index := s.displayOrder()[s.selected]
	s.layers = slices.Delete(s.layers, index, index+1)
	if len(s.layers) == 0 {
		s.active, s.selected, s.firstVisible, s.stackFirstVisible = -1, 0, 0, 0
		return
	}
	s.selected = min(s.selected, len(s.layers)-1)
	if s.active == index {
		s.active = s.displayOrder()[s.selected]
	} else if s.active > index {
		s.active--
	}
}

func (s *stylingState) stackItems() ([]panel.ListItem, int) {
	counts, numbers := map[maplibre.RenderType]int{}, map[int]int{}
	for i, layer := range s.layers {
		counts[layer.style.Type]++
		numbers[i] = counts[layer.style.Type]
	}
	items, active := []panel.ListItem{}, 0
	for row, index := range s.displayOrder() {
		layer := s.layers[index]
		name := renderTypeLabel(layer.style.Type)
		if counts[layer.style.Type] > 1 {
			name = fmt.Sprintf("%s %d", name, numbers[index])
		}
		if layer.style.Type == maplibre.RenderSymbol {
			name = layer.style.IconName + " (" + name + ")"
		}
		included := 0
		var children []string
		names := make([]string, 0, len(layer.options))
		for name := range layer.options {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			draft := layer.options[name]
			if draft.included {
				value := "awaiting value"
				if draft.validationErr != nil {
					value = "invalid value"
				} else if draft.hasValue {
					data, err := json.Marshal(draft.value)
					if err == nil {
						value = string(data)
					} else {
						value = "invalid value"
					}
				}
				children = append(children, name+": "+value)
				included++
			}
		}
		if layer.style.Type == maplibre.RenderSymbol {
			children = append(children, "SVG: "+layer.style.IconName)
		}
		detail := fmt.Sprintf("%d options", included)
		if included == 1 {
			detail = "1 option"
		}
		items = append(items, panel.ListItem{Name: name, Detail: detail, Status: layer.status(), Children: children})
		if index == s.active {
			active = row
		}
	}
	return items, active
}

func (l *styleLayerState) visibleProperties() []styleProperty {
	properties := make([]styleProperty, 0, len(l.properties))
	for _, property := range l.properties {
		if !l.includedOnly || l.options[property.name].included {
			properties = append(properties, property)
		}
	}
	return properties
}

func (l *styleLayerState) currentProperty() (styleProperty, bool) {
	properties := l.visibleProperties()
	if len(properties) == 0 {
		return styleProperty{}, false
	}
	return properties[panel.MoveListSelection(l.selected, 0, len(properties))], true
}

// Preserve the current name when filtering; otherwise choose its next neighbor.
func (l *styleLayerState) recoverSelection(previous string) {
	properties := l.visibleProperties()
	index := sort.Search(len(properties), func(i int) bool { return properties[i].name >= previous })
	l.selected = panel.MoveListSelection(index, 0, len(properties))
}

func (m *Model) stylingInput(msg tea.KeyPressMsg) {
	s := m.currentStyling()
	if s == nil {
		return
	}
	switch msg.String() {
	case "backspace":
		if s.mode == styleEdit {
			layer := s.activeLayer()
			if layer == nil || layer.style.Type == maplibre.RenderSymbol {
				return
			}
			if property, ok := layer.currentProperty(); ok {
				draft := layer.options[property.name]
				if draft.included {
					draft.included = false
					layer.options[property.name] = draft
					layer.recoverSelection(property.name)
				}
			}
		} else {
			s.removeSelectedLayer()
		}
	case "esc":
		if s.mode == styleEdit {
			s.mode = styleSelection
			_, s.selected = s.stackItems()
		} else if s.mode == styleTypes {
			s.mode = styleSelection
		} else if s.mode == styleIcons {
			s.mode = styleTypes
		}
	case "h":
		if s.mode == styleEdit {
			layer := s.activeLayer()
			if layer.style.Type == maplibre.RenderSymbol {
				return
			}
			property, _ := layer.currentProperty()
			layer.includedOnly = !layer.includedOnly
			layer.recoverSelection(property.name)
		}
	case "up", "down":
		delta := 1
		if msg.String() == "up" {
			delta = -1
		}
		switch s.mode {
		case styleTypes:
			s.typeSelected = panel.MoveListSelection(s.typeSelected, delta, len(m.styleTypes()))
		case styleIcons:
			s.iconSelected = panel.MoveListSelection(s.iconSelected, delta, len(m.iconNames()))
		case styleSelection:
			s.selected = panel.MoveListSelection(s.selected, delta, len(s.layers)+1)
		case styleEdit:
			layer := s.activeLayer()
			count := len(layer.visibleProperties())
			if layer.style.Type == maplibre.RenderSymbol {
				count = len(m.iconNames())
			}
			layer.selected = panel.MoveListSelection(layer.selected, delta, count)
		}
	case "enter":
		switch s.mode {
		case styleTypes:
			types := m.styleTypes()
			if len(types) == 0 {
				return
			}
			kind := types[panel.MoveListSelection(s.typeSelected, 0, len(types))]
			if kind == maplibre.RenderSymbol {
				s.mode = styleIcons
				return
			}
			m.addStyleLayer(s, kind, "")
		case styleIcons:
			names := m.iconNames()
			if len(names) == 0 {
				return
			}
			m.addStyleLayer(s, maplibre.RenderSymbol, names[panel.MoveListSelection(s.iconSelected, 0, len(names))])
		case styleSelection:
			if s.selected >= len(s.layers) {
				s.mode = styleTypes
				return
			}
			s.active, s.mode = s.displayOrder()[s.selected], styleEdit
		case styleEdit:
			layer := s.activeLayer()
			if layer.style.Type == maplibre.RenderSymbol {
				if name, ok := m.currentIcon(layer); ok {
					layer.style.IconName = name
					if layer.style.Layout == nil {
						layer.style.Layout = make(map[string]any)
					}
					layer.style.Layout["icon-image"] = name
				}
				return
			}
			if property, ok := layer.currentProperty(); ok {
				if layer.includedOnly {
					m.activePane = panel.ControlsPane
					return
				}
				draft := layer.options[property.name]
				draft.included = !draft.included
				layer.options[property.name] = draft
				layer.recoverSelection(property.name)
			}
		}
	}
}

func (m *Model) stylingPresentation(opts *panel.ScaffoldOptions) {
	opts.ShowStack = true
	opts.StylingEmptyText, opts.StackEmptyText = "Choose a feature", "Choose a feature"
	opts.InfoContent = "Choose a feature"
	s := m.currentStyling()
	if s == nil {
		return
	}
	opts.Stack, opts.ActiveStackLayer = s.stackItems()
	opts.StackFirstVisible = s.stackFirstVisible
	opts.StackEmptyText = "No style layers"
	opts.InfoContent = "Choose a styling option"
	switch s.mode {
	case styleTypes:
		opts.StylingHeading = "Add style layer"
		for _, kind := range m.styleTypes() {
			opts.Styling = append(opts.Styling, panel.ListItem{Name: renderTypeLabel(kind), Detail: "Enter to add"})
		}
		opts.SelectedStyling, opts.FirstVisibleStyling = s.typeSelected, s.typeFirstVisible
		opts.StylingEmptyText = "No available types"
	case styleIcons:
		opts.StylingHeading, opts.StylingEmptyText = "Choose SVG icon", "No SVG icons loaded"
		for _, name := range m.iconNames() {
			opts.Styling = append(opts.Styling, panel.ListItem{Name: name, Detail: "Enter to use"})
		}
		opts.SelectedStyling, opts.FirstVisibleStyling = s.iconSelected, s.iconFirstVisible
	case styleSelection:
		opts.StylingHeading = "Style layers"
		opts.Styling = append(append([]panel.ListItem(nil), opts.Stack...), panel.ListItem{Name: "Add layer", Detail: "Choose type", Status: panel.ListAdd})
		opts.SelectedStyling, opts.FirstVisibleStyling = s.selected, s.firstVisible
		if m.activePane != panel.FeatureStylingPane {
			opts.SelectedStyling = opts.ActiveStackLayer
			opts.HideStylingSelection = len(s.layers) == 0
		}
	case styleEdit:
		layer := s.activeLayer()
		opts.StylingHeading = "Edit " + renderTypeLabel(layer.style.Type)
		if layer.style.Type == maplibre.RenderSymbol {
			opts.StylingEmptyText = "No SVG icons loaded"
			for _, name := range m.iconNames() {
				item := panel.ListItem{Name: name, Detail: "Enter to use", Children: []string{"Type: SVG"}}
				if name == layer.style.IconName {
					item.Status, item.Detail = panel.ListActive, "Current SVG"
				}
				opts.Styling = append(opts.Styling, item)
			}
		} else {
			opts.StylingEmptyText = "No options selected"
			if !layer.includedOnly {
				opts.StylingEmptyText = "No available options"
			}
			for _, property := range layer.visibleProperties() {
				defaultValue := "none"
				if len(property.spec.Default) > 0 {
					var value bytes.Buffer
					if json.Compact(&value, property.spec.Default) == nil {
						defaultValue = value.String()
					}
				}
				opts.Styling = append(opts.Styling, panel.ListItem{Name: property.name, Status: layer.options[property.name].status(), Children: []string{"Type: " + string(property.spec.Type), "Default: " + defaultValue}})
			}
		}
		opts.SelectedStyling, opts.FirstVisibleStyling = layer.selected, layer.firstVisible
		opts.InfoContent = m.propertyInfo(layer)
	}
	if m.stylingCatalog.err != nil && s.mode == styleEdit && s.activeLayer().style.Type != maplibre.RenderSymbol {
		opts.Styling = nil
		opts.StylingEmptyText = "Style reference: " + m.stylingCatalog.err.Error()
		opts.InfoContent = opts.StylingEmptyText
	}
}

func (m *Model) propertyInfo(layer *styleLayerState) string {
	if layer.style.Type == maplibre.RenderSymbol {
		displayName := func(name string) string {
			return strings.NewReplacer("\n", `\n`, "\r", `\r`, "\t", `\t`).Replace(ansi.Strip(name))
		}
		rows := []string{"Current SVG: " + displayName(layer.style.IconName)}
		if name, ok := m.currentIcon(layer); ok {
			rows = append(rows, "Highlighted SVG: "+displayName(name), "", "Enter to use the highlighted SVG.")
		} else {
			rows = append(rows, "", "No SVG icons loaded")
		}
		return strings.Join(rows, "\n")
	}
	property, ok := layer.currentProperty()
	if !ok {
		return "Choose an option"
	}
	spec := property.spec
	rows := []string{property.name, "Section: " + string(property.section)}
	rows = append(rows, "", spec.Doc)
	if spec.Minimum != nil {
		rows = append(rows, fmt.Sprintf("Minimum: %g", *spec.Minimum))
	}
	if spec.Maximum != nil {
		rows = append(rows, fmt.Sprintf("Maximum: %g", *spec.Maximum))
	}
	if spec.Length != nil {
		rows = append(rows, fmt.Sprintf("Length: %d", *spec.Length))
	}
	choices := make([]string, 0, len(spec.Values))
	for choice := range spec.Values {
		choices = append(choices, choice)
	}
	sort.Strings(choices)
	if len(choices) > 0 {
		rows = append(rows, "Choices: "+strings.Join(choices, ", "))
	}
	draft := layer.options[property.name]
	switch draft.status() {
	case panel.ListUnopened:
		rows = append(rows, "", "Excluded — Enter to include")
	case panel.ListIncomplete:
		rows = append(rows, "", "Included — awaiting a value")
	case panel.ListComplete:
		value, _ := json.Marshal(draft.value)
		rows = append(rows, "", "Value: "+string(value))
	}
	return strings.Join(rows, "\n")
}
