package tui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	RemoveLayer       key.Binding
	Back              key.Binding
	Filter            key.Binding
	Help              key.Binding
	Quit              key.Binding
	Select            key.Binding
	LeftPane          key.Binding
	RightPane         key.Binding
	SelectionUp       key.Binding
	SelectionDown     key.Binding
	IncreaseSelection key.Binding
	DecreaseSelection key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		RemoveLayer: key.NewBinding(key.WithKeys("backspace"), key.WithHelp("⌫", "Remove layer")),
		Back:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Style layers")),
		Filter:      key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "Filter options")),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("⌃C", "Quit"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "Select"),
		),
		LeftPane: key.NewBinding(
			// Ghostty's macOS Option+Left binding sends Escape+b (Alt+b).
			key.WithKeys("alt+left", "alt+b", "ctrl+left"),
			key.WithHelp("⌥←", "Left pane"),
		),
		RightPane: key.NewBinding(
			// Ghostty's macOS Option+Right binding sends Escape+f (Alt+f).
			key.WithKeys("alt+right", "alt+f", "ctrl+right"),
			key.WithHelp("⌥→", "Right pane"),
		),
		SelectionUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "Up"),
		),
		SelectionDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "Down"),
		),
		IncreaseSelection: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "Increase"),
		),
		DecreaseSelection: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "Decrease"),
		),
	}
}

type detailKeyMap struct {
	RemoveLayer       key.Binding
	ControlFocus      key.Binding
	Back              key.Binding
	Filter            key.Binding
	Help              key.Binding
	Quit              key.Binding
	Select            key.Binding
	LeftPane          key.Binding
	RightPane         key.Binding
	SelectionUp       key.Binding
	SelectionDown     key.Binding
	IncreaseSelection key.Binding
	DecreaseSelection key.Binding
}

func newDetailKeyMap() detailKeyMap {
	return detailKeyMap{
		RemoveLayer:  key.NewBinding(key.WithKeys("backspace"), key.WithHelp("Backspace", "Remove the highlighted style layer in the style-layer selection list")),
		ControlFocus: key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("Tab / Shift+Tab", "Focus the next or previous input, adjustment button, or action in Controls")),
		Back:         key.NewBinding(key.WithKeys("esc"), key.WithHelp("Escape", "Cancel input editing; from Controls return to Styling; otherwise return to style layers or SVG layer types")),
		Filter:       key.NewBinding(key.WithKeys("h"), key.WithHelp("H", "Show all style options or only included options; inactive for SVG editing")),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("Question mark", "Show or hide the complete keyboard reference; type a question mark while editing an input"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("Control+C", "Exit GeoFlatpack after active processing finishes"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "Select an item, toggle a style option (open Controls when filtered), use an SVG, edit/save a field, or apply an action"),
		),
		LeftPane: key.NewBinding(
			key.WithKeys("alt+left", "alt+b", "ctrl+left"),
			key.WithHelp("Option/Alt+Left / Control+Left", "Move to the left pane"),
		),
		RightPane: key.NewBinding(
			key.WithKeys("alt+right", "alt+f", "ctrl+right"),
			key.WithHelp("Option/Alt+Right / Control+Right", "Move to the right pane"),
		),
		SelectionUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("Up", "Moves selection up"),
		),
		SelectionDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("Down", "Moves selection down"),
		),
		IncreaseSelection: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("Right", "Increase a number or change a choice in Controls; move the cursor while editing"),
		),
		DecreaseSelection: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("Left", "Decrease a number or change a choice in Controls; move the cursor while editing"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Help,
		k.Quit,
		k.Select,
		k.LeftPane,
		k.RightPane,
		k.SelectionUp,
		k.SelectionDown,
		k.IncreaseSelection,
		k.DecreaseSelection,
	}
}

func (k detailKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{
		k.RemoveLayer,
		k.ControlFocus,
		k.Back,
		k.Filter,
		k.Help,
		k.Quit,
		k.Select,
		k.LeftPane,
		k.RightPane,
		k.SelectionUp,
		k.SelectionDown,
		k.IncreaseSelection,
		k.DecreaseSelection,
	},
	}
}
