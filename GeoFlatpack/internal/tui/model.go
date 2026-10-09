// Package tui provides the terminal frontend.
package tui

import (
	"GeoFlatpack/internal/app"
	"GeoFlatpack/internal/tui/editor"
	"GeoFlatpack/internal/tui/terminalpreview"
	"GeoFlatpack/internal/tui/workflow"
	"GeoFlatpack/style/maplibre/svg"
	"fmt"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

type screen int

const (
	loadingScreen screen = iota
	completionScreen
	scaffoldScreen
	failureScreen
)

// Model coordinates screens, global shortcuts, and the editor, preview, and
// workflow subsystems. Run owns preparation and cleanup.
type Model struct {
	Options    app.Options
	help       help.Model
	keys       keyMap
	detailKeys detailKeyMap
	styles     styles
	width      int
	height     int
	spinner    spinner.Model
	viewport   viewport.Model
	screen     screen
	editor     *editor.Editor
	startedAt  time.Time
	elapsed    time.Duration
	quitting   bool
	err        error
	workflow   *workflow.Workflow
	preview    *terminalpreview.State
}

var _ tea.Model = (*Model)(nil)

// NewModel creates a loading screen; Init schedules input preparation.
func NewModel(opts app.Options) *Model {
	return newModel(opts, workflow.PrepareInput)
}

func newModel(opts app.Options, prepare workflow.PrepareFunc) *Model {
	s := newStyles()
	h := help.New()
	h.Styles = s.help
	m := &Model{
		Options:    opts,
		help:       h,
		keys:       newKeyMap(),
		detailKeys: newDetailKeyMap(),
		styles:     s,
		editor:     editor.New(),
		width:      80,
		height:     24,
		spinner:    spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(s.title)),
		viewport:   viewport.New(),
		startedAt:  time.Now(),
		workflow:   workflow.New(prepare),
	}

	m.viewport.SetHorizontalStep(0)
	m.refreshViewport()

	return m
}

func (m *Model) Init() tea.Cmd {
	var graphics tea.Cmd
	if m.preview != nil {
		graphics = m.preview.Init()
	}

	return tea.Batch(m.workflow.Prepare(m.Options), m.spinner.Tick, graphics)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.editor.SetContext(m.Options.WriteFGB, m.editorVisible())
	var graphics tea.Cmd
	if m.preview != nil {
		graphics = m.preview.Input(msg, m.previewDimensions())
	}

	switch msg := msg.(type) {
	case workflow.WriteDoneMsg:
		if !m.workflow.Complete(msg) {
			break
		}

		if m.quitting {
			return m, m.quitCommand()
		}
	case workflow.PreparedMsg:
		m.elapsed, m.err = msg.Elapsed, msg.Err
		if m.quitting {
			return m, m.quitCommand()
		}

		if msg.Err != nil {
			m.screen = failureScreen
		} else {
			m.workflow.Prepared(msg)
			var layers []app.Layer
			var icons map[string]svg.Svg
			if msg.Session != nil {
				layers, icons = msg.Session.Layers(), msg.Session.Icons()
			}

			m.editor.Load(layers, icons, msg.Session != nil)
			m.screen = completionScreen
		}

		m.viewport.GotoTop()
	case spinner.TickMsg:
		if m.screen == loadingScreen {
			m.elapsed = max(0, msg.Time.Sub(m.startedAt))
			m.spinner, cmd = m.spinner.Update(msg)
		}
	case tea.WindowSizeMsg:
		m.editor.StopDragging()
		m.width, m.height = max(1, msg.Width), max(1, msg.Height)
	case tea.MouseClickMsg, tea.MouseMotionMsg, tea.MouseReleaseMsg:
		if m.screen == scaffoldScreen && !m.help.ShowAll && !m.workflow.Busy() {
			l := m.layout()
			cmd = m.editor.MouseInput(msg, m.scaffoldOptions(l), l.frame.GetPaddingLeft(), l.frame.GetPaddingTop()+l.headerHeight+l.gap)
		}
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.screen == loadingScreen || m.workflow.Busy() {
				m.quitting = true
				if m.workflow.Busy() {
					m.workflow.Quitting()
				}

				break
			}

			return m, m.quitCommand()
		case m.workflow.Busy():
			// The writer exclusively owns the session until its result arrives.
		case key.Matches(msg, m.keys.PreviewSample) && m.screen == scaffoldScreen && !m.help.ShowAll && !m.editor.Editing() && m.preview != nil && m.preview.Supported():
			m.preview.ToggleSample()
		case key.Matches(msg, m.keys.WriteFiles) && m.screen == scaffoldScreen && !m.help.ShowAll && !m.editor.Editing():
			cmd = m.startWrite()
		case key.Matches(msg, m.keys.Select) && m.screen == completionScreen:
			m.screen = scaffoldScreen
			m.editor.Open()
			m.viewport.GotoTop()
			m.help.ShowAll = false
			m.keys.Help.SetHelp("?", "help")
		case key.Matches(msg, m.keys.Help) && !m.editor.Editing():
			m.editor.StopDragging()
			m.help.ShowAll = !m.help.ShowAll
			m.viewport.GotoTop()
			if m.help.ShowAll {
				m.keys.Help.SetHelp("?", "hide help")
			} else {
				m.keys.Help.SetHelp("?", "help")
			}
		default:
			if m.editorVisible() {
				cmd = m.editor.Input(msg)
			} else {
				m.refreshViewport()
				m.viewport, cmd = m.viewport.Update(msg)
			}
		}
	default:
		if !m.workflow.Busy() && m.editor.Editing() {
			cmd = m.editor.Input(msg)
		}
	}

	m.refreshViewport()

	return m, tea.Batch(cmd, graphics, m.refreshPreview())
}

func (m *Model) editorVisible() bool {
	return m.screen == scaffoldScreen && !m.help.ShowAll && !m.workflow.Busy()
}

func (m *Model) canWrite() bool {
	return m.editorVisible() && !m.editor.Editing() && (m.Options.WriteFGB || m.Options.WriteStyle) && m.workflow.Session() != nil && m.editor.Ready()
}

func (m *Model) startWrite() tea.Cmd {
	m.editor.RefreshReadiness()
	var err error
	if !m.Options.WriteFGB && !m.Options.WriteStyle {
		err = fmt.Errorf("No outputs enabled")
	} else if m.workflow.Session() == nil || !m.editor.Ready() {
		err = fmt.Errorf("Complete all layers before writing")
	}

	var selections []app.StyleSelection
	if err == nil {
		selections, err = m.editor.Selections()
	}

	return m.workflow.Start(selections, err)
}

func (m *Model) quitCommand() tea.Cmd { return m.preview.Quit() }

func (m *Model) refreshPreview() tea.Cmd {
	if !m.preview.Supported() {
		return nil
	}

	visible := m.editorVisible() && !m.quitting && m.workflow.Session() != nil
	var snapshot editor.Target
	var err error
	if visible {
		snapshot, err = m.editor.PreviewTarget()
	}

	return m.preview.Refresh(snapshot, err, m.previewDimensions(), visible)
}
