package tui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	OmitLayer         key.Binding
	PreviewSample     key.Binding
	WriteFiles        key.Binding
	RemoveLayer       key.Binding
	RemoveOption      key.Binding
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
		OmitLayer:     key.NewBinding(key.WithKeys("x", "X"), key.WithHelp("X", "Omit / restore layer")),
		PreviewSample: key.NewBinding(key.WithKeys("d", "D"), key.WithHelp("D", "Sample / real preview")),
		WriteFiles:    key.NewBinding(key.WithKeys("w", "W"), key.WithHelp("W", "Write files")),
		RemoveLayer:   key.NewBinding(key.WithKeys("backspace"), key.WithHelp("⌫", "Remove layer")),
		RemoveOption:  key.NewBinding(key.WithKeys("backspace"), key.WithHelp("⌫", "Remove option")),
		Back:          key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Style layers")),
		Filter:        key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "Filter options")),
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
	Conditional       key.Binding
	ConditionBack     key.Binding
	MoveCondition     key.Binding
	OmitLayer         key.Binding
	PreviewSample     key.Binding
	WriteFiles        key.Binding
	RemoveLayer       key.Binding
	RemoveOption      key.Binding
	ColorPicker       key.Binding
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
		Conditional:   key.NewBinding(key.WithKeys("c"), key.WithHelp("C / H in Categories", "C marks or unmarks the highlighted field for conditional styling; H shows all fields or marked fields only. No category field cannot be marked. Enter in all fields activates the category and opens Features; Enter in marked fields opens its condition editor in Controls.")),
		ConditionBack: key.NewBinding(key.WithKeys("backspace"), key.WithHelp("Backspace in condition Controls", "Return to Categories outside text editing; delete text while editing. Enter edits or commits a value; Escape cancels the edit.")),
		MoveCondition: key.NewBinding(key.WithKeys("ctrl+up", "ctrl+down"), key.WithHelp("Control+Up / Control+Down", "Move a condition and its styling together. Conditions match in order; the first match wins. Numbers, strings and booleans remain distinct; ordering requires numbers. Missing, null and incompatible values use Otherwise. X in Features omits/restores Otherwise while retaining its draft.")),
		OmitLayer:     key.NewBinding(key.WithKeys("x", "X"), key.WithHelp("X", "Omit / restore layer in Layers; omitted layers show ⊗ and are excluded from all outputs, retaining their styling drafts")),
		PreviewSample: key.NewBinding(key.WithKeys("d", "D"), key.WithHelp("D", "Toggle compact sample geometry or real geometry in Preview for all features; inactive while editing an input")),
		ColorPicker:   key.NewBinding(key.WithKeys("enter", "esc", "tab", "shift+tab", "left", "right"), key.WithHelp("Color picker", "Tab/Shift+Tab focuses Square, H, S, L, Hex, R, G, B in order or reverse. Left/Right adjusts a focused slider or RGB value without Enter. Enter activates square/slider adjustment or Hex/RGB text editing; Escape leaves it. Edits apply live. Click and drag to pick colors; alpha is always 255.")),
		WriteFiles:    key.NewBinding(key.WithKeys("w", "W"), key.WithHelp("W", "Write enabled outputs to -o when at least one layer is included and all included layers are green; inactive while editing an input")),
		RemoveLayer:   key.NewBinding(key.WithKeys("backspace"), key.WithHelp("Backspace", "Remove the highlighted style layer in the style-layer selection list")),
		RemoveOption:  key.NewBinding(key.WithKeys("backspace"), key.WithHelp("Backspace", "Remove the highlighted included option in the styling editor, including the filtered list")),
		ControlFocus:  key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("Tab / Shift+Tab", "Focus the next or previous input, adjustment button, or action in Controls")),
		Back:          key.NewBinding(key.WithKeys("esc"), key.WithHelp("Escape", "Cancel input editing; from Controls return to Styling; otherwise return to style layers or SVG layer types")),
		Filter:        key.NewBinding(key.WithKeys("h"), key.WithHelp("H", "Show all style options or only included options; Change SVG remains available")),
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
		k.Conditional,
		k.ConditionBack,
		k.MoveCondition,
		k.OmitLayer,
		k.PreviewSample,
		k.WriteFiles,
		k.RemoveLayer,
		k.RemoveOption,
		k.ColorPicker,
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
