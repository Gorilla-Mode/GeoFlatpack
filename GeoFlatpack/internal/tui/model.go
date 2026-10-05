// Package tui provides the terminal frontend.
package tui

import (
	"GeoFlatpack/internal/app"
	"GeoFlatpack/internal/tui/panel"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

type screen int

const (
	loadingScreen screen = iota
	completionScreen
	scaffoldScreen
	failureScreen
)

// Model retains the loaded session for future processing screens. Run owns its
// preparation and cleanup lifecycle.
type Model struct {
	Options           app.Options
	help              help.Model
	keys              keyMap
	detailKeys        detailKeyMap
	styles            styles
	width             int
	height            int
	spinner           spinner.Model
	viewport          viewport.Model
	screen            screen
	activePane        panel.Pane
	layers            []panel.ListItem
	categories        []categoryState
	stylingCatalog    stylingCatalog
	selectedLayer     int
	firstVisibleLayer int
	startedAt         time.Time
	elapsed           time.Duration
	quitting          bool
	err               error
	session           loadedSession
	preparation       *preparation
}

var _ tea.Model = (*Model)(nil)

// NewModel creates a loading screen; Init schedules input preparation.
func NewModel(opts app.Options) *Model {
	return newModel(opts, prepareInput)
}

func newModel(opts app.Options, prepare prepareFunc) *Model {
	s := newStyles()
	h := help.New()
	h.Styles = s.help
	m := &Model{
		Options:        opts,
		help:           h,
		keys:           newKeyMap(),
		detailKeys:     newDetailKeyMap(),
		styles:         s,
		stylingCatalog: cachedStylingCatalog(),
		width:          80,
		height:         24,
		spinner:        spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(s.title)),
		viewport:       viewport.New(),
		startedAt:      time.Now(),
		preparation:    newPreparation(prepare),
	}
	m.viewport.SetHorizontalStep(0)
	m.refreshViewport()
	return m
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.preparation.command(m.Options), m.spinner.Tick)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case preparedMsg:
		m.elapsed, m.err = msg.elapsed, msg.err

		if m.quitting {
			return m, tea.Quit
		}
		if msg.err != nil {
			m.screen = failureScreen
		} else {
			m.session = msg.session
			m.layers = nil
			m.categories = nil
			if m.session != nil {
				layers := m.session.Layers()
				m.layers = layerItems(layers)
				for _, layer := range layers {
					m.categories = append(m.categories, newCategoryState(layer.Data))
				}
			}
			m.screen = completionScreen
		}
		m.viewport.GotoTop()
	case spinner.TickMsg:
		if m.screen == loadingScreen {
			m.elapsed = max(0, msg.Time.Sub(m.startedAt))
			m.spinner, cmd = m.spinner.Update(msg)
		}
	case tea.WindowSizeMsg:
		m.width, m.height = max(1, msg.Width), max(1, msg.Height)
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.screen == loadingScreen {
				m.quitting = true
				break
			}
			return m, tea.Quit
		case key.Matches(msg, m.keys.Select) && m.screen == completionScreen:
			m.screen = scaffoldScreen
			m.activePane = panel.LayerPane
			m.selectedLayer, m.firstVisibleLayer = 0, 0
			m.viewport.GotoTop()
			m.help.ShowAll = false
			m.keys.Help.SetHelp("?", "help")
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			m.viewport.GotoTop()
			if m.help.ShowAll {
				m.keys.Help.SetHelp("?", "hide help")
			} else {
				m.keys.Help.SetHelp("?", "help")
			}
		case key.Matches(msg, m.keys.RightPane) && m.screen == scaffoldScreen && !m.help.ShowAll:
			m.activePane = (m.activePane + 1) % panel.PaneCount
		case key.Matches(msg, m.keys.LeftPane) && m.screen == scaffoldScreen && !m.help.ShowAll:
			m.activePane = (m.activePane + panel.PaneCount - 1) % panel.PaneCount
		case key.Matches(msg, m.keys.SelectionUp) && m.screen == scaffoldScreen && !m.help.ShowAll && m.activePane == panel.LayerPane:
			m.selectedLayer = panel.MoveListSelection(m.selectedLayer, -1, len(m.layers))
		case key.Matches(msg, m.keys.SelectionDown) && m.screen == scaffoldScreen && !m.help.ShowAll && m.activePane == panel.LayerPane:
			m.selectedLayer = panel.MoveListSelection(m.selectedLayer, 1, len(m.layers))
		case key.Matches(msg, m.keys.Select) && m.screen == scaffoldScreen && !m.help.ShowAll && m.activePane == panel.LayerPane && len(m.layers) > 0:
			if m.layers[m.selectedLayer].Status == panel.ListUnopened {
				m.layers[m.selectedLayer].Status = panel.ListIncomplete
			}
			m.activePane = panel.CategoryPane
		case key.Matches(msg, m.keys.SelectionUp) && m.screen == scaffoldScreen && !m.help.ShowAll && m.activePane == panel.CategoryPane:
			if categories := m.currentCategories(); categories != nil {
				categories.selected = panel.MoveListSelection(categories.selected, -1, len(categories.items))
			}
		case key.Matches(msg, m.keys.SelectionDown) && m.screen == scaffoldScreen && !m.help.ShowAll && m.activePane == panel.CategoryPane:
			if categories := m.currentCategories(); categories != nil {
				categories.selected = panel.MoveListSelection(categories.selected, 1, len(categories.items))
			}
		case key.Matches(msg, m.keys.Select) && m.screen == scaffoldScreen && !m.help.ShowAll && m.activePane == panel.CategoryPane:
			if categories := m.currentCategories(); categories != nil && len(categories.items) > 0 {
				categories.activate()
				m.layers[m.selectedLayer].Status = panel.ListIncomplete
				m.activePane = panel.FeaturesPane
			}
		case key.Matches(msg, m.keys.SelectionUp) && m.screen == scaffoldScreen && !m.help.ShowAll && m.activePane == panel.FeaturesPane:
			if features := m.currentFeatures(); features != nil {
				features.selected = panel.MoveListSelection(features.selected, -1, len(features.items))
			}
		case key.Matches(msg, m.keys.SelectionDown) && m.screen == scaffoldScreen && !m.help.ShowAll && m.activePane == panel.FeaturesPane:
			if features := m.currentFeatures(); features != nil {
				features.selected = panel.MoveListSelection(features.selected, 1, len(features.items))
			}
		case key.Matches(msg, m.keys.Select) && m.screen == scaffoldScreen && !m.help.ShowAll && m.activePane == panel.FeaturesPane:
			if features := m.currentFeatures(); features != nil && len(features.items) > 0 {
				features.choose()
				m.activePane = panel.FeatureStylingPane
			}
		case m.screen == scaffoldScreen && !m.help.ShowAll && m.activePane == panel.FeatureStylingPane:
			m.stylingInput(msg)
		case m.screen == scaffoldScreen && !m.help.ShowAll && m.activePane == panel.ControlsPane:
			// Reserved for value controls; Info remains display-only.
		default:
			if m.help.ShowAll || m.screen != scaffoldScreen {
				m.refreshViewport()
				m.viewport, cmd = m.viewport.Update(msg)
			}
		}
	}
	m.refreshViewport()
	return m, cmd
}

func (m *Model) View() tea.View {
	m.refreshViewport()
	view := tea.NewView(m.panel())
	view.AltScreen = true
	return view
}
