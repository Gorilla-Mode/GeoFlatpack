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
	ConditionValues   key.Binding
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
		Conditional:     key.NewBinding(key.WithKeys("c"), key.WithHelp("C / H in Categories", "C marks the highlighted field and immediately activates the shared conditional collection (green indicators, blue names); press C again to unmark; H shows all fields or marked fields only. Enter on a marked field in the full view activates all marked fields and opens their combinations in Features; Enter in the filtered view edits that field’s conditions and keeps its category highlighted without changing export activation. Changing marks updates an active collection immediately; removing its last field clears activation. No category field cannot be marked.")),
		ConditionBack:   key.NewBinding(key.WithKeys("backspace"), key.WithHelp("Backspace in condition Controls", "Return to Categories outside text editing; delete text while editing. Enter edits or commits a value; Escape cancels the edit.")),
		ConditionValues: key.NewBinding(key.WithKeys("enter", "up", "down", "tab", "shift+tab"), key.WithHelp("Condition values", "Value types follow the field’s observed types, plus Missing value; schema types are used when no scalar values are loaded. Numbers use text input. Enter on a string or boolean Value opens its choices; Up/Down or Tab/Shift+Tab browses and Enter selects. Strings offer Missing value first, then every distinct loaded value, with Enter custom value to type another string. Selecting Missing value preserves the string draft and styling. Escape cancels custom input back to the choices, or closes the choices without changing the rule. Write, help and preview shortcuts are inactive while choosing or editing a value.")),
		MoveCondition:   key.NewBinding(key.WithKeys("ctrl+up", "ctrl+down"), key.WithHelp("Control+Up / Control+Down", "Move a condition and its styling together. Each field uses its first matching condition; Features styles every combination across marked fields. Numbers, strings and booleans remain distinct; ordering requires numbers. Choose Missing value with == for absent/null attributes or != for any non-null value; its Value row needs no input and is skipped by navigation. If any field has no matching rule, use one shared Otherwise. New combinations inherit independent copies of parent styles; removing fields restores saved smaller-collection drafts. X in Features omits/restores Otherwise while retaining its draft.")),
		OmitLayer:       key.NewBinding(key.WithKeys("x", "X"), key.WithHelp("X", "Omit / restore layer in Layers; omitted layers show ⊗ and are excluded from all outputs, retaining their styling drafts")),
		PreviewSample:   key.NewBinding(key.WithKeys("d", "D"), key.WithHelp("D", "Toggle compact sample geometry or real geometry in Preview for all features; inactive while editing an input")),
		ColorPicker:     key.NewBinding(key.WithKeys("enter", "esc", "tab", "shift+tab", "left", "right"), key.WithHelp("Color picker", "Tab/Shift+Tab focuses Square, H, S, L, Hex, R, G, B in order or reverse. Left/Right adjusts a focused slider or RGB value without Enter. Enter activates square/slider adjustment or Hex/RGB text editing; Escape leaves it. Edits apply live. Click and drag to pick colors; alpha is always 255.")),
		WriteFiles:      key.NewBinding(key.WithKeys("w", "W"), key.WithHelp("W", "Write enabled outputs to -o when at least one layer is included and all included layers are green; inactive while editing an input")),
		RemoveLayer:     key.NewBinding(key.WithKeys("backspace"), key.WithHelp("Backspace", "Remove the highlighted style layer in the style-layer selection list")),
		RemoveOption:    key.NewBinding(key.WithKeys("backspace"), key.WithHelp("Backspace", "Remove the highlighted included option in the styling editor, including the filtered list")),
		ControlFocus:    key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("Tab / Shift+Tab", "Focus the next or previous input, adjustment button, or action in Controls")),
		Back:            key.NewBinding(key.WithKeys("esc"), key.WithHelp("Escape", "Cancel input editing; from Controls return to Styling; otherwise return to style layers or SVG layer types")),
		Filter:          key.NewBinding(key.WithKeys("h"), key.WithHelp("H", "Show all style options or only included options; Change SVG remains available")),
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
		k.ConditionValues,
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
