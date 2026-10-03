package tui

import (
	"GeoFlatpack/internal/tui/panel"

	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
)

const (
	colorAccent                 = "6"
	colorBorder                 = "240"
	colorMuted                  = "244"
	colorSuccess                = "2"
	colorFailure                = "1"
	colorElapsedBackground      = "236"
	colorSelectedPane           = "252"
	colorInactivePane           = "248"
	colorListBackground         = "234"
	colorSelectedListBackground = "236"
	colorIncomplete             = "3"
)

type styles struct {
	frame   lipgloss.Style
	box     lipgloss.Style
	title   lipgloss.Style
	muted   lipgloss.Style
	success lipgloss.Style
	failure lipgloss.Style
	elapsed lipgloss.Style
	help    help.Styles

	pane              lipgloss.Style
	inactivePane      lipgloss.Style
	paneTitle         lipgloss.Style
	inactivePaneTitle lipgloss.Style
	list              panel.ListStyles
}

func newStyles() styles {
	keyStyle := lipgloss.NewStyle().
		Bold(true)

	dim := lipgloss.NewStyle().
		Faint(true)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorBorder)).
		Padding(0, 1)

	return styles{
		frame: lipgloss.NewStyle().
			Padding(1, 2),

		box: box,

		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorAccent)),

		muted: dim.Foreground(lipgloss.Color(colorMuted)),

		success: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorSuccess)),

		failure: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorFailure)),

		elapsed: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorAccent)).
			Background(lipgloss.Color(colorElapsedBackground)).
			Padding(0, 1),

		help: help.Styles{
			Ellipsis:       dim,
			ShortKey:       keyStyle,
			ShortDesc:      dim,
			ShortSeparator: dim,
			FullKey:        keyStyle,
			FullDesc:       dim,
			FullSeparator:  dim,
		},

		pane:         box.BorderForeground(lipgloss.Color(colorSelectedPane)),
		inactivePane: box.BorderForeground(lipgloss.Color(colorInactivePane)),
		paneTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorSelectedPane)),
		inactivePaneTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorInactivePane)),
		list: panel.ListStyles{
			Item:         lipgloss.NewStyle().Background(lipgloss.Color(colorListBackground)),
			SelectedItem: lipgloss.NewStyle().Background(lipgloss.Color(colorSelectedListBackground)),
			Name:         lipgloss.NewStyle().Foreground(lipgloss.Color(colorInactivePane)),
			SelectedName: lipgloss.NewStyle().Foreground(lipgloss.Color(colorSelectedPane)).Bold(true),
			Detail:       lipgloss.NewStyle().Foreground(lipgloss.Color(colorInactivePane)),
			Unopened:     lipgloss.NewStyle().Foreground(lipgloss.Color(colorInactivePane)),
			Incomplete:   lipgloss.NewStyle().Foreground(lipgloss.Color(colorIncomplete)),
			Complete:     lipgloss.NewStyle().Foreground(lipgloss.Color(colorSuccess)).Bold(true),
		},
	}
}
