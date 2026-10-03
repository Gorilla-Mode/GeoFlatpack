package panel

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// LayerItem contains only the information displayed by the Layers pane.
type LayerItem struct {
	Name         string
	FeatureCount int
	Complete     bool
}

const layerItemHeight = 2

func layerCapacity(height int) int {
	// A very short pane shows one clipped item; otherwise show complete items.
	return max(1, height/layerItemHeight)
}

// LayerWindow keeps the selection visible using the scaffold's pane dimensions.
func LayerWindow(opts ScaffoldOptions) int {
	if len(opts.Layers) == 0 {
		return 0
	}
	l := newScaffoldLayout(opts.Width, opts.Height)
	style := opts.InactivePaneStyle
	if opts.ActivePane == LayerPane {
		style = opts.PaneStyle
	}
	interior := paneInterior(l.columns[0], style)
	capacity := layerCapacity(max(0, interior.height-1))
	selected := min(max(0, opts.SelectedLayer), len(opts.Layers)-1)
	first := min(max(0, opts.FirstVisibleLayer), max(0, len(opts.Layers)-capacity))
	if selected < first {
		first = selected
	} else if selected >= first+capacity {
		first = selected - capacity + 1
	}
	return first
}

func renderLayers(opts ScaffoldOptions, interior paneSize) string {
	height := max(0, interior.height-1) // Reserve the pane heading.
	if interior.width == 0 || height == 0 {
		return ""
	}
	if len(opts.Layers) == 0 {
		return opts.MutedStyle.Render(ansi.Truncate("No layers loaded", interior.width, "…"))
	}
	first := LayerWindow(opts)
	end := min(len(opts.Layers), first+layerCapacity(height))
	items := make([]string, 0, end-first)
	for i := first; i < end; i++ {
		items = append(items, renderLayer(opts, opts.Layers[i], i == opts.SelectedLayer, interior.width))
	}
	rows := strings.Split(strings.Join(items, "\n"), "\n")
	return strings.Join(rows[:min(height, len(rows))], "\n")
}

func renderLayer(opts ScaffoldOptions, item LayerItem, selected bool, width int) string {
	name := opts.MutedStyle
	if selected && opts.ActivePane == LayerPane {
		name = opts.TitleStyle
	}
	background := opts.LayerItemStyle.GetBackground()
	name = name.Bold(selected).Background(background)
	indicator := name.Bold(false).Render("○")
	if item.Complete {
		indicator = opts.SuccessStyle.Background(background).Render("●")
	}
	nameWidth := max(0, width-2)
	// Embedded ANSI resets must not override the item's name colors or fill.
	label := ansi.Truncate(ansi.Strip(item.Name), nameWidth, "…")
	first := indicator + opts.LayerItemStyle.Render(" ") + name.Render(label)
	count := fmt.Sprintf("%d features", item.FeatureCount)
	if item.FeatureCount == 1 {
		count = "1 feature"
	}
	second := opts.MutedStyle.Background(background).Render("  " + count)
	content := ansi.Truncate(first, width, "") + "\n" + ansi.Truncate(second, width, "")
	return opts.LayerItemStyle.Width(width).Height(layerItemHeight).Render(content)
}
