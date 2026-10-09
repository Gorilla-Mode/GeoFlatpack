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

// CategoryWindow uses the same list rules inside the Category pane.
func CategoryWindow(opts ScaffoldOptions) int {
	l := newScaffoldLayout(opts.Width, opts.Height)
	style := opts.InactivePaneStyle
	if opts.ActivePane == CategoryPane {
		style = opts.PaneStyle
	}

	return ListWindow(categoryListOptions(opts, paneInterior(l.columns[1], style)))
}

// FeaturesWindow keeps the cursor or chosen styling target visible.
func FeaturesWindow(opts ScaffoldOptions) int {
	l := newScaffoldLayout(opts.Width, opts.Height)
	style := opts.InactivePaneStyle
	if opts.ActivePane == FeaturesPane {
		style = opts.PaneStyle
	}

	return ListWindow(featureListOptions(opts, paneInterior(l.columns[2], style)))
}

func featureListOptions(opts ScaffoldOptions, interior paneSize) ListOptions {
	list := ListOptions{
		Width: interior.width - 2*listInset(interior), Height: listContentHeight(interior),
		Items: opts.Features, Selected: opts.SelectedFeature, FirstVisible: opts.FirstVisibleFeature,
		Styles: opts.ListStyles, EmptyText: opts.FeatureEmptyText,
	}

	if opts.ActivePane != FeaturesPane {
		list.HideSelection = !opts.FeatureChosen
		if opts.FeatureChosen {
			list.Selected = opts.ChosenFeature
		}
	}

	return list
}

func categoryListOptions(opts ScaffoldOptions, interior paneSize) ListOptions {
	list := ListOptions{
		Width: interior.width - 2*listInset(interior), Height: listContentHeight(interior),
		Items: opts.Categories, Selected: opts.SelectedCategory, FirstVisible: opts.FirstVisibleCategory,
		Styles: opts.ListStyles, EmptyText: opts.CategoryEmptyText,
	}

	if opts.ActivePane != CategoryPane {
		// Outside this pane, show the committed choice rather than the cursor.
		list.HideSelection = true
		for i, item := range opts.Categories {
			if item.Status == ListActive {
				list.Selected, list.HideSelection = i, false
				break
			}
		}
	}

	return list
}

func layerListOptions(opts ScaffoldOptions, interior paneSize) ListOptions {
	return ListOptions{
		Width: interior.width - 2*listInset(interior), Height: listContentHeight(interior),
		Items: opts.Layers, Selected: opts.SelectedLayer, FirstVisible: opts.FirstVisibleLayer,
		Styles: opts.ListStyles, EmptyText: "No layers loaded",
	}
}

func listContentHeight(interior paneSize) int {
	return max(0, interior.height-1-listHeadingGap(interior)-listFooterHeight(interior))
}

func listInset(interior paneSize) int {
	if interior.width >= 3 {
		return 1
	}

	return 0
}

func listHeadingGap(interior paneSize) int {
	if interior.height-1-listFooterHeight(interior) >= 2 {
		return 1
	}

	return 0
}

func listFooterHeight(interior paneSize) int {
	// Keep space for the heading and selected item's name in short panes.
	if interior.width > 0 && interior.height >= 3 {
		return 1
	}

	return 0
}
