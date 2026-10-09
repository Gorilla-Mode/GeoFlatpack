package tui

import (
	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/internal/tui/terminalpreview"
	tea "charm.land/bubbletea/v2"
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
		pickerClipped := false
		if scaffold {
			opts := m.scaffoldOptions(l)
			if opts.ColorPicker != nil {
				w, h := panel.ControlsDimensions(opts)
				pickerClipped = panel.ColorPickerGeometry(*opts.ColorPicker, w, h).Clipped(w, h)
			}
		}

		if m.screen == scaffoldScreen && (m.width < 120 || m.height < 24 || pickerClipped) {
			warning := m.styles.muted.Render("Expand terminal")
			if lipgloss.Width(header)+3+lipgloss.Width(warning) <= l.width {
				header += " • " + warning
			} else {
				header = ansi.Truncate(warning, l.width, "")
			}
		}

		if m.workflow.Status() != "" {
			style := m.styles.success
			if m.workflow.Busy() {
				style = m.styles.muted
			} else if m.workflow.Err() != nil {
				style = m.styles.failure
			}

			message := strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(ansi.Strip(m.workflow.Status()))
			status := style.Inline(true).Render(message)
			if lipgloss.Width(header)+3+lipgloss.Width(status) <= l.width {
				header += " • " + status
			} else {
				header = ansi.Truncate(status, l.width, "…")
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

	hints := m.keys.ShortHelp()[3:]
	selectKey := m.keys.Select
	helpKey := m.keys.Help
	if scaffold && m.editor.Pane() == panel.ControlsPane {
		selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Edit / apply"))
		hints = append([]key.Binding{
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back to editing")),
			key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("Tab/⇧Tab", "Control")),
		}, hints...)
		if m.editor.Editing() {
			helpKey = key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Cancel"))
			selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Save"))
			hints = []key.Binding{m.keys.LeftPane, m.keys.RightPane, key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←/→", "Cursor"))}
		}

		if c := m.editor.Context(); c.Color {
			selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Adjust / edit"))
			if c.Editing {
				helpKey = key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Leave input"))
				selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Finish input"))
			} else if c.Adjusting {
				helpKey = key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Leave adjustment"))
				selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Finish adjustment"))
				hints = []key.Binding{key.NewBinding(key.WithKeys("up", "down", "left", "right"), key.WithHelp("↑/↓/←/→", "Adjust color")), key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("Tab/⇧Tab", "Control")), m.keys.LeftPane, m.keys.RightPane}
			} else {
				hints = []key.Binding{key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back to editing")), key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("Tab/⇧Tab", "Control")), m.keys.LeftPane, m.keys.RightPane}
				if c.ColorSelected >= panel.PickerHue && c.ColorSelected <= panel.PickerLightness || c.ColorSelected >= panel.PickerRed && c.ColorSelected <= panel.PickerBlue {
					hints = append(hints, m.keys.DecreaseSelection, m.keys.IncreaseSelection)
				}
			}
		}
	}

	if scaffold && m.editor.Pane() == panel.FeatureStylingPane {
		if s := m.editor.Context(); s.HasStyling {
			contextHints := []key.Binding{}
			if s.Selecting && s.LayerSelected {
				contextHints = append(contextHints, m.keys.RemoveLayer)
			}

			if !s.Selecting {
				back := m.keys.Back
				if s.Icons {
					back = key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Layer types"))
					if s.ChangingIcon {
						back = key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "SVG options"))
					}
				}

				contextHints = append(contextHints, back)
			}

			if s.Edit {
				if s.Included {
					contextHints = append(contextHints, m.keys.RemoveOption)
				}

				contextHints = append(contextHints, m.keys.Filter)
				selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Toggle option"))
				if s.IncludedOnly {
					selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Edit option"))
				}

				if s.ChangeSVG {
					selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Change SVG"))
				}
			}

			hints = append(contextHints, hints...)
		}
	}

	primary := []key.Binding{helpKey, m.keys.Quit, selectKey}
	if scaffold && m.preview != nil && m.preview.Supported() && !m.editor.Editing() {
		hints = append([]key.Binding{m.keys.PreviewSample}, hints...)
	}

	if m.canWrite() {
		hints = append([]key.Binding{m.keys.WriteFiles}, hints...)
	}

	if m.workflow.Busy() {
		primary = []key.Binding{m.keys.Quit}
		hints = nil
	}

	sections = append(sections, panel.Footer(panel.FooterOptions{
		Style:      l.box,
		MutedStyle: m.styles.muted,
		Width:      l.width,
		Height:     l.footerHeight,
		Help:       m.help,
		Primary:    primary,
		Hints:      hints,
		Scrollable: !scaffold && m.viewport.TotalLineCount() > m.viewport.Height(),
	}))

	// Lip Gloss v2 dimensions include both padding and borders.

	return l.frame.PaddingBottom(0).Width(m.width).Height(m.height).Render(strings.Join(sections, "\n"))
}

func (m *Model) scaffoldOptions(l panelLayout) panel.ScaffoldOptions {
	opts := panel.ScaffoldOptions{
		Width: l.width, Height: l.bodyHeight, ActivePane: m.editor.Pane(),
		PaneStyle: m.styles.pane, InactivePaneStyle: m.styles.inactivePane,
		TitleStyle: m.styles.paneTitle, MutedStyle: m.styles.inactivePaneTitle,
		ScrollHintStyle:   m.styles.muted,
		ListStyles:        m.styles.list,
		CategoryEmptyText: "No layers loaded",
		FeatureEmptyText:  "No layers loaded",
	}

	opts = m.editor.Presentation(opts)
	m.preview.Presentation(&opts)

	return opts
}

type panelLayout struct {
	frame         lipgloss.Style
	box           lipgloss.Style
	bodyStyle     lipgloss.Style
	width         int
	height        int
	headerHeight  int
	headingHeight int
	gap           int
	footerGap     int
	footerHeight  int
	bodyHeight    int
	contentWidth  int
	contentHeight int
}

func (m *Model) layout() panelLayout {
	frame := lipgloss.NewStyle()

	// Restore the original outer padding when the body has room to use it.
	if m.width >= 24 && m.height >= 18 {
		frame = m.styles.frame
	} else if m.width >= 12 && m.height >= 8 {
		frame = frame.Padding(0, 1)
	}

	l := panelLayout{
		frame: frame,
		width: max(1, m.width-frame.GetHorizontalFrameSize()),

		// The footer uses the bottom padding row, as with the original + 1.
		height: max(1, m.height-frame.GetVerticalFrameSize()+min(1, frame.GetPaddingBottom())),
	}

	// Both boxes use the same outline. Drop padding, then borders, before
	// sacrificing Select on a small terminal.
	if l.width >= lipgloss.Width("↵ Select")+2 && l.height >= 9 {
		l.box = m.styles.box
		if l.width < 12 {
			l.box = l.box.Padding(0)
		}
	}

	l.footerHeight = 1 + l.box.GetVerticalFrameSize()
	if l.height >= 2 {
		l.headerHeight = 1
	}

	if l.height >= 12 {
		l.gap = 1
	}

	l.footerGap = l.gap
	if !m.help.ShowAll && m.screen != scaffoldScreen && l.box.GetVerticalFrameSize() > 0 {
		l.gap = 0
		l.footerGap = 0
	}

	l.bodyHeight = max(0, l.height-l.headerHeight-l.footerHeight-l.gap-l.footerGap)
	if m.help.ShowAll || m.screen != scaffoldScreen {
		l.bodyStyle = l.box
	}

	l.contentWidth = max(1, l.width-l.bodyStyle.GetHorizontalFrameSize())
	l.contentHeight = max(0, l.bodyHeight-l.bodyStyle.GetVerticalFrameSize())

	if l.contentHeight > 0 && (m.help.ShowAll || m.screen != scaffoldScreen) {
		l.headingHeight = 1
		l.contentHeight--
		if l.contentHeight >= 7 {
			l.headingHeight++
			l.contentHeight--
		}
	}

	return l
}

func (m *Model) refreshViewport() {
	m.editor.SetContext(m.Options.WriteFGB, m.editorVisible())
	m.editor.Refresh()
	l := m.layout()
	m.viewport.SetWidth(l.contentWidth)
	m.viewport.SetHeight(l.contentHeight)
	if m.screen == scaffoldScreen && !m.help.ShowAll {
		m.editor.RefreshWindows(m.scaffoldOptions(l))
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
		SVGDir:       m.Options.SVGDir,
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

	if m.workflow.Session() != nil {
		opts.Loaded = true
		for _, layer := range m.workflow.Session().Layers() {
			opts.LayerNames = append(opts.LayerNames, layer.Name)
		}

		for _, name := range m.editor.IconNames() {
			opts.SVGNames = append(opts.SVGNames, name+".svg")
		}
	}

	return panel.Processing(opts)
}

func (m *Model) View() tea.View {
	m.refreshViewport()
	view := tea.NewView(m.panel())
	view.AltScreen = true
	if m.screen == scaffoldScreen && !m.help.ShowAll && !m.workflow.Busy() {
		if _, ok := m.editor.ColorPicker(); ok {
			view.MouseMode = tea.MouseModeAllMotion
		}
	}

	return view
}

func (m *Model) previewDimensions() terminalpreview.Dimensions {
	return terminalpreview.Dimensions{TerminalWidth: m.width, TerminalHeight: m.height, Region: panel.PreviewRegion(m.scaffoldOptions(m.layout()))}
}
