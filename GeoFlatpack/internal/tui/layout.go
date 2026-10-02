package tui

import "charm.land/lipgloss/v2"

type panelLayout struct {
	frame         lipgloss.Style
	box           lipgloss.Style
	bodyStyle     lipgloss.Style
	width         int
	height        int
	headerHeight  int
	headingHeight int
	gap           int
	footerGap     int
	footerHeight  int
	bodyHeight    int
	contentWidth  int
	contentHeight int
}

func (m *Model) layout() panelLayout {
	frame := lipgloss.NewStyle()
	// Restore the original outer padding when the body has room to use it.
	if m.width >= 24 && m.height >= 18 {
		frame = m.styles.frame
	} else if m.width >= 12 && m.height >= 8 {
		frame = frame.Padding(0, 1)
	}

	l := panelLayout{
		frame: frame,
		width: max(1, m.width-frame.GetHorizontalFrameSize()),
		// The footer uses the bottom padding row, as with the original + 1.
		height: max(1, m.height-frame.GetVerticalFrameSize()+min(1, frame.GetPaddingBottom())),
	}

	// Both boxes use the same outline. Drop padding, then borders, before
	// sacrificing Select on a small terminal.
	if l.width >= lipgloss.Width("↵ Select")+2 && l.height >= 9 {
		l.box = m.styles.box
		if l.width < 12 {
			l.box = l.box.Padding(0)
		}
	}

	l.footerHeight = 1 + l.box.GetVerticalFrameSize()
	if l.height >= 2 {
		l.headerHeight = 1
	}

	if l.height >= 12 {
		l.gap = 1
	}
	l.footerGap = l.gap
	if !m.help.ShowAll && m.screen != scaffoldScreen && l.box.GetVerticalFrameSize() > 0 {
		l.gap = 0
		l.footerGap = 0
	}

	l.bodyHeight = max(0, l.height-l.headerHeight-l.footerHeight-l.gap-l.footerGap)
	if m.help.ShowAll || m.screen != scaffoldScreen {
		l.bodyStyle = l.box
	}

	l.contentWidth = max(1, l.width-l.bodyStyle.GetHorizontalFrameSize())
	l.contentHeight = max(0, l.bodyHeight-l.bodyStyle.GetVerticalFrameSize())

	if l.contentHeight > 0 && (m.help.ShowAll || m.screen != scaffoldScreen) {
		l.headingHeight = 1
		l.contentHeight--
		if l.contentHeight >= 7 {
			l.headingHeight++
			l.contentHeight--
		}
	}
	return l
}
