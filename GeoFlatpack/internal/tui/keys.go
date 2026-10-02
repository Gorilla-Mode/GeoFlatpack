package tui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Help                    key.Binding
	HelpDetail              key.Binding
	Quit                    key.Binding
	QuitDetail              key.Binding
	Select                  key.Binding
	SelectDetail            key.Binding
	LeftPane                key.Binding
	RightPane               key.Binding
	LeftPaneDetail          key.Binding
	RightPaneDetail         key.Binding
	SelectionUp             key.Binding
	SelectionDown           key.Binding
	SelectionUpDetail       key.Binding
	SelectionDownDetail     key.Binding
	IncreaseSelection       key.Binding
	DecreaseSelection       key.Binding
	IncreaseSelectionDetail key.Binding
	DecreaseSelectionDetail key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		HelpDetail: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("Question mark", "Show or hide the complete keyboard reference"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("⌃C", "Quit"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("↵", "Select"),
		),
		SelectDetail: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("Enter", "Primary action. Selects the current item, enters field, etc."),
		),
		QuitDetail: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("Control+C", "Exit GeoFlatpack immediately"),
		),
		LeftPane: key.NewBinding(
			key.WithKeys("super+left", "ctrl+left"),
			key.WithHelp("⌘←", "Left pane"),
		),
		RightPane: key.NewBinding(
			key.WithKeys("super+right", "ctrl+right"),
			key.WithHelp("⌘→", "Right pane"),
		),
		LeftPaneDetail: key.NewBinding(
			key.WithKeys("super+left", "ctrl+left"),
			key.WithHelp("Super+Left / Control+Left", "Move to the left pane"),
		),
		RightPaneDetail: key.NewBinding(
			key.WithKeys("super+right", "ctrl+right"),
			key.WithHelp("Super+Right / Control+Right", "Move to the right pane"),
		),
		SelectionUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "Up"),
		),
		SelectionDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "Down"),
		),
		SelectionUpDetail: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("Up", "Moves selection up"),
		),
		SelectionDownDetail: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("Down", "Moves selection down"),
		),
		IncreaseSelection: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "Increase"),
		),
		DecreaseSelection: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "Decrease"),
		),
		IncreaseSelectionDetail: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("Right", "Increases the selection"),
		),
		DecreaseSelectionDetail: key.NewBinding(
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

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{
		k.HelpDetail,
		k.QuitDetail,
		k.SelectDetail,
		k.LeftPaneDetail,
		k.RightPaneDetail,
		k.SelectionUpDetail,
		k.SelectionDownDetail,
		k.IncreaseSelectionDetail,
		k.DecreaseSelectionDetail,
	},
	}
}
