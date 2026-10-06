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
	ListActive
	ListAdd
)

// ListItem supplies presentation data for a layer, category, or feature option.
type ListItem struct {
	Name     string
	Detail   string
	Status   ListStatus
	Children []string
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
	Width         int
	Height        int
	Items         []ListItem
	Selected      int
	HideSelection bool
	FirstVisible  int
	Styles        ListStyles
	EmptyText     string
}

const (
	listItemHeight = 2
	listItemGap    = 1
)

func listHeight(item ListItem) int {
	if item.Detail == "" && len(item.Children) > 0 {
		return 1 + len(item.Children)
	}
	return listItemHeight + len(item.Children)
}

func listEnd(items []ListItem, first, height int) (end, used int) {
	end, used = first+1, listHeight(items[first])
	for end < len(items) && used+listItemGap+listHeight(items[end]) <= height {
		used += listItemGap + listHeight(items[end])
		end++
	}
	return end, used
}

func listOverflowHeight(opts ListOptions) int {
	if opts.Height < 2 {
		return 0 // Keep at least the selected item's name visible.
	}
	used := -listItemGap
	for _, item := range opts.Items {
		used += listItemGap + listHeight(item)
		if used > opts.Height {
			return 1
		}
	}
	return 0
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
	selected := MoveListSelection(opts.Selected, 0, len(opts.Items))
	height := opts.Height - listOverflowHeight(opts)
	if listHeight(opts.Items[selected]) > height {
		return selected
	}
	first := min(max(0, opts.FirstVisible), selected)
	used := listHeight(opts.Items[selected])
	for i := first; i < selected; i++ {
		used += listHeight(opts.Items[i]) + listItemGap
	}
	for used > height && first < selected {
		used -= listHeight(opts.Items[first]) + listItemGap
		first++
	}
	// Backfill when the list ends before the available space is exhausted.
	end, used := listEnd(opts.Items, first, height)
	if end == len(opts.Items) {
		for first > 0 && used+listItemGap+listHeight(opts.Items[first-1]) <= height {
			first--
			used += listItemGap + listHeight(opts.Items[first])
		}
	}
	return first
}

// List renders filled items with optional tree children and one unfilled gap.
// Overflow clips the last visible item and reserves a bottom row for dots.
// Selection remains highlighted unless the caller hides it.
func List(opts ListOptions) string {
	if opts.Width <= 0 || opts.Height <= 0 {
		return ""
	}
	if len(opts.Items) == 0 {
		return opts.Styles.Name.Render(ansi.Truncate(opts.EmptyText, opts.Width, "…"))
	}
	first := ListWindow(opts)
	overflow := listOverflowHeight(opts)
	height := opts.Height - overflow
	selected := MoveListSelection(opts.Selected, 0, len(opts.Items))
	rows := make([]string, 0, opts.Height)
	for i := first; i < len(opts.Items) && len(rows) < height; i++ {
		if i > first {
			rows = append(rows, strings.Repeat(" ", opts.Width))
		}
		if len(rows) >= height {
			break
		}
		itemRows := strings.Split(renderListItem(opts, opts.Items[i], !opts.HideSelection && i == selected), "\n")
		rows = append(rows, itemRows[:min(height-len(rows), len(itemRows))]...)
	}
	if overflow > 0 {
		for len(rows) < height {
			rows = append(rows, strings.Repeat(" ", opts.Width))
		}
		rows = append(rows, opts.Styles.Detail.Inline(true).Render(ansi.Truncate("...", opts.Width, "")))
	}
	return strings.Join(rows, "\n")
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
	case ListAdd:
		indicator = "+"
	case ListIncomplete:
		indicator, indicatorStyle = "●", opts.Styles.Incomplete
	case ListComplete, ListActive:
		indicator, indicatorStyle = "●", opts.Styles.Complete
	}
	// Embedded ANSI resets must not override the item's colors or fill.
	label := ansi.Truncate(listText(item.Name), max(0, opts.Width-2), "…")
	first := indicatorStyle.Background(background).Render(indicator) + fill.Render(" ") + name.Render(label)
	detail := opts.Styles.Detail.Background(background).Inline(true)
	rows := []string{ansi.Truncate(first, opts.Width, "")}
	if item.Detail != "" || len(item.Children) == 0 {
		rows = append(rows, detail.Render(ansi.Truncate("  "+listText(item.Detail), opts.Width, "")))
	}
	for i, child := range item.Children {
		branch := "  ├── "
		if i == len(item.Children)-1 {
			branch = "  └── "
		}
		rows = append(rows, detail.Render(ansi.Truncate(branch+listText(child), opts.Width, "…")))
	}
	return fill.Width(opts.Width).Height(listHeight(item)).Render(strings.Join(rows, "\n"))
}

func listText(value string) string {
	return strings.NewReplacer("\n", `\n`, "\r", `\r`, "\t", `\t`).Replace(ansi.Strip(value))
}
