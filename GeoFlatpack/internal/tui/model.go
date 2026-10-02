// Package tui provides the terminal frontend scaffold.
package tui

import (
	"GeoFlatpack/internal/app"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Model retains workflow options for future processing screens.
type Model struct {
	Options    app.Options
	help       help.Model
	keys       keyMap
	detailKeys detailKeyMap
	styles     styles
	width      int
	height     int
}

// NewModel creates the scaffold without preparing or processing input files.
func NewModel(opts app.Options) Model {
	s := newStyles()
	h := help.New()
	h.Styles = s.help
	h.SetWidth(80 - s.frame.GetHorizontalFrameSize())
	return Model{
		Options:    opts,
		help:       h,
		keys:       newKeyMap(),
		detailKeys: newDetailKeyMap(),
		styles:     s,
		width:      80,
		height:     24,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = max(1, msg.Width), max(1, msg.Height)
		m.help.SetWidth(max(1, m.width-m.styles.frame.GetHorizontalFrameSize()))
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			if m.help.ShowAll {
				m.keys.Help.SetHelp("?", "hide help")
			} else {
				m.keys.Help.SetHelp("?", "help")
			}
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	width := max(1, m.width-m.styles.frame.GetHorizontalFrameSize())
	height := max(1, m.height-m.styles.frame.GetVerticalFrameSize())

	helpText := m.styles.helpBox.
		Width(max(1, width-m.styles.helpBox.GetHorizontalFrameSize())).
		Render(m.help.View(helpKeyMap{m.keys, m.detailKeys}))

	body := strings.Join([]string{
		m.styles.title.Width(width).Render("GeoFlatpack"),
		m.styles.message.Width(width).Render(
			"TUI scaffold. File processing is not connected yet.",
		),
	}, "\n\n")

	bodyHeight := max(0, height-lipgloss.Height(helpText)+1)

	content := lipgloss.NewStyle().
		Width(width).
		Height(bodyHeight).
		Render(body) + helpText

	view := tea.NewView(
		m.styles.frame.
			Width(width).
			Height(height).
			Render(content),
	)

	view.AltScreen = true
	return view
}
