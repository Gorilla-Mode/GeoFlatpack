// Package editor owns navigation and private styling drafts.
package editor

import (
	"GeoFlatpack/internal/app"
	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/style/maplibre/svg"

	tea "charm.land/bubbletea/v2"
)

type Editor struct {
	sources                          []app.Layer
	icons                            map[string]svg.Svg
	loaded, writeFGB, interactive    bool
	activePane                       panel.Pane
	layers                           []panel.ListItem
	categories                       []categoryState
	stylingCatalog                   stylingCatalog
	selectedLayer, firstVisibleLayer int
}

func New() *Editor {
	return &Editor{stylingCatalog: cachedStylingCatalog()}
}

func (m *Editor) Load(layers []app.Layer, icons map[string]svg.Svg, loaded bool) {
	m.sources, m.icons, m.loaded = layers, icons, loaded
	m.layers = layerItems(layers)
	m.categories = nil
	for _, layer := range layers {
		m.categories = append(m.categories, newCategoryState(layer.Data))
	}
}

// SetContext gates live input and initial color application while help or work
// owns the screen. Output options remain controlled by Model.Options.
func (m *Editor) SetContext(writeFGB, interactive bool) {
	m.writeFGB, m.interactive = writeFGB, interactive
}

func (m *Editor) Pane() panel.Pane { return m.activePane }

func (m *Editor) Focus(pane panel.Pane) {
	m.StopDragging()
	m.activePane = pane
}

func (m *Editor) Open() {
	m.Focus(panel.LayerPane)
	m.selectedLayer, m.firstVisibleLayer = 0, 0
}

// Input handles editor shortcuts and delegates form events to their drafts.
func (m *Editor) Input(msg tea.Msg) tea.Cmd {
	if !m.interactive {
		return nil
	}

	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "alt+right", "alt+f", "ctrl+right":
			m.Focus((m.activePane + 1) % panel.PaneCount)

			return nil
		case "alt+left", "alt+b", "ctrl+left":
			m.Focus((m.activePane + panel.PaneCount - 1) % panel.PaneCount)

			return nil
		case "esc":
			if m.activePane == panel.ControlsPane && !m.Editing() {
				m.Focus(panel.FeatureStylingPane)

				return nil
			}
		}

		delta := 0
		if k.String() == "up" {
			delta = -1
		}

		if k.String() == "down" {
			delta = 1
		}

		switch m.activePane {
		case panel.LayerPane:
			if delta != 0 {
				m.selectedLayer = panel.MoveListSelection(m.selectedLayer, delta, len(m.layers))
			}

			if k.String() == "enter" && len(m.layers) > 0 {
				if m.layers[m.selectedLayer].Status == panel.ListUnopened {
					m.layers[m.selectedLayer].Status = panel.ListIncomplete
				}

				m.activePane = panel.CategoryPane
			}
		case panel.CategoryPane:
			if c := m.currentCategories(); c != nil {
				if delta != 0 {
					c.selected = panel.MoveListSelection(c.selected, delta, len(c.items))
				}

				if k.String() == "enter" && len(c.items) > 0 {
					c.activate()
					m.layers[m.selectedLayer].Status = panel.ListIncomplete
					m.activePane = panel.FeaturesPane
				}
			}
		case panel.FeaturesPane:
			if f := m.currentFeatures(); f != nil {
				if delta != 0 {
					f.selected = panel.MoveListSelection(f.selected, delta, len(f.items))
				}

				if k.String() == "enter" && len(f.items) > 0 {
					f.choose()
					m.activePane = panel.FeatureStylingPane
				}
			}
		case panel.FeatureStylingPane:
			m.stylingInput(k)
		case panel.ControlsPane:
			return m.controlsInput(msg)
		}
	} else if m.Editing() {
		return m.controlsInput(msg)
	}

	return nil
}

func (m *Editor) Refresh() {
	if m.interactive && m.activePane == panel.ControlsPane {
		m.currentControl()
	}

	m.RefreshReadiness()
}
