package editor

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"GeoFlatpack/fgb"
	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/style/maplibre"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

func conditionType(typ string) string {
	switch strings.ToLower(typ) {
	case "string", "datetime":
		return "string"
	case "bool", "boolean":
		return "boolean"
	case "number", "byte", "ubyte", "short", "ushort", "int", "uint", "long", "ulong", "float", "double":
		return "number"
	}
	return ""
}

// Cache full, raw values once at load time, independently of geometry and the
// abbreviated examples displayed in Categories.
func (c *conditionState) loadValues(data *fgb.Fgb, field maplibre.CategoryField) {
	c.allowedTypes, c.stringChoices = nil, nil
	types, stringsSeen := map[string]bool{}, map[string]bool{}
	for _, feature := range data.Features {
		category, err := maplibre.NewCategoryValue(feature.Properties[field.Name])
		if err != nil {
			continue
		}
		switch value := category.FilterValue().(type) {
		case string:
			types["string"], stringsSeen[value] = true, true
		case bool:
			types["boolean"] = true
		case json.Number:
			types["number"] = true
		}
	}
	if len(types) == 0 {
		for _, typ := range field.Types {
			if kind := conditionType(typ); kind != "" {
				types[kind] = true
			}
		}
	}
	for _, typ := range []string{"number", "string", "boolean"} {
		if types[typ] {
			c.allowedTypes = append(c.allowedTypes, typ)
		}
	}
	c.allowedTypes = append(c.allowedTypes, "missing")
	c.valueType = c.allowedTypes[0]
	for value := range stringsSeen {
		c.stringChoices = append(c.stringChoices, value)
	}
	slices.Sort(c.stringChoices)
}

func (c *conditionState) ruleValue(field string, rule conditionRuleDraft) (any, error) {
	if !slices.Contains(c.allowedTypes, rule.valueType) {
		return nil, fmt.Errorf("Value type %s is unavailable for this field", rule.valueType)
	}
	return rule.value(field)
}

func (c *conditionState) valueChoices() []string {
	if c.rules[c.selected/3].valueType == "boolean" {
		return []string{"true", "false"}
	}
	return c.stringChoices
}

func (c *conditionState) openValueChooser() {
	c.choosing, c.choiceSelected, c.choiceFirstVisible = true, 0, 0
	rule := c.rules[c.selected/3]
	choices := c.valueChoices()
	if rule.valueType == "string" {
		c.choiceSelected = 1 // Start on an observed string or the custom action.
	}
	if rule.defined {
		if index := slices.Index(choices, rule.text); index >= 0 {
			c.choiceSelected += index
		} else if rule.valueType == "string" {
			c.choiceSelected = len(choices) + 1 // Custom value action.
		}
	}
}

func (c *conditionState) editValue() tea.Cmd {
	c.input = textinput.New()
	c.input.Prompt = ""
	c.input.CharLimit = 0
	styles := c.input.Styles()
	styles.Cursor.Blink = false
	c.input.SetStyles(styles)
	c.input.SetValue(c.rules[c.selected/3].text)
	c.input.CursorEnd()
	c.editing = true
	return c.input.Focus()
}

func (m *Editor) conditionChoiceInput(key tea.KeyPressMsg) tea.Cmd {
	c := m.conditionEditor
	choices := c.valueChoices()
	count := len(choices)
	offset := 0
	if c.rules[c.selected/3].valueType == "string" {
		offset = 1 // Missing value precedes the strings.
		count += 2 // Missing value and custom input are always available.
	}
	switch key.String() {
	case "esc":
		c.choosing = false
	case "backspace":
		m.Focus(panel.CategoryPane)
	case "up", "down", "tab", "shift+tab":
		delta := 1
		if key.String() == "up" || key.String() == "shift+tab" {
			delta = -1
		}
		if key.String() == "tab" || key.String() == "shift+tab" {
			c.choiceSelected = (c.choiceSelected + delta + count) % count
		} else {
			c.choiceSelected = panel.MoveListSelection(c.choiceSelected, delta, count)
		}
	case "enter":
		rule := &c.rules[c.selected/3]
		if offset == 1 && c.choiceSelected == 0 {
			rule.valueType = "missing"
			c.choosing = false
			c.normalizeFocus()
			m.rebuildConditions()
			return nil
		}
		index := c.choiceSelected - offset
		if index == len(choices) {
			return c.editValue()
		}
		rule.text, rule.defined = choices[index], true
		c.choosing = false
		m.rebuildConditions()
	}
	return nil
}

func (c *conditionState) choicesPresentation(opts *panel.ScaffoldOptions) {
	stringValue := c.rules[c.selected/3].valueType == "string"
	if stringValue {
		opts.Controls = append(opts.Controls, panel.ControlItem{Label: "Missing value", Value: "Enter to select"})
	}
	for _, value := range c.valueChoices() {
		label := value
		if stringValue {
			label = strconv.Quote(value)
		}
		opts.Controls = append(opts.Controls, panel.ControlItem{Label: label, Value: "Enter to select"})
	}
	if stringValue {
		opts.Controls = append(opts.Controls, panel.ControlItem{Label: "Enter custom value", Value: "Enter to edit"})
	}
	opts.SelectedControl, opts.FirstVisibleControl = c.choiceSelected, c.choiceFirstVisible
}
