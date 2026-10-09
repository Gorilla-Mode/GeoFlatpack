package editor

import (
	"GeoFlatpack/internal/tui/panel"
	"slices"
)

// Presentation supplies panel data without exposing mutable styling drafts.
func (m *Editor) Presentation(opts panel.ScaffoldOptions) panel.ScaffoldOptions {
	opts = panel.ScaffoldOptions{
		Width: opts.Width, Height: opts.Height, ActivePane: m.activePane,
		PaneStyle: opts.PaneStyle, InactivePaneStyle: opts.InactivePaneStyle,
		TitleStyle: opts.TitleStyle, MutedStyle: opts.MutedStyle,
		ScrollHintStyle: opts.ScrollHintStyle, ListStyles: opts.ListStyles,
		PreviewContent: opts.PreviewContent, PreviewWarning: opts.PreviewWarning,
		PreviewSample: opts.PreviewSample, PreviewImage: opts.PreviewImage,
	}

	opts.Layers = m.layers
	opts.SelectedLayer, opts.FirstVisibleLayer = m.selectedLayer, m.firstVisibleLayer
	opts.CategoryEmptyText, opts.FeatureEmptyText = "No layers loaded", "No layers loaded"
	m.present(&opts)
	opts.Layers = copyItems(opts.Layers)
	opts.Categories = copyItems(opts.Categories)
	opts.Features = copyItems(opts.Features)

	return opts
}

func (m *Editor) present(opts *panel.ScaffoldOptions) {
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

	m.stylingPresentation(opts)
	m.controlsPresentation(opts)
}

// RefreshWindows keeps each navigation window attached to its owning draft.
func (m *Editor) RefreshWindows(opts panel.ScaffoldOptions) {
	m.firstVisibleLayer = panel.LayerWindow(m.Presentation(opts))
	if categories := m.currentCategories(); categories != nil {
		categories.firstVisible = panel.CategoryWindow(m.Presentation(opts))
	}

	if features := m.currentFeatures(); features != nil {
		features.firstVisible = panel.FeaturesWindow(m.Presentation(opts))
	}

	if s := m.currentStyling(); s != nil {
		opts := m.Presentation(opts)
		first := panel.StylingWindow(opts)
		s.stackFirstVisible = panel.StackWindow(opts)
		switch s.mode {
		case styleTypes:
			s.typeFirstVisible = first
		case styleIcons:
			s.iconFirstVisible = first
		case styleSelection:
			s.firstVisible = first
		case styleEdit:
			layer := s.activeLayer()
			layer.firstVisible = first
		}
	}

	if _, _, c := m.currentControl(); c != nil {
		c.firstVisible = panel.ControlsWindow(m.Presentation(opts))
	}
}

// Context describes the current editor actions for the root's shortcut hints.
type Context struct {
	Color, Editing, Adjusting               bool
	ColorSelected                           int
	HasStyling, Selecting, LayerSelected    bool
	Icons, ChangingIcon                     bool
	Edit, Included, IncludedOnly, ChangeSVG bool
}

func (m *Editor) Context() Context {
	var result Context
	if _, _, c := m.currentControl(); c != nil {
		result.Editing = c.editing
		if c.color != nil {
			result.Color, result.Adjusting, result.ColorSelected = true, c.color.active, c.color.selected
		}
	}

	if s := m.currentStyling(); s != nil {
		result.HasStyling = true
		result.Selecting = s.mode == styleSelection
		result.LayerSelected = s.selected >= 0 && s.selected < len(s.layers)
		result.Icons, result.ChangingIcon, result.Edit = s.mode == styleIcons, s.changingIcon, s.mode == styleEdit
		if result.Edit {
			layer := s.activeLayer()
			result.IncludedOnly = layer.includedOnly
			result.ChangeSVG = layer.optionOffset() > 0 && layer.selected == 0
			if property, ok := layer.currentProperty(); ok {
				result.Included = layer.options[property.name].included
			}
		}
	}

	return result
}

func copyItems(items []panel.ListItem) []panel.ListItem {
	snapshot := slices.Clone(items)
	for i := range snapshot {
		snapshot[i].Children = slices.Clone(snapshot[i].Children)
	}

	return snapshot
}
