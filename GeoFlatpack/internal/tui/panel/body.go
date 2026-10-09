package panel

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// BodyOptions supplies the body layout, heading styles, and viewport content.
type BodyOptions struct {
	Style         lipgloss.Style
	TitleStyle    lipgloss.Style
	Width         int
	Height        int
	ContentWidth  int
	ContentHeight int
	HeadingHeight int
	ShowHelp      bool
	Content       string
}

// Body renders the heading and visible content inside the body box.
func Body(opts BodyOptions) string {
	var body []string
	if opts.HeadingHeight > 0 {
		heading := bodyHeading(opts.ContentWidth, opts.ShowHelp, opts.TitleStyle)
		if opts.HeadingHeight > 1 {
			heading += "\n"
		}

		body = append(body, heading)
	}

	if opts.ContentHeight > 0 {
		body = append(body, opts.Content)
	}

	return opts.Style.Width(opts.Width).Height(opts.Height).Render(strings.Join(body, "\n"))
}

func bodyHeading(width int, showHelp bool, title lipgloss.Style) string {
	heading := "Processing file"
	if showHelp {
		heading = "Keyboard reference"
	}

	return title.Render(ansi.Truncate(heading, width, ""))
}
