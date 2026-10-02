package tui

import (
	"GeoFlatpack/internal/tui/panel"
	"strings"

	"charm.land/bubbles/v2/key"
)

func (m *Model) panel() string {
	l := m.layout()
	var sections []string

	if l.headerHeight > 0 {
		sections = append(sections, panel.Header(l.width, m.styles.title))
	}

	if l.bodyHeight > 0 {
		if l.gap > 0 {
			sections = append(sections, "")
		}

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
		Scrollable: m.viewport.TotalLineCount() > m.viewport.Height(),
	}))

	// Lip Gloss v2 dimensions include both padding and borders.
	return l.frame.PaddingBottom(0).Width(m.width).Height(m.height).Render(strings.Join(sections, "\n"))
}
