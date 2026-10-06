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
	FeaturesPane
	FeatureStylingPane
	ControlsPane
	PaneCount
)

// ScaffoldOptions separates the body dimensions and focus styles from content.
type ScaffoldOptions struct {
	Width                int
	Height               int
	ActivePane           Pane
	PaneStyle            lipgloss.Style
	InactivePaneStyle    lipgloss.Style
	TitleStyle           lipgloss.Style
	MutedStyle           lipgloss.Style
	ScrollHintStyle      lipgloss.Style
	ListStyles           ListStyles
	Layers               []ListItem
	SelectedLayer        int
	FirstVisibleLayer    int
	Categories           []ListItem
	SelectedCategory     int
	FirstVisibleCategory int
	CategoryEmptyText    string
	Features             []ListItem
	SelectedFeature      int
	FirstVisibleFeature  int
	ChosenFeature        int
	FeatureChosen        bool
	FeatureEmptyText     string
	Styling              []ListItem
	StylingHeading       string
	StylingEmptyText     string
	SelectedStyling      int
	FirstVisibleStyling  int
	HideStylingSelection bool
	ShowStack            bool
	Stack                []ListItem
	ActiveStackLayer     int
	StackFirstVisible    int
	StackEmptyText       string
	InfoContent          string
	Controls             []ControlItem
	SelectedControl      int
	FirstVisibleControl  int
	ControlsEmptyText    string
	ColorPicker          *ColorPickerOptions
	PreviewContent       string
	PreviewWarning       string
	PreviewImage         bool
}

type paneSize struct {
	width  int
	height int
}

type scaffoldLayout struct {
	gap              int
	columns          [5]paneSize
	preview          paneSize
	controls         paneSize
	legacyRightWidth int
	split            bool
}

func newScaffoldLayout(width, height int) scaffoldLayout {
	width, height = max(0, width), max(0, height)
	l := scaffoldLayout{gap: min(1, width/4)}
	available := width - 4*l.gap
	l.legacyRightWidth = available - 4*(available*16/100)
	used := 0
	for i := 0; i < 4; i++ {
		w := available * 12 / 100
		l.columns[i] = paneSize{w, height}
		used += w
	}
	l.columns[4] = paneSize{available - used, height}

	l.preview = l.columns[4]
	interior := l.preview.interior()
	if interior.width > 0 && interior.height >= 3 {
		l.split = true
		// Reserve the shared divider. Give Layers/Controls 60% of the rows,
		// about 20% more height than the previous even split.
		remaining := interior.height - 1
		controlsHeight := min(remaining-1, (remaining*3+2)/5)
		l.preview.height = remaining - controlsHeight
		l.controls = paneSize{l.preview.width, controlsHeight}
		if interior != l.columns[4] {
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
	size         paneSize
	heading      string
	box          lipgloss.Style
	title        lipgloss.Style
	content      string
	footer       string
	contentInset int
	contentGap   int
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
		interior := paneInterior(size, box)
		paneOpts := paneOptions{size: size, heading: heading, box: box, title: title}
		paneOpts.contentInset, paneOpts.contentGap = listInset(interior), listHeadingGap(interior)
		if listFooterHeight(interior) > 0 {
			paneOpts.footer = opts.ScrollHintStyle.Render(ansi.Truncate("↑/↓ Scroll", interior.width, ""))
		}
		if pane == LayerPane {
			paneOpts.content = List(layerListOptions(opts, interior))
		} else if pane == CategoryPane {
			paneOpts.content = List(categoryListOptions(opts, interior))
		} else if pane == FeaturesPane {
			paneOpts.content = List(featureListOptions(opts, interior))
		} else if pane == FeatureStylingPane {
			paneOpts.content = List(stylingListOptions(opts, interior))
		}
		return renderPane(paneOpts)
	}

	preview := renderPreviewControls(opts, l)
	stylingHeading := opts.StylingHeading
	if stylingHeading == "" {
		stylingHeading = "Feature styling"
	}
	gap := lipgloss.NewStyle().Width(l.gap).Height(opts.Height).Render(strings.Repeat(" ", l.gap))
	return lipgloss.JoinHorizontal(lipgloss.Top,
		selectable(LayerPane, l.columns[0], "Layer selection"), gap,
		selectable(CategoryPane, l.columns[1], "Category"), gap,
		selectable(FeaturesPane, l.columns[2], "Features"), gap,
		selectable(FeatureStylingPane, l.columns[3], stylingHeading), gap,
		preview,
	)
}

func renderPreviewControls(opts ScaffoldOptions, l scaffoldLayout) string {
	preview := paneOptions{size: l.preview, heading: "Preview", box: opts.PaneStyle, title: opts.TitleStyle}
	populatePreview(&preview, opts, paneInterior(l.preview, preview.box))
	if !l.split {
		return renderPane(preview)
	}
	if opts.ShowStack {
		return renderStackControls(opts, l, preview)
	}
	controls := paneOptions{
		size: l.controls, heading: "Controls",
		box: opts.InactivePaneStyle, title: opts.MutedStyle,
	}
	if opts.ActivePane == ControlsPane {
		controls.box, controls.title = opts.PaneStyle, opts.TitleStyle
	}

	if l.columns[4].interior() == l.columns[4] {
		// Without an outer outline, keep a plain divider between the sections.
		preview.box = preview.box.Border(lipgloss.RoundedBorder(), false)
		controls.box = controls.box.Border(lipgloss.RoundedBorder(), false)
		populateControls(&controls, opts, paneInterior(l.controls, controls.box))
		divider := lipgloss.NewStyle().Foreground(controls.box.GetBorderTopForeground()).
			Render(strings.Repeat("─", l.columns[4].width))
		return strings.Join([]string{renderPane(preview), divider, renderPane(controls)}, "\n")
	}

	// The upper section supplies the single shared divider, styled with Controls.
	border := lipgloss.RoundedBorder()
	border.BottomLeft, border.BottomRight = "├", "┤"
	preview.box = preview.box.Border(border).
		BorderBottomForeground(controls.box.GetBorderTopForeground())
	controls.box = controls.box.BorderTop(false)
	populateControls(&controls, opts, paneInterior(l.controls, controls.box))
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
	contentEnd := len(rows)
	if opts.footer != "" {
		contentEnd--
		rows[contentEnd] = opts.footer
	}
	for i, line := range strings.Split(opts.content, "\n") {
		y := i + 1 + opts.contentGap
		if y >= contentEnd {
			break
		}
		rows[y] = strings.Repeat(" ", opts.contentInset) + ansi.Truncate(line, interior.width-2*opts.contentInset, "")
	}
	return box.Width(opts.size.width).Height(opts.size.height).
		MaxWidth(opts.size.width).MaxHeight(opts.size.height).
		Render(strings.Join(rows, "\n"))
}
