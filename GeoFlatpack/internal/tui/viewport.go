package tui

import (
	"GeoFlatpack/internal/tui/panel"

	"github.com/charmbracelet/x/ansi"
)

func (m *Model) refreshViewport() {
	l := m.layout()
	m.viewport.SetWidth(l.contentWidth)
	m.viewport.SetHeight(l.contentHeight)
	if m.screen == scaffoldScreen && !m.help.ShowAll {
		m.firstVisibleLayer = panel.LayerWindow(m.scaffoldOptions(l))
		if categories := m.currentCategories(); categories != nil {
			categories.firstVisible = panel.CategoryWindow(m.scaffoldOptions(l))
		}
		m.viewport.SetContent("")
		return
	}
	content := m.body(l.contentWidth)

	if m.help.ShowAll {
		content = panel.KeyboardReference(m.detailKeys.FullHelp(), m.help.Styles, m.styles.muted)
	}
	// Wrap before setting content so scroll offsets refer to displayed rows,
	// including word boundaries, ANSI styles, and wide Unicode characters.
	m.viewport.SetContent(ansi.Wrap(content, l.contentWidth, ""))
}

func (m *Model) body(width int) string {
	if m.screen == scaffoldScreen {
		return ""
	}

	opts := panel.ProcessingOptions{
		State:        panel.Complete,
		Input:        m.Options.Input,
		Quitting:     m.quitting,
		Elapsed:      m.elapsed,
		Err:          m.err,
		Width:        width,
		TitleStyle:   m.styles.title,
		MutedStyle:   m.styles.muted,
		SuccessStyle: m.styles.success,
		FailureStyle: m.styles.failure,
		ElapsedStyle: m.styles.elapsed,
	}
	switch m.screen {
	case loadingScreen:
		opts.State = panel.Loading
		opts.Spinner = m.spinner.View()
	case failureScreen:
		opts.State = panel.Failure
	case completionScreen:
	case scaffoldScreen:
	}
	if m.session != nil {
		opts.Loaded = true
		for _, layer := range m.session.Layers() {
			opts.LayerNames = append(opts.LayerNames, layer.Name)
		}
	}
	return panel.Processing(opts)
}
