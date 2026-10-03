package panel

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// ListStatus describes readiness independently of selection and keyboard focus.
type ListStatus int

const (
	ListUnopened ListStatus = iota
	ListIncomplete
	ListComplete
)

// ListItem supplies presentation data for a layer, category, or feature option.
type ListItem struct {
	Name   string
	Detail string
	Status ListStatus
}

// ListStyles separates persistent selection and readiness from pane focus.
type ListStyles struct {
	Item         lipgloss.Style
	SelectedItem lipgloss.Style
	Name         lipgloss.Style
	SelectedName lipgloss.Style
	Detail       lipgloss.Style
	Unopened     lipgloss.Style
	Incomplete   lipgloss.Style
	Complete     lipgloss.Style
}

// ListOptions describes the available content area and the list's visible state.
type ListOptions struct {
	Width        int
	Height       int
	Items        []ListItem
	Selected     int
	FirstVisible int
	Styles       ListStyles
	EmptyText    string
}

const (
	listItemHeight = 2
	listItemGap    = 1
)

func listCapacity(height int) int {
	// A very short list shows the selected item's name row.
	return max(1, (max(0, height)+listItemGap)/(listItemHeight+listItemGap))
}

// MoveListSelection moves without wrapping, returning zero for an empty list.
func MoveListSelection(selected, delta, count int) int {
	return min(max(0, selected+delta), max(0, count-1))
}

// ListWindow returns the first visible item needed to keep selection in view.
func ListWindow(opts ListOptions) int {
	if len(opts.Items) == 0 {
		return 0
	}
	capacity := listCapacity(opts.Height)
	selected := MoveListSelection(opts.Selected, 0, len(opts.Items))
	first := min(max(0, opts.FirstVisible), max(0, len(opts.Items)-capacity))
	if selected < first {
		first = selected
	} else if selected >= first+capacity {
		first = selected - capacity + 1
	}
	return first
}

// List renders two-row filled items separated by one unfilled row. Selection
// remains highlighted regardless of which pane owns keyboard focus.
func List(opts ListOptions) string {
	if opts.Width <= 0 || opts.Height <= 0 {
		return ""
	}
	if len(opts.Items) == 0 {
		return opts.Styles.Name.Render(ansi.Truncate(opts.EmptyText, opts.Width, "…"))
	}
	first := ListWindow(opts)
	end := min(len(opts.Items), first+listCapacity(opts.Height))
	selected := MoveListSelection(opts.Selected, 0, len(opts.Items))
	items := make([]string, 0, end-first)
	for i := first; i < end; i++ {
		items = append(items, renderListItem(opts, opts.Items[i], i == selected))
	}
	separator := "\n" + strings.Repeat(" ", opts.Width) + "\n"
	rows := strings.Split(strings.Join(items, separator), "\n")
	return strings.Join(rows[:min(opts.Height, len(rows))], "\n")
}

func renderListItem(opts ListOptions, item ListItem, selected bool) string {
	fill, name := opts.Styles.Item, opts.Styles.Name
	if selected {
		fill, name = opts.Styles.SelectedItem, opts.Styles.SelectedName
	}
	background := fill.GetBackground()
	name = name.Bold(selected).Background(background).Inline(true)
	indicator, indicatorStyle := "○", opts.Styles.Unopened
	switch item.Status {
	case ListIncomplete:
		indicator, indicatorStyle = "●", opts.Styles.Incomplete
	case ListComplete:
		indicator, indicatorStyle = "●", opts.Styles.Complete
	}
	// Embedded ANSI resets must not override the item's colors or fill.
	label := ansi.Truncate(ansi.Strip(item.Name), max(0, opts.Width-2), "…")
	first := indicatorStyle.Background(background).Render(indicator) + fill.Render(" ") + name.Render(label)
	second := opts.Styles.Detail.Background(background).Inline(true).Render("  " + ansi.Strip(item.Detail))
	content := ansi.Truncate(first, opts.Width, "") + "\n" + ansi.Truncate(second, opts.Width, "")
	return fill.Width(opts.Width).Height(listItemHeight).Render(content)
}
