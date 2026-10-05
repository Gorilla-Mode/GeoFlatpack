package tui

import (
	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/style/maplibre"
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
		if m.writeStatus != "" {
			style := m.styles.success
			if m.writing {
				style = m.styles.muted
			} else if m.writeErr != nil {
				style = m.styles.failure
			}
			message := strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(ansi.Strip(m.writeStatus))
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
	if scaffold && m.activePane == panel.ControlsPane {
		selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Edit / apply"))
		hints = append([]key.Binding{
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back to editing")),
			key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("Tab/⇧Tab", "Control")),
		}, hints...)
		if m.controlsEditing() {
			helpKey = key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Cancel"))
			selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Save"))
			hints = []key.Binding{m.keys.LeftPane, m.keys.RightPane, key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←/→", "Cursor"))}
		}
		if _, _, c := m.currentControl(); c != nil && c.color != nil {
			selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Adjust / edit"))
			if c.editing {
				helpKey = key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Leave input"))
				selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Finish input"))
			} else if c.color.active {
				helpKey = key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Leave adjustment"))
				selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Finish adjustment"))
				hints = []key.Binding{key.NewBinding(key.WithKeys("up", "down", "left", "right"), key.WithHelp("↑/↓/←/→", "Adjust color")), key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("Tab/⇧Tab", "Control")), m.keys.LeftPane, m.keys.RightPane}
			} else {
				hints = []key.Binding{key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Back to editing")), key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("Tab/⇧Tab", "Control")), m.keys.LeftPane, m.keys.RightPane}
				if c.color.selected >= panel.PickerHue && c.color.selected <= panel.PickerLightness || c.color.selected >= panel.PickerRed && c.color.selected <= panel.PickerBlue {
					hints = append(hints, m.keys.DecreaseSelection, m.keys.IncreaseSelection)
				}
			}
		}
	}
	if scaffold && m.activePane == panel.FeatureStylingPane {
		if s := m.currentStyling(); s != nil {
			contextHints := []key.Binding{}
			if s.mode == styleSelection && s.selected >= 0 && s.selected < len(s.layers) {
				contextHints = append(contextHints, m.keys.RemoveLayer)
			}
			if s.mode != styleSelection {
				back := m.keys.Back
				if s.mode == styleIcons {
					back = key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Layer types"))
				}
				contextHints = append(contextHints, back)
			}
			if s.mode == styleEdit {
				if s.activeLayer().style.Type == maplibre.RenderSymbol {
					selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Use SVG"))
				} else {
					layer := s.activeLayer()
					if property, ok := layer.currentProperty(); ok && layer.options[property.name].included {
						contextHints = append(contextHints, m.keys.RemoveOption)
					}
					contextHints = append(contextHints, m.keys.Filter)
					selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Toggle option"))
					if s.activeLayer().includedOnly {
						selectKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "Edit option"))
					}
				}
			}
			hints = append(contextHints, hints...)
		}
	}
	primary := []key.Binding{helpKey, m.keys.Quit, selectKey}
	if m.canWrite() {
		hints = append([]key.Binding{m.keys.WriteFiles}, hints...)
	}
	if m.writing {
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
		Width: l.width, Height: l.bodyHeight, ActivePane: m.activePane,
		PaneStyle: m.styles.pane, InactivePaneStyle: m.styles.inactivePane,
		TitleStyle: m.styles.paneTitle, MutedStyle: m.styles.inactivePaneTitle,
		ScrollHintStyle: m.styles.muted,
		ListStyles:      m.styles.list,
		Layers:          m.layers, SelectedLayer: m.selectedLayer, FirstVisibleLayer: m.firstVisibleLayer,
		CategoryEmptyText: "No layers loaded",
		FeatureEmptyText:  "No layers loaded",
	}
	if categories := m.currentCategories(); categories != nil {
		opts.Categories = categories.items
		opts.SelectedCategory = categories.selected
		opts.FirstVisibleCategory = categories.firstVisible
		opts.CategoryEmptyText = categories.emptyText
		opts.FeatureEmptyText = "Choose a category"
	}
	if features := m.currentFeatures(); features != nil {
		opts.Features = features.items
		opts.SelectedFeature = features.selected
		opts.FirstVisibleFeature = features.firstVisible
		opts.ChosenFeature = features.chosen
		opts.FeatureChosen = features.chosen >= 0
		opts.FeatureEmptyText = "No styling items"
	}
	m.stylingPresentation(&opts)
	m.controlsPresentation(&opts)
	return opts
}
