package panel

import (
	"charm.land/lipgloss/v2"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// ControlItem is a field or action. Value may contain a styled input cursor.
type ControlItem struct {
	Label, Value, Error        string
	Disabled, Editing, Numeric bool
	Removable                  bool
	Button                     int // 0: field, 1: decrease, 2: increase, 3: remove element
}

func controlsListOptions(opts ScaffoldOptions, size paneSize) ListOptions {
	items := make([]ListItem, len(opts.Controls))
	for i, c := range opts.Controls {
		items[i] = ListItem{Name: c.Label, Detail: c.Value}
		if c.Error != "" {
			items[i].Children = []string{c.Error}
		}
	}
	return ListOptions{Width: max(0, size.width-2*listInset(size)), Height: listContentHeight(size), Items: items,
		Selected: opts.SelectedControl, FirstVisible: opts.FirstVisibleControl, Styles: opts.ListStyles, EmptyText: opts.ControlsEmptyText}
}

// ControlsDimensions and ControlsWindow share the renderer's layout calculation.
func ControlsDimensions(opts ScaffoldOptions) (int, int) {
	l := newScaffoldLayout(opts.Width, opts.Height)
	if !l.split {
		return 0, 0
	}
	lower := newLowerLayout(opts, l)
	o := controlsListOptions(opts, lower.controls)
	return o.Width, o.Height
}

func ControlsWindow(opts ScaffoldOptions) int {
	l := newScaffoldLayout(opts.Width, opts.Height)
	if !l.split {
		return opts.FirstVisibleControl
	}
	return ListWindow(controlsListOptions(opts, newLowerLayout(opts, l).controls))
}

func populateControls(p *paneOptions, opts ScaffoldOptions, size paneSize) {
	p.contentInset, p.contentGap = listInset(size), listHeadingGap(size)
	p.content = renderControls(opts, size)
	if listFooterHeight(size) > 0 {
		p.footer = opts.ScrollHintStyle.Render(ansi.Truncate("↑/↓ Scroll", size.width, ""))
	}
}

func renderControls(opts ScaffoldOptions, size paneSize) string {
	o := controlsListOptions(opts, size)
	if o.Width <= 0 || o.Height <= 0 {
		return ""
	}
	if len(opts.Controls) == 0 {
		return opts.ListStyles.Name.Inline(true).Render(ansi.Truncate(o.EmptyText, o.Width, "…"))
	}
	height := o.Height - listOverflowHeight(o)
	rows := []string{}
	for i := ListWindow(o); i < len(opts.Controls) && len(rows) < height; i++ {
		if len(rows) > 0 {
			rows = append(rows, strings.Repeat(" ", o.Width))
		}
		c := opts.Controls[i]
		selected := i == opts.SelectedControl && !c.Disabled
		fill, label := opts.ListStyles.Item, opts.ListStyles.Name
		if selected {
			fill, label = opts.ListStyles.SelectedItem, opts.ListStyles.SelectedName
		}
		bg := fill.GetBackground()
		label = label.Background(bg).Inline(true)
		valueStyle := opts.ListStyles.Detail.Background(bg).Inline(true)
		if c.Disabled {
			label = label.Faint(true)
			valueStyle = valueStyle.Faint(true)
		}
		name := listText(c.Label)
		if c.Editing {
			name += " (editing)"
		}
		value := c.Value
		if c.Numeric {
			minus, plus := "[−]", "[+]"
			remove := "[×]"
			if selected && c.Button == 1 {
				minus = label.Render(minus)
			}
			if selected && c.Button == 2 {
				plus = label.Render(plus)
			}
			if selected && c.Button == 3 {
				remove = label.Render(remove)
			}
			buttons := minus + " " + plus
			if c.Removable {
				buttons += " " + remove
			}
			if selected && c.Button != 0 {
				value = buttons + " " + value
			} else {
				value += " " + buttons
			}
		}
		lines := []string{label.Render(ansi.Truncate(name, o.Width, "…")), valueStyle.Render(ansi.Truncate(value, o.Width, "…"))}
		if c.Error != "" {
			lines = append(lines, label.Foreground(lipgloss.Color("1")).Render(ansi.Truncate(listText(c.Error), o.Width, "…")))
		}
		for _, line := range lines {
			if len(rows) >= height {
				break
			}
			rows = append(rows, fill.Width(o.Width).Render(line))
		}
	}
	if listOverflowHeight(o) > 0 {
		for len(rows) < height {
			rows = append(rows, strings.Repeat(" ", o.Width))
		}
		rows = append(rows, opts.ListStyles.Detail.Inline(true).Render(ansi.Truncate("...", o.Width, "")))
	}
	return strings.Join(rows, "\n")
}
