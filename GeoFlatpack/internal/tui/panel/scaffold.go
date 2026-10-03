package panel

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Pane identifies the selectable scaffold panes in navigation order.
type Pane int

const (
	LayerPane Pane = iota
	CategoryPane
	FeaturePane
	ControlsPane
	PaneCount
)

// ScaffoldOptions separates the body dimensions and focus styles from content.
type ScaffoldOptions struct {
	Width             int
	Height            int
	ActivePane        Pane
	PaneStyle         lipgloss.Style
	InactivePaneStyle lipgloss.Style
	TitleStyle        lipgloss.Style
	MutedStyle        lipgloss.Style
	ListStyles        ListStyles
	Layers            []ListItem
	SelectedLayer     int
	FirstVisibleLayer int
}

type paneSize struct {
	width  int
	height int
}

type scaffoldLayout struct {
	gap      int
	columns  [4]paneSize
	preview  paneSize
	controls paneSize
	split    bool
}

func newScaffoldLayout(width, height int) scaffoldLayout {
	width, height = max(0, width), max(0, height)
	l := scaffoldLayout{gap: min(1, width/3)}
	available := width - 3*l.gap
	used := 0
	for i, ratio := range [...]int{18, 18, 27} {
		w := available * ratio / 100
		l.columns[i] = paneSize{w, height}
		used += w
	}
	l.columns[3] = paneSize{available - used, height}

	l.preview = l.columns[3]
	interior := l.preview.interior()
	if interior.width > 0 && interior.height >= 3 {
		l.split = true
		// Reserve the shared divider, then split the heading/content rows.
		// Preview gets the extra row when the remaining height is odd.
		remaining := interior.height - 1
		l.preview.height = (remaining + 1) / 2
		l.controls = paneSize{l.preview.width, remaining / 2}
		if interior != l.columns[3] {
			l.preview.height += 2 // top border and shared divider
			l.controls.height++   // bottom border
		}
	}
	return l
}

func (s paneSize) interior() paneSize {
	if s.width >= 3 && s.height >= 3 {
		return paneSize{s.width - 2, s.height - 2}
	}
	return s
}

type paneOptions struct {
	size    paneSize
	heading string
	box     lipgloss.Style
	title   lipgloss.Style
	content string
}

// Scaffold fills its assigned body directly; it is never viewport content.
func Scaffold(opts ScaffoldOptions) string {
	if opts.Width <= 0 || opts.Height <= 0 {
		return ""
	}
	l := newScaffoldLayout(opts.Width, opts.Height)
	selectable := func(pane Pane, size paneSize, heading string) string {
		box, title := opts.InactivePaneStyle, opts.MutedStyle
		if pane == opts.ActivePane {
			box, title = opts.PaneStyle, opts.TitleStyle
		}
		paneOpts := paneOptions{size: size, heading: heading, box: box, title: title}
		if pane == LayerPane {
			paneOpts.content = List(layerListOptions(opts, paneInterior(size, box)))
		}
		return renderPane(paneOpts)
	}

	preview := renderPreviewControls(opts, l)
	gap := lipgloss.NewStyle().Width(l.gap).Height(opts.Height).Render(strings.Repeat(" ", l.gap))
	return lipgloss.JoinHorizontal(lipgloss.Top,
		selectable(LayerPane, l.columns[0], "Layer selection"), gap,
		selectable(CategoryPane, l.columns[1], "Category option"), gap,
		selectable(FeaturePane, l.columns[2], "Feature option"), gap,
		preview,
	)
}

func renderPreviewControls(opts ScaffoldOptions, l scaffoldLayout) string {
	preview := paneOptions{size: l.preview, heading: "Preview", box: opts.PaneStyle, title: opts.TitleStyle}
	if !l.split {
		return renderPane(preview)
	}
	controls := paneOptions{
		size: l.controls, heading: "Controls/color picker",
		box: opts.InactivePaneStyle, title: opts.MutedStyle,
	}
	if opts.ActivePane == ControlsPane {
		controls.box, controls.title = opts.PaneStyle, opts.TitleStyle
	}

	if l.columns[3].interior() == l.columns[3] {
		// Without an outer outline, keep a plain divider between the sections.
		preview.box = preview.box.Border(lipgloss.RoundedBorder(), false)
		controls.box = controls.box.Border(lipgloss.RoundedBorder(), false)
		divider := lipgloss.NewStyle().Foreground(controls.box.GetBorderTopForeground()).
			Render(strings.Repeat("─", l.columns[3].width))
		return strings.Join([]string{renderPane(preview), divider, renderPane(controls)}, "\n")
	}

	// The upper section supplies the single shared divider, styled with Controls.
	border := lipgloss.RoundedBorder()
	border.BottomLeft, border.BottomRight = "├", "┤"
	preview.box = preview.box.Border(border).
		BorderBottomForeground(controls.box.GetBorderTopForeground())
	controls.box = controls.box.BorderTop(false)
	return renderPane(preview) + "\n" + renderPane(controls)
}

func paneBox(size paneSize, style lipgloss.Style) lipgloss.Style {
	box := style.Padding(0)
	if size.width < box.GetHorizontalFrameSize()+1 || size.height < box.GetVerticalFrameSize()+1 {
		// Drop outlines when they would leave no room for a heading.
		box = box.Border(lipgloss.RoundedBorder(), false)
	}
	return box
}

func paneInterior(size paneSize, style lipgloss.Style) paneSize {
	box := paneBox(size, style)
	return paneSize{max(0, size.width-box.GetHorizontalFrameSize()), max(0, size.height-box.GetVerticalFrameSize())}
}

func renderPane(opts paneOptions) string {
	if opts.size.width <= 0 || opts.size.height <= 0 {
		return ""
	}
	box := paneBox(opts.size, opts.box)
	interior := paneInterior(opts.size, opts.box)

	rows := make([]string, interior.height)
	inset := 0
	if interior.width >= 3 {
		inset = 1
	}
	rows[0] = lipgloss.PlaceHorizontal(interior.width, lipgloss.Left,
		strings.Repeat(" ", inset)+opts.title.Render(ansi.Truncate(opts.heading, interior.width-2*inset, "")))
	for i, line := range strings.Split(opts.content, "\n") {
		if i+1 >= len(rows) {
			break
		}
		rows[i+1] = ansi.Truncate(line, interior.width, "")
	}
	return box.Width(opts.size.width).Height(opts.size.height).
		MaxWidth(opts.size.width).MaxHeight(opts.size.height).
		Render(strings.Join(rows, "\n"))
}
