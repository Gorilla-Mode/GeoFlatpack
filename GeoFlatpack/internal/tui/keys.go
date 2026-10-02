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
			key.WithKeys("super+left", "ctrl+left"),
			key.WithHelp("⌘←", "Left pane"),
		),
		RightPane: key.NewBinding(
			key.WithKeys("super+right", "ctrl+right"),
			key.WithHelp("⌘→", "Right pane"),
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
			key.WithHelp("Control+C", "Exit GeoFlatpack immediately"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "Primary action. Selects the current item, enters field, etc."),
		),
		LeftPane: key.NewBinding(
			key.WithKeys("super+left", "ctrl+left"),
			key.WithHelp("Super+Left / Control+Left", "Move to the left pane"),
		),
		RightPane: key.NewBinding(
			key.WithKeys("super+right", "ctrl+right"),
			key.WithHelp("Super+Right / Control+Right", "Move to the right pane"),
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

type helpKeyMap struct {
	keyMap
	detailKeyMap
}
