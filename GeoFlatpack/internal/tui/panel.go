package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/tree"
	"github.com/charmbracelet/x/ansi"
)

type panelLayout struct {
	frame         lipgloss.Style
	box           lipgloss.Style
	bodyStyle     lipgloss.Style
	width         int
	height        int
	header        string
	heading       string
	gap           int
	footerRows    int
	footerHeight  int
	bodyHeight    int
	contentWidth  int
	contentHeight int
}

func (m Model) layout() panelLayout {
	frame := lipgloss.NewStyle()
	// Restore the original outer padding when the body has room to use it.
	if m.width >= 24 && m.height >= 18 {
		frame = m.styles.frame
	} else if m.width >= 12 && m.height >= 8 {
		frame = frame.Padding(0, 1)
	}

	l := panelLayout{
		frame:      frame,
		width:      max(1, m.width-frame.GetHorizontalFrameSize()),
		height:     max(1, m.height-frame.GetVerticalFrameSize()),
		footerRows: 1,
	}

	// Both boxes use the same outline. Drop padding, then borders, before
	// sacrificing Select on a small terminal.
	if l.width >= lipgloss.Width("↵ Select")+2 && l.height >= 9 {
		l.box = m.styles.box
		if l.width < 12 {
			l.box = l.box.Padding(0)
		}
	}

	if l.height >= 6 {
		l.footerRows = 2
	}

	l.footerHeight = l.footerRows + l.box.GetVerticalFrameSize()
	if l.height >= 2 {
		l.header = m.styles.title.Render(ansi.Truncate("GeoFlatpack", l.width, ""))
	}

	if l.height >= 12 {
		l.gap = 1
	}

	headerHeight := 0
	if l.header != "" {
		headerHeight = 1
	}

	l.bodyHeight = max(0, l.height-headerHeight-l.footerHeight-2*l.gap)
	if m.help.ShowAll || m.screen != scaffoldScreen {
		l.bodyStyle = l.box
	}

	l.contentWidth = max(1, l.width-l.bodyStyle.GetHorizontalFrameSize())
	l.contentHeight = max(0, l.bodyHeight-l.bodyStyle.GetVerticalFrameSize())

	if l.contentHeight > 0 && (m.help.ShowAll || m.screen != scaffoldScreen) {
		heading := "Processing file"
		if m.help.ShowAll {
			heading = "Keyboard reference"
		}
		l.heading = m.styles.title.Render(ansi.Truncate(heading, l.contentWidth, ""))
		l.contentHeight--
		if l.contentHeight >= 7 {
			l.heading += "\n"
			l.contentHeight--
		}
	}
	return l
}

func (m *Model) refreshViewport() {
	l := m.layout()
	m.viewport.SetWidth(l.contentWidth)
	m.viewport.SetHeight(l.contentHeight)
	content := m.body(l.contentWidth)

	if m.help.ShowAll {
		content = m.expandedHelp()
	}
	// Wrap before setting content so scroll offsets refer to displayed rows,
	// including word boundaries, ANSI styles, and wide Unicode characters.
	m.viewport.SetContent(ansi.Wrap(content, l.contentWidth, ""))
}

func (m Model) panel() string {
	l := m.layout()
	var sections []string

	if l.header != "" {
		sections = append(sections, l.header)
	}

	if l.bodyHeight > 0 {
		if l.gap > 0 {
			sections = append(sections, "")
		}

		var body []string
		if l.heading != "" {
			body = append(body, l.heading)
		}

		if l.contentHeight > 0 {
			body = append(body, m.viewport.View())
		}

		sections = append(sections, l.bodyStyle.Width(l.width).Height(l.bodyHeight).Render(strings.Join(body, "\n")))
		if l.gap > 0 {
			sections = append(sections, "")
		}
	}

	sections = append(sections, l.box.Width(l.width).Height(l.footerHeight).Render(m.footer(l)))

	// Lip Gloss v2 dimensions include both padding and borders.
	return l.frame.Width(m.width).Height(m.height).Render(strings.Join(sections, "\n"))
}

func (m Model) body(width int) string {
	if m.screen == scaffoldScreen {
		return "TUI scaffold. Input loaded; styling controls are not connected yet."
	}

	var status string
	switch m.screen {
	case loadingScreen:
		message := "Processing input…"
		if m.quitting {
			message = "Finishing processing before exiting…"
		}
		status = m.spinner.View() + " " + message
	case failureScreen:
		status = m.styles.failure.Render(fmt.Sprintf("Could not process input\n%s", m.err))
	default:
		status = m.styles.success.Render("✓ Processing complete")
	}

	filename := filepath.Base(m.Options.Input)
	layers := tree.Root(ansi.Wrap(filename, width, "")).RootStyle(m.styles.title).
		EnumeratorStyle(m.styles.muted.PaddingRight(1)).
		IndenterStyle(m.styles.muted.PaddingRight(1))
	count := ""

	if m.session != nil {
		loaded := m.session.Layers()
		for _, layer := range loaded {
			layers.Child(ansi.Wrap(layer.Name, max(1, width-4), ""))
		}
		count = m.styles.muted.Render(fmt.Sprintf("Loaded layers: %d", len(loaded)))
	} else if m.screen == loadingScreen {
		layers.Child(m.styles.muted.Render("Reading layers…"))
	}

	parts := []string{
		status,
		m.styles.elapsed.Render(fmt.Sprintf("◷ Elapsed %.1f s", m.elapsed.Seconds())),
		"",
		m.styles.muted.Render(filepath.Dir(m.Options.Input)),
		layers.String(),
	}

	if count != "" {
		parts = append(parts, count)
	}
	switch m.screen {
	case completionScreen:
		parts = append(parts, "", m.styles.title.Render("↵ Continue"))
	default:
		panic("unhandled default case")
	}
	return strings.Join(parts, "\n")
}

func (m Model) expandedHelp() string {
	var rows []string
	// Render complete rows before wrapping. The help bubble's column truncation
	// would otherwise discard the entire reference on a narrow terminal.
	for _, group := range m.detailKeys.FullHelp() {
		for _, binding := range group {
			if !binding.Enabled() {
				continue
			}
			h := binding.Help()
			rows = append(rows, m.help.Styles.FullKey.Render(h.Key)+" "+m.help.Styles.FullDesc.Render(h.Desc))
		}
	}
	rows = append(rows, "", m.styles.muted.Render("↑/↓ and PgUp/PgDn scroll this view."))
	return strings.Join(rows, "\n")
}

func (m Model) footer(l panelLayout) string {
	h := m.keys.Select.Help()
	selectHint := m.help.Styles.ShortKey.Render(h.Key) + " " + m.help.Styles.ShortDesc.Render(h.Desc)
	// Select has its own row. Clip only when even the compact terminal cannot
	// physically fit the label; the help bubble never gets to truncate it.
	footerWidth := max(1, l.width-l.box.GetHorizontalFrameSize())
	selectHint = ansi.Truncate(selectHint, footerWidth, "")
	if l.footerRows == 1 {
		return selectHint
	}

	var hints []key.Binding
	for _, binding := range m.keys.ShortHelp() {
		if binding.Help() != m.keys.Select.Help() {
			hints = append(hints, binding)
		}
	}

	width := footerWidth
	var scrollHint string
	if m.viewport.TotalLineCount() > m.viewport.Height() {
		scrollHint = "↑/↓ PgUp/PgDn Scroll"
		if lipgloss.Width(scrollHint) > width {
			scrollHint = "↑↓ Scroll"
		}
		scrollHint = ansi.Truncate(m.styles.muted.Render(scrollHint), width, "")
		width -= lipgloss.Width(scrollHint) + 3
	}

	helper := m.help
	helper.SetWidth(max(1, width))
	otherHints := ""

	if width > 0 {
		otherHints = helper.ShortHelpView(hints)
	}

	if scrollHint != "" && otherHints != "" {
		scrollHint += " • "
	}

	return ansi.Truncate(scrollHint+otherHints, footerWidth, "") + "\n" + selectHint
}