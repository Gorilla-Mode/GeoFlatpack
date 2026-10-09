package editor

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/style/maplibre"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// Each property owns its form, including an unfinished edit and focus window.
type controlState struct {
	fields                                  []string
	selected, button, firstVisible, element int
	editing                                 bool
	input                                   textinput.Model
	err                                     string
	color                                   *colorControl
}

type controlRow struct {
	label, action     string
	field             int
	numeric, disabled bool
	removable         bool
}

func (m *Editor) currentControl() (*styleLayerState, styleProperty, *controlState) {
	s := m.currentStyling()
	if s == nil || s.mode != styleEdit || s.activeLayer() == nil {
		return nil, styleProperty{}, nil
	}

	layer := s.activeLayer()
	p, ok := layer.currentProperty()
	if !ok {
		return nil, styleProperty{}, nil
	}

	if layer.controls == nil {
		layer.controls = make(map[string]*controlState)
	}

	c := layer.controls[p.name]
	if c == nil {
		c = &controlState{input: textinput.New()}
		c.input.Prompt = ""
		inputStyles := c.input.Styles()
		inputStyles.Cursor.Blink = false
		c.input.SetStyles(inputStyles)
		var value any
		d := layer.options[p.name]
		if d.hasValue {
			value = d.value
		} else if len(p.spec.Default) > 0 {
			_ = json.Unmarshal(p.spec.Default, &value)
		}

		c.load(p, value)
		layer.controls[p.name] = c
	}

	c.normalizeFocus(p)
	if c.color != nil && m.activePane == panel.ControlsPane && m.interactive && !layer.options[p.name].hasValue {
		c.color.apply(layer, p)
	}

	return layer, p, c
}

func (c *controlState) resetFocus() {
	c.selected, c.button, c.element, c.firstVisible = 0, 0, 0, 0
	c.editing = false
	c.input.Blur()
	c.err = ""
}

// A shrinking form can leave a cached field position on an action, a disabled
// row, or outside the list. Recover before rendering or indexing input fields.
func (c *controlState) normalizeFocus(p styleProperty) {
	if p.spec.Type == maplibre.ColorType {
		if c.color != nil && (c.color.selected < 0 || c.color.selected >= panel.PickerControlCount) {
			c.color.selected = 0
			c.color.active, c.color.dragging, c.editing = false, false, false
			c.input.Blur()
		}

		return
	}

	rows := c.rows(p)
	invalid := c.selected < 0 || c.selected >= len(rows)
	if !invalid {
		row := rows[c.selected]
		invalid = row.disabled || (c.editing && row.action != "")
	}

	if invalid {
		c.resetFocus()
		for i, row := range rows {
			if !row.disabled {
				c.selected = i
				break
			}
		}
	}

	if c.selected < len(rows) {
		lastButton := 0
		if rows[c.selected].numeric {
			lastButton = 2
		}

		if rows[c.selected].removable {
			lastButton = 3
		}

		if c.button < 0 || c.button > lastButton {
			c.button = 0
		}
	}

	if c.element < 0 || c.element >= len(c.fields) {
		c.element = 0
	}

	if c.firstVisible < 0 || c.firstVisible >= len(rows) {
		c.firstVisible = 0
	}
}

func scalarText(value any) string {
	if value == nil {
		return ""
	}

	if s, ok := value.(string); ok {
		return s
	}

	b, _ := json.Marshal(value)

	return string(b)
}

func (c *controlState) load(p styleProperty, value any) {
	switch p.spec.Type {
	case maplibre.ColorType:
		c.fields = nil
		if c.color == nil {
			c.color = &colorControl{}
		}

		c.color.load(value)
	case maplibre.ArrayType:
		c.fields = nil

		// Normalize slices supplied by tests or callers as well as decoded JSON.
		b, _ := json.Marshal(value)
		var values []any
		_ = json.Unmarshal(b, &values)
		for _, v := range values {
			c.fields = append(c.fields, scalarText(v))
		}

		if p.spec.Length != nil {
			for len(c.fields) < *p.spec.Length {
				c.fields = append(c.fields, "0")
			}
		}
	case maplibre.TransitionType:
		c.fields = []string{"0", "0"}
		if obj, ok := value.(map[string]any); ok {
			for i, k := range []string{"duration", "delay"} {
				if v, ok := obj[k]; ok {
					c.fields[i] = scalarText(v)
				}
			}
		}
	default:
		c.fields = []string{scalarText(value)}
	}
}

func (c *controlState) rows(p styleProperty) []controlRow {
	var rows []controlRow
	for i := range c.fields {
		label := "Value"
		if p.spec.Type == maplibre.ArrayType {
			label = fmt.Sprintf("Element %d", i+1)
		}

		if p.spec.Type == maplibre.TransitionType {
			label = []string{"Duration (ms)", "Delay (ms)"}[i]
		}

		rows = append(rows, controlRow{label: label, field: i, numeric: p.spec.Type == maplibre.NumberType || p.spec.Type == maplibre.ArrayType || p.spec.Type == maplibre.TransitionType, removable: p.spec.Type == maplibre.ArrayType && p.spec.Length == nil})
	}

	if p.spec.Type == maplibre.ArrayType && p.spec.Length == nil {
		rows = append(rows, controlRow{label: "Add element", action: "add"}, controlRow{label: "Remove element", action: "remove", disabled: len(c.fields) == 0})
	}

	rows = append(rows, controlRow{label: "Use default", action: "default", disabled: len(p.spec.Default) == 0})

	return rows
}

func (c *controlState) move(rows []controlRow, delta int, tab bool) {
	if len(rows) == 0 {
		return
	}

	c.selected = panel.MoveListSelection(c.selected, 0, len(rows))
	if tab && rows[c.selected].numeric {
		b := c.button + delta
		last := 2
		if rows[c.selected].removable {
			last = 3
		}

		if b >= 0 && b <= last {
			c.button = b
			return
		}
	}

	next := c.selected + delta
	for attempts := 0; attempts < len(rows); attempts++ {
		if tab {
			next = (next + len(rows)) % len(rows)
		}

		if next < 0 || next >= len(rows) {
			return
		}

		if !rows[next].disabled {
			break
		}

		next += delta
	}

	if next < 0 || next >= len(rows) || rows[next].disabled {
		return
	}

	c.selected = next
	c.button = 0
	if tab && delta < 0 && rows[next].numeric {
		c.button = 2
		if rows[next].removable {
			c.button = 3
		}
	}

	if rows[next].action == "" {
		c.element = rows[next].field
	}
}

func numericSpec(p styleProperty) maplibre.PropertySpec {
	s := p.spec
	s.Type = maplibre.NumberType
	if p.spec.Type == maplibre.TransitionType {
		zero := 0.0
		s.Minimum = &zero
		s.Maximum = nil
	}

	return s
}

func parseControlNumber(input string, spec maplibre.PropertySpec) (any, error) {
	if strings.HasPrefix(strings.TrimSpace(input), "json:") {
		return nil, fmt.Errorf("enter a number")
	}

	return maplibre.ParseProperty(input, spec)
}

func (c *controlState) parse(p styleProperty, fields []string) (any, error) {
	switch p.spec.Type {
	case maplibre.ArrayType:
		if p.spec.Length != nil && len(fields) != *p.spec.Length {
			return nil, fmt.Errorf("expected %d elements", *p.spec.Length)
		}

		values := make([]any, len(fields))
		for i, f := range fields {
			v, err := parseControlNumber(f, numericSpec(p))
			if err != nil {
				return nil, fmt.Errorf("element %d: %w", i+1, err)
			}

			values[i] = v
		}

		return values, nil
	case maplibre.TransitionType:
		values := map[string]any{}
		for i, k := range []string{"duration", "delay"} {
			v, err := parseControlNumber(fields[i], numericSpec(p))
			if err != nil {
				return nil, fmt.Errorf("%s: %w", k, err)
			}

			values[k] = v
		}

		return values, nil
	case maplibre.StringType, maplibre.ReferenceType, "resolvedImage":
		return fields[0], nil
	default:

		// Typed controls accept literal values; the CLI's json: escape is not an input type.
		if strings.HasPrefix(strings.TrimSpace(fields[0]), "json:") {
			return nil, fmt.Errorf("enter a literal %s value", p.spec.Type)
		}

		return maplibre.ParseProperty(fields[0], p.spec)
	}
}

func (c *controlState) commit(layer *styleLayerState, p styleProperty, fields []string) bool {
	value, err := c.parse(p, fields)
	if err != nil {
		c.err = err.Error()

		return false
	}

	d := layer.options[p.name]
	d.value = value
	d.hasValue = true
	d.included = true
	d.validationErr = nil
	layer.options[p.name] = d
	c.fields = fields
	c.err = ""

	return true
}

func (c *controlState) adjust(layer *styleLayerState, p styleProperty, delta int) {
	rows := c.rows(p)
	row := rows[c.selected]
	if row.action != "" {
		return
	}

	fields := append([]string(nil), c.fields...)
	if row.numeric {
		s := numericSpec(p)
		v, err := strconv.ParseFloat(fields[row.field], 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			v = 0
		}

		step := 1.0
		if p.spec.Type == maplibre.NumberType && s.Minimum != nil && s.Maximum != nil && *s.Minimum == 0 && *s.Maximum == 1 {
			step = .1
		}

		if p.spec.Type == maplibre.TransitionType {
			step = 50
		}

		v += float64(delta) * step
		if math.Abs(v) <= math.MaxFloat64/1e9 {
			v = math.Round(v*1e9) / 1e9
		}

		if s.Minimum != nil {
			v = max(v, *s.Minimum)
		}

		if s.Maximum != nil {
			v = min(v, *s.Maximum)
		}

		fields[row.field] = strconv.FormatFloat(v, 'f', -1, 64)
	} else if p.spec.Type == maplibre.BooleanType {
		fields[row.field] = strconv.FormatBool(fields[row.field] != "true")
	} else if p.spec.Type == maplibre.EnumType {
		choices := make([]string, 0, len(p.spec.Values))
		for choice := range p.spec.Values {
			choices = append(choices, choice)
		}

		sort.Strings(choices)
		if len(choices) == 0 {
			return
		}

		i := sort.SearchStrings(choices, fields[row.field])
		if i >= len(choices) || choices[i] != fields[row.field] {
			if delta > 0 {
				i = -1
			} else {
				i = 0
			}
		}

		i = (i + delta + len(choices)) % len(choices)
		fields[row.field] = choices[i]
	} else {
		return
	}

	c.commit(layer, p, fields)
}

func (m *Editor) controlsInput(msg tea.Msg) tea.Cmd {
	layer, p, c := m.currentControl()
	if c == nil {
		return nil
	}

	if c.color != nil {
		return c.colorInput(layer, p, msg)
	}

	key, isKey := msg.(tea.KeyPressMsg)
	if c.editing {
		if isKey {
			switch key.String() {
			case "esc":
				c.editing = false
				c.input.Blur()
				c.err = ""

				return nil
			case "enter":
				fields := append([]string(nil), c.fields...)
				fields[c.selected] = c.input.Value()
				if c.commit(layer, p, fields) {
					c.editing = false
					c.input.Blur()
				}

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

	rows := c.rows(p)
	c.selected = panel.MoveListSelection(c.selected, 0, len(rows))
	row := rows[c.selected]
	switch key.String() {
	case "up":
		c.move(rows, -1, false)
	case "down":
		c.move(rows, 1, false)
	case "tab":
		c.move(rows, 1, true)
	case "shift+tab":
		c.move(rows, -1, true)
	case "left":
		if !row.disabled {
			c.adjust(layer, p, -1)
		}
	case "right":
		if !row.disabled {
			c.adjust(layer, p, 1)
		}
	case "enter":
		if row.disabled {
			return nil
		}

		switch row.action {
		case "default":
			var value any
			if err := json.Unmarshal(p.spec.Default, &value); err != nil {
				c.err = err.Error()

				return nil
			}

			d := layer.options[p.name]
			d.value = value
			d.hasValue = true
			d.included = true
			d.validationErr = nil
			layer.options[p.name] = d
			c.load(p, value)
			c.err = ""
			c.button = 0
		case "add":
			fields := append(append([]string(nil), c.fields...), "0")
			if c.commit(layer, p, fields) {
				c.selected = len(fields) - 1
				c.element = c.selected
				c.button = 0
			}
		case "remove":
			i := min(c.element, len(c.fields)-1)
			fields := append([]string(nil), c.fields[:i]...)
			fields = append(fields, c.fields[i+1:]...)
			if c.commit(layer, p, fields) {
				c.resetFocus()
			}
		default:
			if row.removable && c.button == 3 {
				fields := append([]string(nil), c.fields[:row.field]...)
				fields = append(fields, c.fields[row.field+1:]...)
				if c.commit(layer, p, fields) {
					c.resetFocus()
				}

				return nil
			}

			if row.numeric && c.button > 0 {
				delta := -1
				if c.button == 2 {
					delta = 1
				}

				c.adjust(layer, p, delta)

				return nil
			}

			if p.spec.Type == maplibre.BooleanType || p.spec.Type == maplibre.EnumType {
				c.adjust(layer, p, 1)

				return nil
			}

			c.editing = true
			c.err = ""
			c.input.SetValue(c.fields[row.field])
			c.input.CursorEnd()

			return c.input.Focus()
		}
	}

	return nil
}

func (m *Editor) Editing() bool {
	if !m.interactive || m.activePane != panel.ControlsPane {
		return false
	}

	_, _, c := m.currentControl()

	return c != nil && (c.editing || (c.color != nil && c.color.active))
}

func (m *Editor) controlsPresentation(opts *panel.ScaffoldOptions) {
	opts.ControlsEmptyText = "Choose a styling option"
	_, p, c := m.currentControl()
	if c == nil {
		if s := m.currentStyling(); s != nil && s.mode == styleEdit && s.activeLayer().optionOffset() > 0 && s.activeLayer().selected == 0 {
			opts.ControlsEmptyText = "Change SVG in Styling"
		}

		return
	}

	if c.color != nil {
		width, height := panel.ControlsDimensions(*opts)
		geometry := panel.ColorPickerGeometry(c.color.presentation(c, p, true), width, height)
		if c.color.selected >= panel.PickerHex && c.color.selected <= panel.PickerBlue {
			reserved := 1
			if c.color.selected == panel.PickerHex {
				reserved++
			}

			c.input.SetWidth(max(1, geometry.Controls[c.color.selected].Width-reserved))
		}

		o := c.color.presentation(c, p, m.activePane == panel.ControlsPane && m.interactive)
		opts.ColorPicker = &o
		return
	}

	rows := c.rows(p)
	c.selected = panel.MoveListSelection(c.selected, 0, len(rows))
	width, _ := panel.ControlsDimensions(*opts)
	for i, row := range rows {
		item := panel.ControlItem{Label: row.label, Disabled: row.disabled, Numeric: row.numeric, Removable: row.removable}
		if row.action == "" {
			item.Value = strings.NewReplacer("\n", `\n`, "\r", `\r`, "\t", `\t`).Replace(ansi.Strip(c.fields[row.field]))
			if item.Value == "" {
				item.Value = "(empty)"
			}

			if c.editing && i == c.selected {
				reserved := 1 // the input renders a cursor cell beyond its width
				if row.numeric {
					reserved += 8
				}

				if row.removable {
					reserved += 4
				}

				c.input.SetWidth(max(1, width-reserved))
				c.input.SetCursor(c.input.Position())
				input := c.input
				if m.activePane != panel.ControlsPane || !m.interactive {
					input.Blur()
				}

				item.Value = input.View()
				item.Editing = true
			}
		} else {
			item.Value = "Enter to use"
			if row.action == "add" {
				item.Value = "Enter to add"
			}

			if row.action == "remove" {
				item.Value = fmt.Sprintf("Remove element %d", c.element+1)
			}

			if row.disabled {
				item.Value = "No default available"
				if row.action == "remove" {
					item.Value = "No elements"
				}
			}

			if row.action == "default" && !row.disabled {
				item.Value = string(p.spec.Default)
			}
		}

		if i == c.selected {
			item.Button = c.button
			item.Error = c.err
		}

		opts.Controls = append(opts.Controls, item)
	}

	opts.SelectedControl = c.selected
	opts.FirstVisibleControl = c.firstVisible
}
