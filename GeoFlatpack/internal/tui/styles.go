package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
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
}

func newStyles() styles {
	keyStyle := lipgloss.NewStyle().
		Bold(true)

	dim := lipgloss.NewStyle().
		Faint(true)

	return styles{
		frame: lipgloss.NewStyle().
			Padding(1, 2),

		box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1),

		title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6")),

		muted: dim.Foreground(lipgloss.Color("244")),

		success: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("2")),

		failure: lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")),

		elapsed: lipgloss.NewStyle().
			Foreground(lipgloss.Color("6")).
			Background(lipgloss.Color("236")).
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
	}
}
