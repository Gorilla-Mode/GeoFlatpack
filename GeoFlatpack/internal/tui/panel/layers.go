package panel

// LayerWindow keeps the selection visible using the scaffold's pane dimensions.
func LayerWindow(opts ScaffoldOptions) int {
	l := newScaffoldLayout(opts.Width, opts.Height)
	style := opts.InactivePaneStyle
	if opts.ActivePane == LayerPane {
		style = opts.PaneStyle
	}
	return ListWindow(layerListOptions(opts, paneInterior(l.columns[0], style)))
}

func layerListOptions(opts ScaffoldOptions, interior paneSize) ListOptions {
	return ListOptions{
		Width: interior.width, Height: max(0, interior.height-1), // Reserve the pane heading.
		Items: opts.Layers, Selected: opts.SelectedLayer, FirstVisible: opts.FirstVisibleLayer,
		Styles: opts.ListStyles, EmptyText: "No layers loaded",
	}
}
