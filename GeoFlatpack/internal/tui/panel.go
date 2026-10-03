package tui

import (
	"GeoFlatpack/internal/tui/panel"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func (m *Model) panel() string {
	l := m.layout()
	var sections []string
	scaffold := m.screen == scaffoldScreen && !m.help.ShowAll

	if l.headerHeight > 0 {
		header := panel.Header(l.width, m.styles.title)
		if m.screen == scaffoldScreen && (m.width < 120 || m.height < 24) {
			warning := m.styles.muted.Render("Expand terminal")
			if lipgloss.Width(header)+3+lipgloss.Width(warning) <= l.width {
				header += " • " + warning
			} else {
				header = ansi.Truncate(warning, l.width, "")
			}
		}
		sections = append(sections, header)
	}

	if l.bodyHeight > 0 {
		if l.gap > 0 {
			sections = append(sections, "")
		}

		if scaffold {
			sections = append(sections, panel.Scaffold(m.scaffoldOptions(l)))
		} else {
			sections = append(sections, panel.Body(panel.BodyOptions{
				Style:         l.bodyStyle,
				TitleStyle:    m.styles.title,
				Width:         l.width,
				Height:        l.bodyHeight,
				ContentWidth:  l.contentWidth,
				ContentHeight: l.contentHeight,
				HeadingHeight: l.headingHeight,
				ShowHelp:      m.help.ShowAll,
				Content:       m.viewport.View(),
			}))
		}
		if l.footerGap > 0 {
			sections = append(sections, "")
		}
	}

	sections = append(sections, panel.Footer(panel.FooterOptions{
		Style:      l.box,
		MutedStyle: m.styles.muted,
		Width:      l.width,
		Height:     l.footerHeight,
		Help:       m.help,
		Primary:    []key.Binding{m.keys.Help, m.keys.Quit, m.keys.Select},
		Hints:      m.keys.ShortHelp()[3:],
		Scrollable: !scaffold && m.viewport.TotalLineCount() > m.viewport.Height(),
	}))

	// Lip Gloss v2 dimensions include both padding and borders.
	return l.frame.PaddingBottom(0).Width(m.width).Height(m.height).Render(strings.Join(sections, "\n"))
}

func (m *Model) scaffoldOptions(l panelLayout) panel.ScaffoldOptions {
	return panel.ScaffoldOptions{
		Width: l.width, Height: l.bodyHeight, ActivePane: m.activePane,
		PaneStyle: m.styles.pane, InactivePaneStyle: m.styles.inactivePane,
		TitleStyle: m.styles.paneTitle, MutedStyle: m.styles.inactivePaneTitle,
		ListStyles: m.styles.list,
		Layers:     m.layers, SelectedLayer: m.selectedLayer, FirstVisibleLayer: m.firstVisibleLayer,
	}
}
