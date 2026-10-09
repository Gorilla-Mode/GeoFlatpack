package panel

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// KeyboardReference renders all enabled bindings before viewport wrapping.
func KeyboardReference(groups [][]key.Binding, styles help.Styles, muted lipgloss.Style) string {
	var rows []string

	// Render complete rows before wrapping. The help bubble's column truncation
	// would otherwise discard the entire reference on a narrow terminal.
	for _, group := range groups {
		for _, binding := range group {
			if !binding.Enabled() {
				continue
			}

			h := binding.Help()
			rows = append(rows, styles.FullKey.Render(h.Key)+" "+styles.FullDesc.Render(h.Desc))
		}
	}

	rows = append(rows, "", muted.Render("↑/↓ and PgUp/PgDn scroll this view."))

	return strings.Join(rows, "\n")
}
