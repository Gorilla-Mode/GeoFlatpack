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
	if listHeight(opts.Items[selected]) > opts.Height {
		return selected
	}
	first := min(max(0, opts.FirstVisible), selected)
	used := listHeight(opts.Items[selected])
	for i := first; i < selected; i++ {
		used += listHeight(opts.Items[i]) + listItemGap
	}
	for used > opts.Height && first < selected {
		used -= listHeight(opts.Items[first]) + listItemGap
		first++
	}
	// Backfill when the list ends before the available space is exhausted.
	end, used := listEnd(opts.Items, first, opts.Height)
	if end == len(opts.Items) {
		for first > 0 && used+listItemGap+listHeight(opts.Items[first-1]) <= opts.Height {
			first--
			used += listItemGap + listHeight(opts.Items[first])
		}
	}
	return first
}

// List renders filled items with optional tree children and one unfilled gap.
// Selection remains highlighted unless the caller hides it.
func List(opts ListOptions) string {
	if opts.Width <= 0 || opts.Height <= 0 {
		return ""
	}
	if len(opts.Items) == 0 {
		return opts.Styles.Name.Render(ansi.Truncate(opts.EmptyText, opts.Width, "…"))
	}
	first := ListWindow(opts)
	end, _ := listEnd(opts.Items, first, opts.Height)
	selected := MoveListSelection(opts.Selected, 0, len(opts.Items))
	items := make([]string, 0, end-first)
	for i := first; i < end; i++ {
		items = append(items, renderListItem(opts, opts.Items[i], !opts.HideSelection && i == selected))
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
	case ListComplete, ListActive:
		indicator, indicatorStyle = "●", opts.Styles.Complete
	}
	// Embedded ANSI resets must not override the item's colors or fill.
	label := ansi.Truncate(listText(item.Name), max(0, opts.Width-2), "…")
	first := indicatorStyle.Background(background).Render(indicator) + fill.Render(" ") + name.Render(label)
	detail := opts.Styles.Detail.Background(background).Inline(true)
	rows := []string{
		ansi.Truncate(first, opts.Width, ""),
		detail.Render(ansi.Truncate("  "+listText(item.Detail), opts.Width, "")),
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
