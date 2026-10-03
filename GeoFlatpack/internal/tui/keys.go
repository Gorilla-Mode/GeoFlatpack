package tui

import "charm.land/bubbles/v2/key"

type keyMap struct {
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
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("Question mark", "Show or hide the complete keyboard reference"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("Control+C", "Exit GeoFlatpack after active processing finishes"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "Primary action. Selects the current item, enters field, etc."),
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
			key.WithHelp("Right", "Increases the selection"),
		),
		DecreaseSelection: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("Left", "Decreases the selection"),
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
