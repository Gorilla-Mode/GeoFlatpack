package tui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"GeoFlatpack/internal/tui/panel"
	tea "charm.land/bubbletea/v2"
	"github.com/lucasb-eyer/go-colorful"
)

type colorControl struct {
	hue, saturation, lightness float64
	selected                   int
	active, dragging           bool
}

func decodePickerColor(value any) (colorful.Color, error) {
	s := strings.TrimSpace(fmt.Sprint(value))
	if strings.HasPrefix(s, "#") {
		if len(s) == 9 {
			s = s[:7]
		}
		return colorful.Hex(s)
	}
	lower := strings.ToLower(s)
	if lower == "transparent" {
		return colorful.Color{}, nil
	}
	for _, prefix := range []string{"rgb(", "rgba("} {
		if strings.HasPrefix(lower, prefix) && strings.HasSuffix(lower, ")") {
			parts := strings.Split(lower[len(prefix):len(lower)-1], ",")
			if (prefix == "rgb(" && len(parts) != 3) || (prefix == "rgba(" && len(parts) != 4) {
				break
			}
			channels := [3]float64{}
			for i := range channels {
				v, err := strconv.ParseFloat(strings.TrimSpace(parts[i]), 64)
				if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 255 {
					return colorful.Color{}, fmt.Errorf("invalid RGB color")
				}
				channels[i] = v / 255
			}
			return colorful.Color{R: channels[0], G: channels[1], B: channels[2]}, nil
		}
	}
	return colorful.Color{}, fmt.Errorf("color has no simple RGB value")
}

func (c *colorControl) rgb() colorful.Color { return colorful.Hsl(c.hue, c.saturation, c.lightness) }

func (c *colorControl) setRGB(value colorful.Color) {
	h, s, l := value.Hsl()
	// Hue is undefined for gray, and saturation at black/white. Retain the
	// user's coordinates so adjusting away from these colors stays predictable.
	if s > 0 {
		c.hue = h
	}
	if l > 0 && l < 1 {
		c.saturation = s
	}
	c.lightness = l
}

func (c *colorControl) load(value any) {
	rgb, err := decodePickerColor(value)
	if err != nil {
		rgb = colorful.Color{}
	}
	c.setRGB(rgb)
	c.active, c.dragging = false, false
}

func (c *colorControl) apply(layer *styleLayerState, p styleProperty) {
	d := layer.options[p.name]
	d.included, d.hasValue, d.validationErr = true, true, nil
	d.value = c.rgb().Hex()
	layer.options[p.name] = d
}

func (c *colorControl) fieldValue() string {
	if c.selected == panel.PickerHex {
		return strings.TrimPrefix(c.rgb().Hex(), "#")
	}
	r, g, b := c.rgb().RGB255()
	values := []uint8{r, g, b}
	return strconv.Itoa(int(values[c.selected-panel.PickerRed]))
}

func (c *colorControl) move(delta int) {
	count := panel.PickerControlCount
	c.selected = (c.selected + delta + count) % count
	c.active, c.dragging = false, false
}

// adjust changes a focused slider or RGB channel without entering edit mode.
func (c *colorControl) adjust(delta int) bool {
	switch c.selected {
	case panel.PickerHue:
		c.hue = math.Mod(c.hue+float64(delta)+360, 360)
	case panel.PickerSaturation:
		c.saturation = max(0, min(1, c.saturation+float64(delta)/100))
	case panel.PickerLightness:
		c.lightness = max(0, min(1, c.lightness+float64(delta)/100))
	case panel.PickerRed, panel.PickerGreen, panel.PickerBlue:
		r, g, b := c.rgb().RGB255()
		channels := []uint8{r, g, b}
		index := c.selected - panel.PickerRed
		channel := uint8(max(0, min(255, int(channels[index])+delta)))
		if channel == channels[index] {
			return false
		}
		channels[index] = channel
		c.setRGB(colorful.Color{R: float64(channels[0]) / 255, G: float64(channels[1]) / 255, B: float64(channels[2]) / 255})
	default:
		return false
	}
	return true
}

func (c *controlState) colorFieldChanged(layer *styleLayerState, p styleProperty) bool {
	v := strings.TrimSpace(c.input.Value())
	var value colorful.Color
	if c.color.selected == panel.PickerHex {
		if strings.HasPrefix(v, "#") {
			position := c.input.Position()
			v = strings.TrimPrefix(v, "#")
			c.input.SetValue(v)
			c.input.SetCursor(max(0, position-1))
		}
		if len(v) != 6 {
			c.err = "Enter six hexadecimal digits"
			return false
		}
		var err error
		value, err = colorful.Hex("#" + v)
		if err != nil {
			c.err = "Enter six hexadecimal digits"
			return false
		}
	} else {
		channel, err := strconv.Atoi(v)
		if err != nil || channel < 0 || channel > 255 {
			c.err = "RGB must be an integer from 0 to 255"
			return false
		}
		r, g, b := c.color.rgb().RGB255()
		channels := []uint8{r, g, b}
		channels[c.color.selected-panel.PickerRed] = uint8(channel)
		value = colorful.Color{R: float64(channels[0]) / 255, G: float64(channels[1]) / 255, B: float64(channels[2]) / 255}
	}
	c.color.setRGB(value)
	c.color.apply(layer, p)
	c.err = ""
	return true
}

func (c *controlState) leaveColorInput() {
	c.editing = false
	c.input.Blur()
	c.err = ""
}

func (c *controlState) colorInput(layer *styleLayerState, p styleProperty, msg tea.Msg) tea.Cmd {
	color := c.color
	k, isKey := msg.(tea.KeyPressMsg)
	if c.editing {
		if isKey {
			switch k.String() {
			case "esc":
				c.leaveColorInput()
				return nil
			case "enter":
				if c.colorFieldChanged(layer, p) {
					c.leaveColorInput()
				}
				return nil
			case "tab", "shift+tab":
				c.leaveColorInput()
				delta := 1
				if k.String() == "shift+tab" {
					delta = -1
				}
				color.move(delta)
				return nil
			}
		}
		before := c.input.Value()
		var cmd tea.Cmd
		c.input, cmd = c.input.Update(msg)
		if c.input.Value() != before {
			c.colorFieldChanged(layer, p)
		}
		return cmd
	}
	if !isKey {
		return nil
	}
	if k.String() == "esc" {
		color.active, color.dragging = false, false
		return nil
	}
	if k.String() == "tab" || k.String() == "shift+tab" {
		delta := 1
		if k.String() == "shift+tab" {
			delta = -1
		}
		color.move(delta)
		return nil
	}
	if color.active {
		delta := 0
		switch k.String() {
		case "up", "right":
			delta = 1
		case "down", "left":
			delta = -1
		case "enter":
			color.active = false
			return nil
		default:
			return nil
		}
		if color.selected == panel.PickerSquare {
			if k.String() == "left" || k.String() == "right" {
				color.saturation = max(0, min(1, color.saturation+float64(delta)/100))
			} else {
				color.lightness = max(0, min(1, color.lightness+float64(delta)/100))
			}
		} else {
			color.adjust(delta)
		}
		color.apply(layer, p)
		return nil
	}
	switch k.String() {
	case "left", "right":
		delta := 1
		if k.String() == "left" {
			delta = -1
		}
		if color.adjust(delta) {
			color.apply(layer, p)
		}
	case "up":
		color.move(-1)
	case "down":
		color.move(1)
	case "enter":
		switch color.selected {
		case panel.PickerSquare, panel.PickerHue, panel.PickerSaturation, panel.PickerLightness:
			color.active = true
		default:
			c.editing = true
			c.err = ""
			c.input.SetValue(color.fieldValue())
			c.input.CursorEnd()
			return c.input.Focus()
		}
	}
	return nil
}

func (c *colorControl) presentation(control *controlState, p styleProperty, focused bool) panel.ColorPickerOptions {
	input := control.input
	if !focused {
		input.Blur()
	}
	return panel.ColorPickerOptions{Hue: c.hue, Saturation: c.saturation, Lightness: c.lightness, Selected: c.selected,
		Active: c.active, Editing: control.editing, Focused: focused, Input: input.View(), Error: control.err}
}

func (m *Model) colorPickerOptions() (panel.ColorPickerOptions, bool) {
	_, p, c := m.currentControl()
	if c == nil || c.color == nil {
		return panel.ColorPickerOptions{}, false
	}
	return c.color.presentation(c, p, m.activePane == panel.ControlsPane), true
}

func (m *Model) stopColorDragging() {
	if _, _, c := m.currentControl(); c != nil && c.color != nil {
		c.color.dragging = false
	}
}

func (m *Model) colorMouseInput(msg tea.Msg) tea.Cmd {
	layer, p, c := m.currentControl()
	if c == nil || c.color == nil {
		return nil
	}
	var mouse tea.Mouse
	click := false
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		mouse = msg.Mouse()
		click = true
	case tea.MouseMotionMsg:
		mouse = msg.Mouse()
		if !c.color.dragging {
			return nil
		}
	case tea.MouseReleaseMsg:
		c.color.dragging = false
		return nil
	default:
		return nil
	}
	if click && mouse.Button != tea.MouseLeft {
		return nil
	}
	l := m.layout()
	opts := m.scaffoldOptions(l)
	region := panel.ControlsRegion(opts)
	x := mouse.X - l.frame.GetPaddingLeft() - region.X
	y := mouse.Y - l.frame.GetPaddingTop() - l.headerHeight - l.gap - region.Y
	if click && !region.Contains(mouse.X-l.frame.GetPaddingLeft(), mouse.Y-l.frame.GetPaddingTop()-l.headerHeight-l.gap) {
		return nil
	}
	g := panel.ColorPickerGeometry(*opts.ColorPicker, region.Width, region.Height)
	x += g.OffsetX
	y += g.OffsetY
	if click {
		index := -1
		for i, r := range g.Controls {
			if r.Contains(x, y) {
				index = i
				break
			}
		}
		if index < 0 {
			return nil
		}
		c.leaveColorInput()
		c.color.selected = index
		c.color.active = false
		c.color.dragging = false
		m.activePane = panel.ControlsPane
		if index >= panel.PickerHex && index <= panel.PickerBlue {
			return c.colorInput(layer, p, tea.KeyPressMsg{Code: tea.KeyEnter})
		}
		c.color.active, c.color.dragging = true, true
	}
	index := c.color.selected
	r := g.Controls[index]
	if index == panel.PickerSquare {
		c.color.saturation = max(0, min(1, float64(x-r.X)/float64(r.Width-1)))
		c.color.lightness = 1 - max(0, min(1, float64(y-r.Y)/float64(r.Height-1)))
	} else if index >= panel.PickerHue && index <= panel.PickerLightness {
		v := 1 - max(0, min(1, float64(y-r.Y-1)/float64(r.Height-3)))
		switch index {
		case panel.PickerHue:
			c.color.hue = v * 360
		case panel.PickerSaturation:
			c.color.saturation = v
		case panel.PickerLightness:
			c.color.lightness = v
		}
	} else {
		return nil
	}
	c.color.apply(layer, p)
	return nil
}
