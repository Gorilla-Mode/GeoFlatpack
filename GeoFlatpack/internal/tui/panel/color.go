package panel

import (
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/lucasb-eyer/go-colorful"
)

const (
	PickerSquare = iota
	PickerHue
	PickerSaturation
	PickerLightness
	PickerHex
	PickerRed
	PickerGreen
	PickerBlue
	PickerControlCount
)

type ColorPickerOptions struct {
	Hue, Saturation, Lightness float64
	Selected                   int
	Active, Editing, Focused   bool
	Input, Error               string
}

type PickerRect struct{ X, Y, Width, Height int }

func (r PickerRect) Contains(x, y int) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.Width && y < r.Y+r.Height
}

type PickerGeometry struct {
	Controls                        [PickerControlCount]PickerRect
	Width, Height, OffsetX, OffsetY int
	CursorX, CursorY                int
}

// ColorPickerGeometry defines both the logical canvas and its focus-following
// viewport. HSL sliders and text inputs follow the square on its right.
func ColorPickerGeometry(o ColorPickerOptions, width, height int) PickerGeometry {
	const inputWidth, sideWidth = 9, 26

	// Keep the stacked fields and gradients usable in small terminals; the
	// viewport follows focus when this minimum canvas cannot fit.
	rows := max(11, height)
	cols := max(14, width-sideWidth)
	g := PickerGeometry{Width: cols + sideWidth, Height: rows}

	// Reserve the focus outline even when hidden, keeping the gradient and
	// mouse coordinates stable as focus and adjustment change.
	square := PickerRect{X: 1, Y: 1, Width: cols - 2, Height: rows - 2}
	g.Controls[PickerSquare] = square
	g.CursorX = square.X + int(math.Round(o.Saturation*float64(square.Width-1)))
	g.CursorY = square.Y + int(math.Round((1-o.Lightness)*float64(square.Height-1)))
	for i := 0; i < 4; i++ {
		g.Controls[PickerHex+i] = PickerRect{X: cols + 16, Y: i * 3, Width: inputWidth, Height: 2}
	}

	for i := 0; i < 3; i++ {
		g.Controls[PickerHue+i] = PickerRect{X: cols + 1 + i*5, Width: 4, Height: rows}
	}

	target := g.Controls[min(max(0, o.Selected), PickerControlCount-1)]
	if o.Selected == PickerSquare {
		target = PickerRect{X: g.CursorX, Y: g.CursorY, Width: 1, Height: 1}
	} else if o.Selected >= PickerHue && o.Selected <= PickerLightness {
		v := []float64{o.Hue / 360, o.Saturation, o.Lightness}[o.Selected-PickerHue]
		target.Y, target.Height = 1+int(math.Round((1-v)*float64(rows-3))), 1
	}

	if target.X+target.Width > width {
		g.OffsetX = target.X + target.Width - max(1, width)
	}

	if target.Y+target.Height > height {
		g.OffsetY = target.Y + target.Height - max(1, height)
	}

	g.OffsetX = max(0, min(g.OffsetX, max(0, g.Width-width)))
	g.OffsetY = max(0, min(g.OffsetY, max(0, g.Height-height)))

	return g
}

func (g PickerGeometry) Clipped(width, height int) bool { return g.Width > width || g.Height > height }

func RenderColorPicker(o ColorPickerOptions, width, height int, styles ListStyles) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	g := ColorPickerGeometry(o, width, height)
	canvas := make([][]string, g.Height)
	for y := range canvas {
		canvas[y] = make([]string, g.Width)
		for x := range canvas[y] {
			canvas[y][x] = " "
		}
	}

	write := func(x, y, w int, value string, style lipgloss.Style) {
		if y < 0 || y >= len(canvas) {
			return
		}

		// Store styled spans in their first cell; blank continuation cells have
		// zero width, so ANSI-aware cropping still operates on terminal cells.
		value = ansi.Truncate(value, w, "")
		value += strings.Repeat(" ", max(0, w-ansi.StringWidth(value)))
		canvas[y][x] = style.Inline(true).Render(value)
		for i := 1; i < w; i++ {
			canvas[y][x+i] = ""
		}
	}

	label := func(index int) lipgloss.Style {
		if index == o.Selected {
			return styles.SelectedName
		}

		return styles.Name
	}

	square := g.Controls[PickerSquare]
	for y := 0; y < square.Height; y++ {
		for x := 0; x < square.Width; x++ {
			c := colorful.Hsl(o.Hue, float64(x)/float64(square.Width-1), 1-float64(y)/float64(square.Height-1))
			style := lipgloss.NewStyle().Background(lipgloss.Color(c.Hex()))
			cell := " "
			if square.X+x == g.CursorX && square.Y+y == g.CursorY {
				cell = "○"
				if o.Active && o.Selected == PickerSquare && o.Focused {
					cell = "●"
				}

				fg := "#ffffff"
				if c.R*.299+c.G*.587+c.B*.114 > .5 {
					fg = "#000000"
				}

				style = style.Foreground(lipgloss.Color(fg)).Bold(true)
			}

			canvas[square.Y+y][square.X+x] = style.Render(cell)
		}
	}

	if o.Focused && o.Selected == PickerSquare && !o.Active {
		outline := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
		write(square.X-1, square.Y-1, square.Width+2, "┌"+strings.Repeat("─", square.Width)+"┐", outline)
		write(square.X-1, square.Y+square.Height, square.Width+2, "└"+strings.Repeat("─", square.Width)+"┘", outline)
		for y := square.Y; y < square.Y+square.Height; y++ {
			write(square.X-1, y, 1, "│", outline)
			write(square.X+square.Width, y, 1, "│", outline)
		}
	}

	color := colorful.Hsl(o.Hue, o.Saturation, o.Lightness)
	r, gc, b := color.RGB255()
	values := []string{color.Hex(), fmt.Sprint(r), fmt.Sprint(gc), fmt.Sprint(b)}
	for i, name := range []string{"Hex", "R", "G", "B"} {
		index := PickerHex + i
		rect := g.Controls[index]
		write(rect.X, rect.Y, rect.Width, name, label(index))
		value := values[i]
		if o.Editing && o.Selected == index {
			value = o.Input
			if i == 0 {
				value = "#" + value
			}
		}

		write(rect.X, rect.Y+1, rect.Width, value, label(index).Background(styles.Item.GetBackground()))
	}

	for i, name := range []string{"H", "S", "L"} {
		index := PickerHue + i
		rect := g.Controls[index]
		write(rect.X, 0, rect.Width, name, label(index))
		v := []float64{o.Hue / 360, o.Saturation, o.Lightness}[i]
		marker := 1 + int(math.Round((1-v)*float64(rect.Height-3)))
		for y := 1; y < rect.Height-1; y++ {
			t := 1 - float64(y-1)/float64(rect.Height-3)
			h, s, l := o.Hue, o.Saturation, o.Lightness
			switch i {
			case 0:
				h, s, l = t*360, 1, .5
			case 1:
				s = t
			case 2:
				l = t
			}

			gradient := colorful.Hsl(h, s, l)
			style := lipgloss.NewStyle().Background(lipgloss.Color(gradient.Hex()))
			write(rect.X+1, y, 2, "  ", style)
			if y == marker {
				write(rect.X, y, 1, "▸", label(index))
			}
		}

		number := fmt.Sprintf("%.0f", v*100)
		if i == 0 {
			number = fmt.Sprintf("%.0f", o.Hue)
		}

		write(rect.X, rect.Height-1, rect.Width, number, label(index))
	}

	lines := make([]string, 0, height)
	for y := g.OffsetY; y < min(g.Height, g.OffsetY+height); y++ {
		line := ansi.Cut(strings.Join(canvas[y], ""), g.OffsetX, g.OffsetX+width)
		lines = append(lines, line)
	}

	if o.Error != "" {
		for len(lines) < height {
			lines = append(lines, "")
		}

		lines[len(lines)-1] = styles.Name.Foreground(lipgloss.Color("1")).Inline(true).Render(ansi.Truncate(listText(o.Error), width, "…"))
	}

	return strings.Join(lines, "\n")
}
