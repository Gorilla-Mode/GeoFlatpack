// Package panel renders terminal panels from the supplied content and styles.
package panel

import (
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Header renders the application title.
func Header(width int, title lipgloss.Style) string {
	return title.Render(ansi.Truncate("GeoFlatpack", width, ""))
}
