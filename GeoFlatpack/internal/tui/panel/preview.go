package panel

import "charm.land/lipgloss/v2"

// PreviewRegion shares the exact borders, heading gap and content insets used
// by the renderer. Coordinates are relative to the scaffold body.
func PreviewRegion(opts ScaffoldOptions) PickerRect {
	l := newScaffoldLayout(opts.Width, opts.Height)
	size := paneInterior(l.preview, opts.PaneStyle)
	x := 4 * l.gap
	for i := 0; i < 4; i++ {
		x += l.columns[i].width
	}
	box := paneBox(l.preview, opts.PaneStyle)
	if box.GetBorderLeft() {
		x++
	}
	if l.split {
		lower := newLowerLayout(opts, l)
		if lower.divided {
			x += lower.stack.width + 1
			size.width = lower.controls.width
		}
	}
	x += listInset(size)
	y := 1 + listHeadingGap(size)
	if box.GetBorderTop() {
		y++
	}
	return PickerRect{X: x, Y: y, Width: max(0, size.width-2*listInset(size)), Height: max(0, size.height-1-listHeadingGap(size))}
}

func populatePreview(p *paneOptions, opts ScaffoldOptions, size paneSize) {
	p.heading = "Preview"
	if opts.PreviewWarning != "" {
		p.heading += " · " + opts.ScrollHintStyle.Foreground(lipgloss.Color("3")).Render(opts.PreviewWarning)
	}
	p.contentInset, p.contentGap = listInset(size), listHeadingGap(size)
	r := PreviewRegion(opts)
	if opts.PreviewImage {
		p.content = opts.PreviewContent
	} else {
		p.content = clippedInfo(opts.PreviewContent, r.Width, r.Height, opts.MutedStyle)
	}
}
