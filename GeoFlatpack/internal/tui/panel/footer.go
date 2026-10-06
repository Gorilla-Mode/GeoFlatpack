package panel

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// FooterOptions supplies the footer layout, ordered bindings, and scroll hint.
type FooterOptions struct {
	Style      lipgloss.Style
	MutedStyle lipgloss.Style
	Width      int
	Height     int
	Help       help.Model
	Primary    []key.Binding
	Hints      []key.Binding
	Scrollable bool
}

// Footer renders keyboard hints inside the footer box.
func Footer(opts FooterOptions) string {
	return opts.Style.Width(opts.Width).Height(opts.Height).Render(footer(opts))
}

func footer(opts FooterOptions) string {
	footerWidth := max(1, opts.Width-opts.Style.GetHorizontalFrameSize())
	helper := opts.Help
	helper.SetWidth(0)
	primary := opts.Primary

	// Keep the normal order, omitting leading hints only when Select cannot fit.
	for len(primary) > 1 && lipgloss.Width(helper.ShortHelpView(primary)) > footerWidth {
		primary = primary[1:]
	}

	primaryWidth := lipgloss.Width(helper.ShortHelpView(primary))
	if primaryWidth > footerWidth {
		return ansi.Truncate(helper.ShortHelpView(primary), footerWidth, "")
	}

	hints := append(primary, opts.Hints...)

	width := footerWidth
	var scrollHint string
	if opts.Scrollable {
		scrollHint = "↑/↓ PgUp/PgDn Scroll"
		if primaryWidth+3+lipgloss.Width(scrollHint) > width {
			scrollHint = "↑↓ Scroll"
		}

		if primaryWidth+3+lipgloss.Width(scrollHint) <= width {
			scrollHint = opts.MutedStyle.Render(scrollHint)
			width -= lipgloss.Width(scrollHint) + 3
		} else {
			scrollHint = ""
		}
	}

	helper.SetWidth(width)
	otherHints := helper.ShortHelpView(hints)
	if scrollHint != "" && otherHints != "" {
		scrollHint += " • "
	}

	return ansi.Truncate(scrollHint+otherHints, footerWidth, "")
}
