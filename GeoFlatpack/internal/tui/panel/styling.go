package panel

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func stylingListOptions(opts ScaffoldOptions, interior paneSize) ListOptions {
	return ListOptions{
		Width: interior.width - 2*listInset(interior), Height: listContentHeight(interior),
		Items: opts.Styling, Selected: opts.SelectedStyling, FirstVisible: opts.FirstVisibleStyling,
		HideSelection: opts.HideStylingSelection, Styles: opts.ListStyles, EmptyText: opts.StylingEmptyText,
	}
}

// StylingWindow uses the same clipping and margins as the other list panes.
func StylingWindow(opts ScaffoldOptions) int {
	l := newScaffoldLayout(opts.Width, opts.Height)
	box := opts.InactivePaneStyle
	if opts.ActivePane == FeatureStylingPane {
		box = opts.PaneStyle
	}
	return ListWindow(stylingListOptions(opts, paneInterior(l.columns[3], box)))
}

type lowerLayout struct {
	box                       lipgloss.Style
	interior, stack, controls paneSize
	divided                   bool
}

func newLowerLayout(opts ScaffoldOptions, l scaffoldLayout) lowerLayout {
	box := opts.InactivePaneStyle
	if opts.ActivePane == ControlsPane {
		box = opts.PaneStyle
	}
	if l.columns[4].interior() == l.columns[4] {
		box = box.Border(lipgloss.RoundedBorder(), false)
	} else {
		box = box.BorderTop(false)
	}
	lower := lowerLayout{box: box, interior: paneInterior(l.controls, box)}
	lower.controls = lower.interior
	legacySize := paneSize{l.legacyRightWidth, l.controls.height}
	w := max(0, (paneInterior(legacySize, box).width-1)/3)
	stack := paneSize{w, lower.interior.height}
	controls := paneSize{lower.interior.width - w - 1, lower.interior.height}
	if opts.ShowStack && w-2*listInset(stack) >= 3 && controls.width-2*listInset(controls) >= 3 {
		lower.divided = true
		lower.stack, lower.controls = stack, controls
	}
	return lower
}

// InfoDimensions returns the display-only description area's content size.
func InfoDimensions(opts ScaffoldOptions) (width, height int) {
	l := newScaffoldLayout(opts.Width, opts.Height)
	if !l.split {
		return 0, 0
	}
	lower := newLowerLayout(opts, l)
	if !lower.divided {
		return 0, 0
	}
	interior := paneInterior(l.preview, opts.PaneStyle)
	interior.width = lower.stack.width
	return max(0, interior.width-2*listInset(interior)), max(0, interior.height-1-listHeadingGap(interior))
}

func stackListOptions(opts ScaffoldOptions, interior paneSize) ListOptions {
	return ListOptions{
		Width: interior.width - 2*listInset(interior), Height: max(0, interior.height-1-listHeadingGap(interior)),
		Items: opts.Stack, Selected: opts.ActiveStackLayer, FirstVisible: opts.StackFirstVisible,
		Styles: opts.ListStyles, EmptyText: opts.StackEmptyText,
	}
}

// StackWindow keeps the active layer visible without adding a focus target.
func StackWindow(opts ScaffoldOptions) int {
	l := newScaffoldLayout(opts.Width, opts.Height)
	lower := newLowerLayout(opts, l)
	if !l.split || !lower.divided {
		return opts.StackFirstVisible
	}
	return ListWindow(stackListOptions(opts, lower.stack))
}

func renderStackControls(opts ScaffoldOptions, l scaffoldLayout, preview paneOptions) string {
	lower := newLowerLayout(opts, l)
	plain := lipgloss.NewStyle()
	title := opts.MutedStyle
	if opts.ActivePane == ControlsPane {
		title = opts.TitleStyle
	}
	controls := paneOptions{
		size: lower.controls, heading: "Controls", box: plain, title: title,
	}
	populateControls(&controls, opts, lower.controls)
	content := renderPane(controls)
	borderStyle := lipgloss.NewStyle().Foreground(lower.box.GetBorderTopForeground())
	if lower.divided {
		stack := renderPane(paneOptions{
			size: lower.stack, heading: "Layers", box: plain, title: opts.MutedStyle,
			content: List(stackListOptions(opts, lower.stack)), contentInset: listInset(lower.stack), contentGap: listHeadingGap(lower.stack),
		})
		separator := borderStyle.Render(strings.TrimSuffix(strings.Repeat("│\n", lower.interior.height), "\n"))
		content = lipgloss.JoinHorizontal(lipgloss.Top, stack, separator, content)
	}
	controlsView := paneBox(l.controls, lower.box).Width(l.controls.width).Height(l.controls.height).
		MaxWidth(l.controls.width).MaxHeight(l.controls.height).Render(content)
	outlined := l.columns[4].interior() != l.columns[4]
	if !outlined {
		preview.box = preview.box.Border(lipgloss.RoundedBorder(), false)
		divider := strings.Repeat("─", l.columns[4].width)
		if lower.divided {
			divider = replaceCell(divider, lower.stack.width, "┼")
		}
		return renderUpper(opts, l, lower, preview) + "\n" + borderStyle.Render(divider) + "\n" + controlsView
	}
	border := lipgloss.RoundedBorder()
	border.BottomLeft, border.BottomRight = "├", "┤"
	preview.box = preview.box.Border(border).BorderBottomForeground(lower.box.GetBorderTopForeground())
	previewRows := strings.Split(renderUpper(opts, l, lower, preview), "\n")
	if lower.divided {
		x := lower.stack.width + 1
		previewRows[0] = replaceCell(previewRows[0], x, lipgloss.NewStyle().Foreground(opts.InactivePaneStyle.GetBorderTopForeground()).Render("┬"))
		previewRows[len(previewRows)-1] = replaceCell(previewRows[len(previewRows)-1], x, borderStyle.Render("┼"))
		if paneBox(l.controls, lower.box).GetBorderBottom() {
			rows := strings.Split(controlsView, "\n")
			rows[len(rows)-1] = replaceCell(rows[len(rows)-1], x, borderStyle.Render("┴"))
			controlsView = strings.Join(rows, "\n")
		}
	}
	return strings.Join(previewRows, "\n") + "\n" + controlsView
}

func renderUpper(opts ScaffoldOptions, l scaffoldLayout, lower lowerLayout, preview paneOptions) string {
	if !lower.divided {
		return renderPane(preview)
	}
	interior := paneInterior(preview.size, preview.box)
	infoSize := paneSize{lower.stack.width, interior.height}
	width, height := InfoDimensions(opts)
	info := renderPane(paneOptions{
		size: infoSize, heading: "Info", box: lipgloss.NewStyle(), title: opts.MutedStyle,
		content: clippedInfo(opts.InfoContent, width, height, opts.ListStyles.Detail), contentInset: listInset(infoSize), contentGap: listHeadingGap(infoSize),
	})
	previewView := renderPane(paneOptions{size: paneSize{lower.controls.width, interior.height}, heading: "Preview", box: lipgloss.NewStyle(), title: opts.TitleStyle})
	separator := lipgloss.NewStyle().Foreground(opts.InactivePaneStyle.GetBorderTopForeground()).
		Render(strings.TrimSuffix(strings.Repeat("│\n", interior.height), "\n"))
	content := lipgloss.JoinHorizontal(lipgloss.Top, info, separator, previewView)
	return paneBox(preview.size, preview.box).Width(preview.size.width).Height(preview.size.height).
		MaxWidth(preview.size.width).MaxHeight(preview.size.height).Render(content)
}

func clippedInfo(content string, width, height int, style lipgloss.Style) string {
	if width <= 0 || height <= 0 || content == "" {
		return ""
	}
	rows := strings.Split(ansi.Wrap(content, width, ""), "\n")
	if len(rows) > height {
		rows = rows[:height]
		rows[height-1] = ansi.Truncate("...", width, "")
	}
	for i, row := range rows {
		rows[i] = style.Inline(true).Render(ansi.Truncate(row, width, ""))
	}
	return strings.Join(rows, "\n")
}

func replaceCell(row string, x int, cell string) string {
	return ansi.Cut(row, 0, x) + cell + ansi.Cut(row, x+1, ansi.StringWidth(row))
}
