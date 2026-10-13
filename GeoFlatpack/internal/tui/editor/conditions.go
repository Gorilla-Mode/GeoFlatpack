package editor

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/style/maplibre"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

type conditionRuleDraft struct {
	id, operator, valueType, text string
	defined                       bool
}

type conditionState struct {
	rules                              []conditionRuleDraft
	geometries                         []maplibre.StyleGroup
	features                           featureState
	valueType                          string
	allowedTypes, stringChoices        []string
	nextID                             int
	selected, firstVisible, ruleIndex  int
	editing                            bool
	choosing                           bool
	choiceSelected, choiceFirstVisible int
	input                              textinput.Model
}

func inferConditionType(types []string) string {
	inferred := ""
	for _, typ := range types {
		var kind string
		switch strings.ToLower(typ) {
		case "missing":
			continue
		case "string", "datetime":
			kind = "string"
		case "bool", "boolean":
			kind = "boolean"
		case "number", "byte", "ubyte", "short", "ushort", "int", "uint", "long", "ulong", "float", "double":
			kind = "number"
		default:
			return "number"
		}
		if inferred != "" && inferred != kind {
			return "number"
		}
		inferred = kind
	}
	if inferred == "" {
		return "number"
	}
	return inferred
}

func (r conditionRuleDraft) value(field string) (any, error) {
	if !r.defined && r.valueType != "missing" {
		return nil, fmt.Errorf("Enter a value")
	}
	var value any
	switch r.valueType {
	case "missing":
		// Missing is an explicit choice; it needs no text value.
		value = nil
	case "string":
		value = r.text
	case "boolean":
		if r.text != "true" && r.text != "false" {
			return nil, fmt.Errorf("Enter true or false")
		}
		value = r.text == "true"
	case "number":
		n := json.Number(strings.TrimSpace(r.text))
		if _, err := json.Marshal(n); err != nil || n == "" {
			return nil, fmt.Errorf("Enter a finite number")
		}
		value = n
	default:
		return nil, fmt.Errorf("Unsupported value type")
	}
	if err := maplibre.ValidateComparison(field, r.operator, value); err != nil {
		return nil, err
	}
	return value, nil
}

func (c *conditionState) valid(field string) bool {
	for _, r := range c.rules {
		if _, err := c.ruleValue(field, r); err != nil {
			return false
		}
	}
	return true
}

func (c *conditionState) label(field string, r conditionRuleDraft) string {
	text := r.text
	if r.valueType == "missing" {
		text = "Missing value"
	} else if !r.defined {
		text = "(unfinished)"
	} else if r.valueType == "string" {
		text = strconv.Quote(text)
	}
	return field + " " + r.operator + " " + text
}

// Stable group IDs retain each stack, omission and cursor as conditions change.
func (c *conditionState) rebuild(field string) {
	old := c.features
	drafts := make(map[maplibre.StyleGroup]int)
	for i, group := range old.groups {
		drafts[group] = i
	}
	next := featureState{chosen: -1}
	appendGroup := func(group maplibre.StyleGroup, label string) {
		item := panel.ListItem{Name: label, Children: []string{label, "Type: " + string(group.GeometryType)}}
		style := stylingState{active: -1}
		if index, ok := drafts[group]; ok {
			style = old.styling[index]
			item.Status, item.Omitted = old.items[index].Status, old.items[index].Omitted
			if index == old.selected {
				next.selected = len(next.items)
			}
			if index == old.chosen {
				next.chosen = len(next.items)
			}
		}
		next.groups = append(next.groups, group)
		next.items = append(next.items, item)
		next.styling = append(next.styling, style)
	}
	for _, rule := range c.rules {
		for _, geometry := range c.geometries {
			appendGroup(maplibre.StyleGroup{GeometryType: geometry.GeometryType, RuleID: rule.id}, c.label(field, rule))
		}
	}
	for _, geometry := range c.geometries {
		appendGroup(geometry, "Otherwise")
	}
	next.selected = panel.MoveListSelection(next.selected, 0, len(next.items))
	next.firstVisible = min(old.firstVisible, next.selected)
	c.features = next
}

func (c *conditionState) snapshot(field string, preview bool) (*maplibre.ConditionalStyle, error) {
	result := &maplibre.ConditionalStyle{Field: field, Otherwise: make(map[maplibre.GeometryType][]maplibre.RenderLayerStyle)}
	indices := make(map[string]int)
	for _, draft := range c.rules {
		value, err := c.ruleValue(field, draft)
		if err != nil {
			return nil, fmt.Errorf("condition %s: %w", draft.id, err)
		}
		indices[draft.id] = len(result.Rules)
		result.Rules = append(result.Rules, maplibre.ConditionalRule{ID: draft.id, Operator: draft.operator, Value: value, Styles: make(map[maplibre.GeometryType][]maplibre.RenderLayerStyle)})
	}
	for i, group := range c.features.groups {
		if c.features.items[i].Omitted {
			continue
		}
		stack, err := snapshotStack(c.features.styling[i], group.GeometryType, preview)
		if err != nil {
			return nil, fmt.Errorf("condition %s: %w", c.features.items[i].Name, err)
		}

		if group.RuleID == "" {
			result.Otherwise[group.GeometryType] = stack
		} else {
			result.Rules[indices[group.RuleID]].Styles[group.GeometryType] = stack
		}
	}
	return result, nil
}

func (m *Editor) conditionsInput(msg tea.Msg) tea.Cmd {
	c := m.conditionEditor
	key, isKey := msg.(tea.KeyPressMsg)
	if c.editing {
		if isKey {
			switch key.String() {
			case "esc":
				c.editing = false
				c.input.Blur()
				return nil
			case "enter":
				r := &c.rules[c.selected/3]
				r.text, r.defined = c.input.Value(), true
				c.editing, c.choosing = false, false
				c.input.Blur()
				m.rebuildConditions()
				return nil
			}
		}
		var cmd tea.Cmd
		c.input, cmd = c.input.Update(msg)
		return cmd
	}
	if !isKey {
		return nil
	}
	if c.choosing {
		return m.conditionChoiceInput(key)
	}
	c.normalizeFocus()
	switch key.String() {
	case "backspace":
		m.Focus(panel.CategoryPane)
	case "up", "down", "tab", "shift+tab":
		delta := 1
		if key.String() == "up" || key.String() == "shift+tab" {
			delta = -1
		}
		c.move(delta, key.String() == "tab" || key.String() == "shift+tab")
	case "ctrl+up", "ctrl+down":
		if c.selected >= len(c.rules)*3 {
			return nil
		}
		from := c.selected / 3
		delta := 1
		if key.String() == "ctrl+up" {
			delta = -1
		}
		to := from + delta
		if to >= 0 && to < len(c.rules) {
			c.rules[from], c.rules[to] = c.rules[to], c.rules[from]
			c.selected += delta * 3
			c.ruleIndex = to
			m.rebuildConditions()
		}
	case "enter", "left", "right":
		if c.selected >= len(c.rules)*3 {
			if key.String() != "enter" {
				return nil
			}
			if c.selected == len(c.rules)*3 {
				c.nextID++
				c.rules = append(c.rules, conditionRuleDraft{id: fmt.Sprintf("rule-%d", c.nextID), operator: "==", valueType: c.valueType})
				c.ruleIndex = len(c.rules) - 1
				c.selected = c.ruleIndex*3 + 2
			} else if len(c.rules) > 0 {
				index := min(c.ruleIndex, len(c.rules)-1)
				c.rules = append(c.rules[:index], c.rules[index+1:]...)
				c.ruleIndex = max(0, index-1)
				c.selected = min(c.selected, len(c.rules)*3)
			}
			m.rebuildConditions()
			return nil
		}
		r := &c.rules[c.selected/3]
		delta := 1
		if key.String() == "left" {
			delta = -1
		}
		cycle := func(value *string, choices []string) {
			index := 0
			for i, choice := range choices {
				if choice == *value {
					index = i
					break
				}
			}
			*value = choices[(index+delta+len(choices))%len(choices)]
		}
		switch c.selected % 3 {
		case 0:
			cycle(&r.operator, []string{"==", "!=", ">", ">=", "<", "<="})
		case 1:
			cycle(&r.valueType, c.allowedTypes)
		case 2:
			if key.String() != "enter" || r.valueType == "missing" {
				return nil
			}
			if r.valueType == "string" || r.valueType == "boolean" {
				c.openValueChooser()
				return nil
			}
			return c.editValue()
		}
		m.rebuildConditions()
	}
	return nil
}

func (m *Editor) conditionsPresentation(opts *panel.ScaffoldOptions) {
	c := m.conditionEditor
	if c.choosing && !c.editing {
		c.choicesPresentation(opts)
		return
	}
	c.normalizeFocus()
	width, _ := panel.ControlsDimensions(*opts)
	for i, rule := range c.rules {
		values := []string{rule.operator, rule.valueType, rule.text}
		if rule.valueType == "missing" {
			values[1], values[2] = "Missing value", "Missing value"
		} else if rule.valueType == "string" && rule.defined {
			values[2] = strconv.Quote(rule.text)
		}
		for j, label := range []string{"Operator", "Value type", "Value"} {
			item := panel.ControlItem{Label: fmt.Sprintf("Condition %d · %s", i+1, label), Value: panelText(values[j])}
			if item.Value == "" {
				item.Value = "(empty)"
			}
			if j == 2 {
				item.Disabled = rule.valueType == "missing"
				if _, err := c.ruleValue(m.conditionField, rule); err != nil {
					item.Error = err.Error()
				}
			}
			if c.editing && c.selected == 3*i+j {
				c.input.SetWidth(max(1, width-1))
				input := c.input
				if m.activePane != panel.ControlsPane || !m.interactive {
					input.Blur()
				}
				item.Value, item.Editing = input.View(), true
			}
			opts.Controls = append(opts.Controls, item)
		}
	}
	opts.Controls = append(opts.Controls, panel.ControlItem{Label: "Add condition", Value: "Enter to add"}, panel.ControlItem{Label: "Remove condition", Value: fmt.Sprintf("Remove condition %d", c.ruleIndex+1), Disabled: len(c.rules) == 0})
	opts.SelectedControl, opts.FirstVisibleControl = c.selected, c.firstVisible
}

func (c *conditionState) valueDisabled(row int) bool {
	return row >= 0 && row < 3*len(c.rules) && row%3 == 2 && c.rules[row/3].valueType == "missing"
}

func (c *conditionState) normalizeFocus() {
	c.selected = panel.MoveListSelection(c.selected, 0, 3*len(c.rules)+2)
	if c.valueDisabled(c.selected) {
		c.selected-- // Keep focus on this rule's value type.
	}
}

func (c *conditionState) move(delta int, wrap bool) {
	count := 3*len(c.rules) + 2
	next := c.selected
	for attempts := 0; attempts < count; attempts++ {
		next += delta
		if wrap {
			next = (next + count) % count
		} else if next < 0 || next >= count {
			return
		}
		if !c.valueDisabled(next) {
			c.selected = next
			if next < 3*len(c.rules) {
				c.ruleIndex = next / 3
			}
			return
		}
	}
}

func panelText(value string) string {
	return strings.NewReplacer("\n", `\n`, "\r", `\r`, "\t", `\t`).Replace(ansi.Strip(value))
}
