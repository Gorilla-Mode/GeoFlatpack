package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
)

type styles struct {
	frame   lipgloss.Style
	title   lipgloss.Style
	message lipgloss.Style
	helpBox lipgloss.Style
	help    help.Styles
}

func newStyles() styles {
	keyStyle := lipgloss.NewStyle().
		Bold(true)

	dim := lipgloss.NewStyle().
		Faint(true)

	helpBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	return styles{
		frame:   lipgloss.NewStyle().Padding(1, 2),
		title:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
		message: lipgloss.NewStyle(),
		helpBox: helpBox,
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
