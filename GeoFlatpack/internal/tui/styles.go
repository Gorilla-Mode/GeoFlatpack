package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
)

type styles struct {
	frame   lipgloss.Style
	title   lipgloss.Style
	message lipgloss.Style
	help    help.Styles
}

func newStyles() styles {
	keyStyle := lipgloss.NewStyle().Bold(true)
	dim := lipgloss.NewStyle().Faint(true)
	return styles{
		frame:   lipgloss.NewStyle().Padding(1, 2),
		title:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
		message: lipgloss.NewStyle(),
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
