package tui

import (
	"GeoFlatpack/fgb"
	"GeoFlatpack/internal/app"
	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/internal/tui/workflow"
	"GeoFlatpack/style/maplibre"
	"GeoFlatpack/style/maplibre/svg"
	"context"
	"errors"
	"fmt"
	"image/color"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
	flatbuffers "github.com/google/flatbuffers/go"
)

type testSession struct {
	closed    int
	closeErr  error
	layers    []app.Layer
	icons     map[string]svg.Svg
	writeFunc func([]app.StyleSelection) error
}

func (s *testSession) Write(selections []app.StyleSelection) error {
	if s.writeFunc != nil {
		return s.writeFunc(selections)
	}
	return nil
}

func (s *testSession) Icons() map[string]svg.Svg { return s.icons }

func (s *testSession) Layers() []app.Layer {
	if s.layers != nil {
		return s.layers
	}
	return []app.Layer{{Name: "Roads"}}
}
func (s *testSession) Close() error {
	s.closed++
	return s.closeErr
}

func updateModel(t *testing.T, m *Model, msg tea.Msg) tea.Cmd {
	t.Helper()
	got, cmd := m.Update(msg)
	if got != m {
		t.Fatal("Update changed the model pointer")
	}
	return cmd
}

func completedModel(t *testing.T) *Model {
	t.Helper()
	m := NewModel(app.Options{Input: "test.gml"})
	updateModel(t, m, workflow.PreparedMsg{Session: &testSession{}, Elapsed: time.Second})
	return m
}

func scaffoldModel(t *testing.T) *Model {
	t.Helper()
	m := completedModel(t)
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	return m
}

func TestScaffoldFocusNavigation(t *testing.T) {
	for _, mod := range []tea.KeyMod{tea.ModCtrl, tea.ModAlt} {
		t.Run(fmt.Sprint(mod), func(t *testing.T) {
			m := completedModel(t)
			m.editor.Focus(panel.ControlsPane)
			updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
			if m.screen != scaffoldScreen || m.editor.Pane() != panel.LayerPane {
				t.Fatal("Enter should open scaffold focused on Layer selection")
			}
			for _, tt := range []struct {
				key  rune
				want panel.Pane
			}{
				{tea.KeyRight, panel.CategoryPane}, {tea.KeyRight, panel.FeaturesPane}, {tea.KeyRight, panel.FeatureStylingPane},
				{tea.KeyRight, panel.ControlsPane}, {tea.KeyRight, panel.LayerPane},
				{tea.KeyLeft, panel.ControlsPane}, {tea.KeyLeft, panel.FeatureStylingPane}, {tea.KeyLeft, panel.FeaturesPane},
				{tea.KeyLeft, panel.CategoryPane}, {tea.KeyLeft, panel.LayerPane},
			} {
				cmd := updateModel(t, m, tea.KeyPressMsg{Code: tt.key, Mod: mod})
				if cmd != nil || m.editor.Pane() != tt.want {
					t.Fatalf("focus = %v, want %v; command = %v", m.editor.Pane(), tt.want, cmd != nil)
				}
			}
		})
	}
}

func decodeTerminalKeys(t *testing.T, input string) []tea.KeyPressMsg {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	events := make(chan uv.Event, len(input)+1)
	reader := uv.NewTerminalReader(strings.NewReader(input), "xterm-ghostty")
	if err := reader.StreamEvents(ctx, events); err != nil {
		t.Fatal(err)
	}
	close(events)
	var keys []tea.KeyPressMsg
	for event := range events {
		key, ok := event.(uv.KeyPressEvent)
		if !ok {
			t.Fatalf("input %q decoded to unexpected event %T", input, event)
		}
		keys = append(keys, tea.KeyPressMsg(key))
	}
	return keys
}

func TestPaneNavigationInputStreams(t *testing.T) {
	for _, tt := range []struct {
		name, left, right string
	}{
		{"Ghostty Option arrows", "\x1bb", "\x1bf"},
		{"XTerm Alt arrows", "\x1b[1;3D", "\x1b[1;3C"},
		{"Kitty Alt arrows", "\x1b[57350;3u", "\x1b[57351;3u"},
		{"Control arrows", "\x1b[1;5D", "\x1b[1;5C"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			keys := decodeTerminalKeys(t, strings.Repeat(tt.right, 5)+strings.Repeat(tt.left, 5))
			if len(keys) != 10 {
				t.Fatalf("decoded %d events, want 10", len(keys))
			}
			m := scaffoldModel(t)
			for i, want := range []panel.Pane{
				panel.CategoryPane, panel.FeaturesPane, panel.FeatureStylingPane, panel.ControlsPane, panel.LayerPane,
				panel.ControlsPane, panel.FeatureStylingPane, panel.FeaturesPane, panel.CategoryPane, panel.LayerPane,
			} {
				before := m.View().Content
				updateModel(t, m, keys[i])
				if m.editor.Pane() != want || m.View().Content == before {
					t.Fatalf("input %q: focus = %d, want %d, or highlight failed to change", keys[i], m.editor.Pane(), want)
				}
			}
			updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
			for _, key := range keys {
				updateModel(t, m, key)
				if m.editor.Pane() != panel.LayerPane {
					t.Fatal("input stream changed focus while help was open")
				}
			}
			updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
			if m.editor.Pane() != panel.LayerPane || m.help.ShowAll {
				t.Fatal("closing help did not preserve focus")
			}
			for _, screen := range []screen{loadingScreen, completionScreen, failureScreen} {
				m.screen = screen
				m.editor.Focus(panel.ControlsPane)
				for _, key := range keys {
					updateModel(t, m, key)
					if m.editor.Pane() != panel.ControlsPane || m.screen != screen {
						t.Fatal("pane input changed a processing screen")
					}
				}
			}
		})
	}
}

func TestCommandBindingsNoLongerNavigate(t *testing.T) {
	m := scaffoldModel(t)
	for _, key := range decodeTerminalKeys(t, "\x01\x05\x1b[1;9D\x1b[1;9C\x1b[57350;9u\x1b[57351;9u") {
		updateModel(t, m, key)
		if m.editor.Pane() != panel.LayerPane {
			t.Fatalf("Command input %q still moved focus", key)
		}
	}
	footer := ansi.Strip(m.View().Content)
	if !strings.Contains(footer, "⌥←") || !strings.Contains(footer, "⌥→") || strings.Contains(footer, "⌘") {
		t.Fatal("footer does not describe Option navigation")
	}
}

func TestScaffoldFocusAcrossHelpAndResize(t *testing.T) {
	for pane := panel.LayerPane; pane < panel.PaneCount; pane++ {
		t.Run(fmt.Sprint(pane), func(t *testing.T) {
			m := scaffoldModel(t)
			m.editor.Focus(pane)
			before := m.View().Content
			for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 40}, {Width: 1, Height: 1}, {Width: 80, Height: 24}} {
				updateModel(t, m, size)
				m.View()
				if m.editor.Pane() != pane {
					t.Fatal("scaffold resize changed focus")
				}
			}
			updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
			if !m.help.ShowAll || !strings.Contains(ansi.Strip(m.View().Content), "Keyboard reference") {
				t.Fatal("help did not open")
			}
			for _, mod := range []tea.KeyMod{tea.ModCtrl, tea.ModAlt} {
				for _, key := range []rune{tea.KeyLeft, tea.KeyRight} {
					updateModel(t, m, tea.KeyPressMsg{Code: key, Mod: mod})
					if m.editor.Pane() != pane {
						t.Fatal("pane navigation changed focus while help was open")
					}
				}
			}
			updateModel(t, m, tea.WindowSizeMsg{Width: 80, Height: 12})
			updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
			if m.viewport.YOffset() == 0 {
				t.Fatal("keyboard reference no longer scrolls")
			}
			for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 40}, {Width: 1, Height: 1}, {Width: 80, Height: 24}} {
				updateModel(t, m, size)
				m.View()
				if m.editor.Pane() != pane {
					t.Fatal("resize changed focus")
				}
			}
			updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
			if m.help.ShowAll || m.editor.Pane() != pane || m.View().Content != before {
				t.Fatal("closing help did not restore the selected pane and scaffold")
			}
		})
	}
}

func TestScaffoldIgnoresUnsupportedContentKeys(t *testing.T) {
	m := scaffoldModel(t)
	before := m.View().Content
	for _, code := range []rune{tea.KeyRight, tea.KeyLeft, tea.KeyPgDown, tea.KeyPgUp, tea.KeyHome, tea.KeyEnd} {
		if cmd := updateModel(t, m, tea.KeyPressMsg{Code: code}); cmd != nil {
			t.Fatalf("inactive key %d produced a command", code)
		}
		if m.View().Content != before || m.editor.Pane() != panel.LayerPane || m.screen != scaffoldScreen {
			t.Fatalf("inactive key %d changed scaffold", code)
		}
	}
	if m.viewport.GetContent() != "" || m.viewport.YOffset() != 0 || strings.Count(ansi.Strip(before), "↑/↓") != 5 || strings.Contains(ansi.Strip(before), "PgUp/PgDn Scroll") {
		t.Fatal("scaffold entered the shared viewport or lost its pane scroll hints")
	}
}

func TestScaffoldFocusStyles(t *testing.T) {
	m := scaffoldModel(t)
	updateModel(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	// Coordinates include the existing outer frame and header gap.
	panes := []struct {
		x, y, width int
		heading     string
	}{
		{2, 3, 13, "Layer selection"}, {16, 3, 13, "Category"},
		{30, 3, 13, "Features"}, {44, 3, 13, "Feature styling"},
		{58, 16, 60, "Controls"}, {58, 3, 60, "Preview"},
	}
	for active := panel.LayerPane; active < panel.PaneCount; active++ {
		if m.editor.Pane() != active {
			t.Fatalf("focus = %d, want %d", m.editor.Pane(), active)
		}
		buffer := uv.NewScreenBuffer(120, 40)
		uv.NewStyledString(m.View().Content).Draw(buffer, buffer.Bounds())
		for i, pane := range panes {
			bright := panel.Pane(i) == active || pane.heading == "Preview"
			paneColor := lipgloss.Color("248")
			if bright {
				paneColor = lipgloss.Color("252")
			}
			points := [][2]int{{pane.x, pane.y}, {pane.x + pane.width/2, pane.y}, {pane.x + pane.width - 1, pane.y + 1}}
			if pane.heading == "Controls" {
				points = append(points, [2]int{pane.x, 35}, [2]int{pane.x + pane.width/2, 35}, [2]int{pane.x + pane.width - 1, 35})
			} else if pane.heading == "Preview" {
				points = append(points, [2]int{pane.x, 15}, [2]int{pane.x + pane.width - 1, 15})
			}
			for _, point := range points {
				if !sameColor(buffer.CellAt(point[0], point[1]).Style.Fg, paneColor) {
					t.Errorf("active %d: %s border at %v has wrong color", active, pane.heading, point)
				}
			}
			headingX := pane.x + 2
			if pane.heading == "Controls" || pane.heading == "Preview" {
				headingX += 14
			}
			cell := buffer.CellAt(headingX, pane.y+1)
			if !sameColor(cell.Style.Fg, paneColor) {
				t.Errorf("active %d: %s heading has wrong color", active, pane.heading)
			}
			if (cell.Style.Attrs&uv.AttrBold != 0) != bright {
				t.Errorf("active %d: %s heading has wrong bold attribute", active, pane.heading)
			}
			if cell.Style.Attrs&uv.AttrFaint != 0 {
				t.Errorf("active %d: %s heading must not be faint", active, pane.heading)
			}
		}
		before := m.View().Content
		updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
		updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt})
		updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
		if m.help.ShowAll || m.editor.Pane() != active || m.View().Content != before {
			t.Errorf("active %d: help toggle changed focus styles", active)
		}
		updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt})
	}
}

func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == b
	}
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

func TestScaffoldTerminalSizes(t *testing.T) {
	m := scaffoldModel(t)
	for _, tt := range []struct {
		width, height int
		warning       bool
	}{{120, 40, false}, {120, 24, false}, {80, 24, true}, {119, 40, true}, {120, 23, true}, {24, 12, true}} {
		updateModel(t, m, tea.WindowSizeMsg{Width: tt.width, Height: tt.height})
		view := m.View().Content
		if lipgloss.Width(view) != tt.width || lipgloss.Height(view) != tt.height {
			t.Errorf("%dx%d: view = %dx%d", tt.width, tt.height, lipgloss.Width(view), lipgloss.Height(view))
		}
		if strings.Contains(ansi.Strip(view), "Expand terminal") != tt.warning {
			t.Errorf("%dx%d: incorrect header warning", tt.width, tt.height)
		}
	}
	for width := 1; width <= 24; width++ {
		for height := 1; height <= 18; height++ {
			for _, showHelp := range []bool{false, true} {
				m.help.ShowAll = showHelp
				updateModel(t, m, tea.WindowSizeMsg{Width: width, Height: height})
				view := m.View().Content
				if lipgloss.Width(view) > width || lipgloss.Height(view) > height {
					t.Fatalf("%dx%d (help %v): overflow %dx%d", width, height, showHelp, lipgloss.Width(view), lipgloss.Height(view))
				}
			}
		}
	}
}

func TestProcessingScreensAndQuit(t *testing.T) {
	m := NewModel(app.Options{Input: "test.gml"})
	if m.Init() == nil || m.screen != loadingScreen || !strings.Contains(ansi.Strip(m.View().Content), "Processing input") {
		t.Fatal("initial loading screen changed")
	}
	for _, msg := range []tea.Msg{tea.KeyPressMsg{Code: tea.KeyEnter}, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModCtrl}} {
		updateModel(t, m, msg)
		if m.screen != loadingScreen || m.editor.Pane() != panel.LayerPane {
			t.Fatal("scaffold keys changed loading state")
		}
	}
	updateModel(t, m, spinner.TickMsg{Time: m.startedAt.Add(time.Second)})
	if m.elapsed != time.Second {
		t.Fatal("loading elapsed time no longer updates")
	}
	session := &testSession{}
	updateModel(t, m, workflow.PreparedMsg{Session: session, Elapsed: 2 * time.Second})
	complete := ansi.Strip(m.View().Content)
	if m.screen != completionScreen || m.workflow.Session() != session || !strings.Contains(complete, "Processing complete") || !strings.Contains(complete, "Roads") || !strings.Contains(complete, "Continue") {
		t.Fatal("completion screen changed")
	}
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	cmd := updateModel(t, m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("scaffold quit is inactive")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok || session.closed != 0 {
		t.Fatal("quit must defer session cleanup to Run")
	}

	m = NewModel(app.Options{Input: "test.gml"})
	failure := errors.New("input failed")
	updateModel(t, m, workflow.PreparedMsg{Err: failure})
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.screen != failureScreen || m.err != failure || !strings.Contains(ansi.Strip(m.View().Content), "input failed") {
		t.Fatal("failure screen changed or Enter opened scaffold")
	}
	if cmd := updateModel(t, m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}); cmd == nil {
		t.Fatal("failure screen quit is inactive")
	}

	m = NewModel(app.Options{})
	if cmd := updateModel(t, m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}); cmd != nil || !m.quitting || m.screen != loadingScreen {
		t.Fatal("loading quit must wait for preparation")
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "Finishing processing before exiting") {
		t.Fatal("deferred quit status changed")
	}
	cmd = updateModel(t, m, workflow.PreparedMsg{Session: session})
	if cmd == nil {
		t.Fatal("deferred quit did not finish after preparation")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("expected deferred quit command")
	}
}

func TestRunModelSessionCleanup(t *testing.T) {
	for _, prepareErr := range []error{nil, errors.New("prepare failed")} {
		t.Run(fmt.Sprint(prepareErr), func(t *testing.T) {
			closeErr := errors.New("close failed")
			session := &testSession{closeErr: closeErr}
			started, release := make(chan struct{}), make(chan struct{})
			m := newModel(app.Options{}, func(app.Options, io.Writer) (workflow.Session, error) {
				close(started)
				<-release
				return session, prepareErr
			})
			in, keys := io.Pipe()
			defer in.Close()
			defer keys.Close()
			done := make(chan error, 1)
			go func() {
				done <- runModel(m, in, io.Discard, tea.WithoutRenderer(), tea.WithoutSignalHandler(), tea.WithWindowSize(80, 24))
			}()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("preparation did not start")
			}
			if _, err := keys.Write([]byte{3}); err != nil {
				t.Fatal(err)
			}
			if session.closed != 0 {
				t.Fatal("session closed during preparation")
			}
			close(release)
			select {
			case err := <-done:
				if !errors.Is(err, closeErr) || (prepareErr != nil && !errors.Is(err, prepareErr)) || session.closed != 1 {
					t.Fatalf("cleanup: error %v, closed %d times", err, session.closed)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("quit did not finish and clean up the session")
			}
		})
	}
}

func modelWithLayers(t *testing.T, layers []app.Layer) *Model {
	t.Helper()
	m := NewModel(app.Options{})
	updateModel(t, m, workflow.PreparedMsg{Session: &testSession{layers: layers}})
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	return m
}

func TestConditionalValueChooserGlobalShortcutsAndHints(t *testing.T) {
	b := flatbuffers.NewBuilder(0)
	flat.HeaderStart(b)
	b.Finish(flat.HeaderEnd(b))
	data := &fgb.Fgb{Header: flat.GetRootAsHeader(b.FinishedBytes(), 0)}
	for _, value := range []string{"a", "z"} {
		b = flatbuffers.NewBuilder(0)
		flat.GeometryStartXyVector(b, 2)
		b.PrependFloat64(1)
		b.PrependFloat64(1)
		xy := b.EndVector(2)
		flat.GeometryStart(b)
		flat.GeometryAddType(b, flat.GeometryTypePoint)
		flat.GeometryAddXy(b, xy)
		geometry := flat.GeometryEnd(b)
		flat.FeatureStart(b)
		flat.FeatureAddGeometry(b, geometry)
		b.Finish(flat.FeatureEnd(b))
		data.Features = append(data.Features, fgb.Feature{Raw: *flat.GetRootAsFeature(b.FinishedBytes(), 0), Properties: map[string]any{"kind": value}})
	}
	m := modelWithLayers(t, []app.Layer{{Name: "source", SourceID: "source", Data: data}})
	m.Options.WriteFGB = true
	updateModel(t, m, tea.WindowSizeMsg{Width: 300, Height: 60})
	press := func(code rune) {
		msg := tea.KeyPressMsg{Code: code}
		if code >= 32 && code < 127 {
			msg.Text = string(code)
		}
		updateModel(t, m, msg)
	}
	for _, code := range []rune{tea.KeyEnter, tea.KeyDown, 'c', 'h', tea.KeyEnter, tea.KeyEnter, tea.KeyEnter} {
		press(code)
	}
	if !m.editor.Context().ChoosingValue || !m.editor.Editing() {
		t.Fatal("string value chooser did not open")
	}
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "Select value") || !strings.Contains(view, "Choose value") || strings.Contains(view, "Cursor") {
		t.Fatal("chooser footer does not describe value selection")
	}
	status := m.workflow.Status()
	for _, code := range []rune{'w', 'W', '?', 'd', 'D', 'l', 'L'} {
		press(code)
		if m.help.ShowAll || m.workflow.Busy() || m.workflow.Status() != status || !m.editor.Context().ChoosingValue || m.canWrite() || m.editor.LayerSelectionHidden() {
			t.Fatalf("global shortcut %c intercepted chooser", code)
		}
	}
	press(tea.KeyEscape)
	if m.editor.Editing() || m.editor.Pane() != panel.ControlsPane {
		t.Fatal("Escape did not close chooser back to form")
	}
	press(tea.KeyEnter)
	press(tea.KeyDown)
	press(tea.KeyEnter)
	opts := m.scaffoldOptions(m.layout())
	if opts.Controls[2].Value != `"z"` || opts.Controls[2].Error != "" {
		t.Fatal("Enter did not commit selected string")
	}
	press(tea.KeyEnter)
	press(tea.KeyDown) // Custom action after current string.
	press(tea.KeyEnter)
	if !m.editor.Context().Editing || m.editor.Context().ChoosingValue || !strings.Contains(ansi.Strip(m.View().Content), "Cursor") {
		t.Fatal("custom input did not show text-editing hints")
	}
	press('?')
	press('w')
	if m.help.ShowAll || m.workflow.Status() != status {
		t.Fatal("global shortcuts intercepted custom input")
	}
	press(tea.KeyEscape)
	if !m.editor.Context().ChoosingValue {
		t.Fatal("custom cancellation did not return to chooser")
	}
	press(tea.KeyBackspace)
	if m.editor.Pane() != panel.CategoryPane || m.editor.Editing() {
		t.Fatal("Backspace did not return to Categories")
	}
}

func TestLayerSelectionToggleAndNavigation(t *testing.T) {
	for initial := panel.LayerPane; initial < panel.PaneCount; initial++ {
		t.Run(fmt.Sprint(initial), func(t *testing.T) {
			m := scaffoldModel(t)
			if presentation(m).HideLayerSelection {
				t.Fatal("Layers must be visible by default")
			}
			m.editor.Focus(initial)
			stylePress(t, m, 'l')
			want := initial
			if initial == panel.LayerPane {
				want = panel.CategoryPane
			}
			if !presentation(m).HideLayerSelection || m.editor.Pane() != want {
				t.Fatal("hiding Layers lost focus or failed to transfer it to Category")
			}
			stylePress(t, m, 'L')
			if presentation(m).HideLayerSelection || m.editor.Pane() != want {
				t.Fatal("showing Layers must preserve focus")
			}
			stylePress(t, m, 'L')
			for _, input := range []struct {
				code rune
				mod  tea.KeyMod
			}{
				{tea.KeyRight, tea.ModAlt}, {tea.KeyRight, tea.ModCtrl}, {'f', tea.ModAlt},
				{tea.KeyLeft, tea.ModAlt}, {tea.KeyLeft, tea.ModCtrl}, {'b', tea.ModAlt},
			} {
				m.editor.Focus(panel.CategoryPane)
				panes := []panel.Pane{panel.FeaturesPane, panel.FeatureStylingPane, panel.ControlsPane, panel.CategoryPane}
				if input.code == tea.KeyLeft || input.code == 'b' {
					panes = []panel.Pane{panel.ControlsPane, panel.FeatureStylingPane, panel.FeaturesPane, panel.CategoryPane}
				}
				for step := 0; step < 8; step++ {
					updateModel(t, m, tea.KeyPressMsg{Code: input.code, Mod: input.mod})
					if want := panes[step%len(panes)]; m.editor.Pane() != want {
						t.Fatalf("hidden Layers navigation: focus = %v, want %v", m.editor.Pane(), want)
					}
				}
				if m.editor.Pane() != panel.CategoryPane {
					t.Fatal("navigation did not wrap through the four visible panes")
				}
			}
			stylePress(t, m, 'l')
			updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt})
			if m.editor.Pane() != panel.LayerPane {
				t.Fatal("showing Layers did not restore it to navigation")
			}
		})
	}
}

func TestLayerSelectionPreservesDraftsAndWindows(t *testing.T) {
	_, data := categoryModel(t)
	layers := make([]app.Layer, 12)
	for i := range layers {
		layers[i] = app.Layer{Name: fmt.Sprintf("Source %d", i), SourceID: fmt.Sprint(i), Data: data}
	}
	m := modelWithLayers(t, layers)
	m.Options.WriteFGB, m.Options.WriteStyle = true, true
	updateModel(t, m, tea.WindowSizeMsg{Width: 180, Height: 24})
	selectRow(t, m, panel.LayerPane, 8)
	stylePress(t, m, tea.KeyEnter)
	selectRow(t, m, panel.CategoryPane, 1)
	stylePress(t, m, tea.KeyEnter)
	selectRow(t, m, panel.FeaturesPane, 7)
	stylePress(t, m, tea.KeyEnter)
	addStyle(t, m, maplibre.RenderCircle)
	setProperty(t, m, "circle-radius", "12")
	before, options := presentation(m), m.Options
	if before.SelectedLayer != 8 || before.FirstVisibleLayer == 0 || before.FirstVisibleFeature == 0 {
		t.Fatal("fixture must select and scroll a source and its Features")
	}
	for _, code := range []rune{'l', 'L'} {
		stylePress(t, m, code)
		after := presentation(m)
		after.HideLayerSelection = before.HideLayerSelection
		if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(m.Options, options) {
			t.Fatal("toggle changed source selection, scrolling, styling, controls, or output settings")
		}
	}
	// Keep an unfinished input while focus moves to Styling, then toggle there.
	typeControl(t, m, "123")
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt})
	stylePress(t, m, 'l')
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt})
	if !m.editor.Editing() || !strings.Contains(ansi.Strip(presentation(m).Controls[0].Value), "123") {
		t.Fatal("toggle discarded an unfinished control draft")
	}
}

func TestLayerSelectionGuardsAndSessionVisibility(t *testing.T) {
	m := scaffoldModel(t)
	updateModel(t, m, tea.WindowSizeMsg{Width: 240, Height: 40})
	if !strings.Contains(ansi.Strip(m.View().Content), "L Hide layers") {
		t.Fatal("visible Layers need a contextual hide hint")
	}
	stylePress(t, m, 'l')
	if !strings.Contains(ansi.Strip(m.View().Content), "L Show layers") {
		t.Fatal("hidden Layers need a contextual show hint")
	}
	stylePress(t, m, '?')
	if !strings.Contains(ansi.Strip(m.View().Content), "Show or hide Layer selection") {
		t.Fatal("keyboard reference does not explain the toggle")
	}
	stylePress(t, m, 'L')
	updateModel(t, m, tea.WindowSizeMsg{Width: 81, Height: 20})
	stylePress(t, m, '?')
	if !presentation(m).HideLayerSelection || m.editor.Pane() != panel.CategoryPane {
		t.Fatal("help or resize lost visibility or focus")
	}
	for _, screen := range []screen{loadingScreen, failureScreen, completionScreen} {
		m.screen = screen
		stylePress(t, m, 'l')
		if !m.editor.LayerSelectionHidden() {
			t.Fatal("shortcut toggled Layers outside the editor")
		}
	}
	stylePress(t, m, tea.KeyEnter)
	if !presentation(m).HideLayerSelection || m.editor.Pane() != panel.CategoryPane {
		t.Fatal("reopening editor lost visibility or focused hidden Layers")
	}

	m, _ = writeReadyModel(t)
	selectProperty(t, m, "circle-radius")
	typeControl(t, m, "")
	for _, code := range []rune{'l', 'L'} {
		updateModel(t, m, tea.KeyPressMsg{Code: code, Text: string(code)})
	}
	view := ansi.Strip(m.View().Content)
	if m.editor.LayerSelectionHidden() || !strings.Contains(ansi.Strip(presentation(m).Controls[0].Value), "lL") || strings.Contains(view, "L Hide layers") {
		t.Fatal("toggle intercepted input text or appeared while editing")
	}
	stylePress(t, m, tea.KeyEscape)
	stylePress(t, m, 'l')
	cmd := writeKey(t, m, 'w')
	if cmd == nil || !m.workflow.Busy() {
		t.Fatal("fixture did not start writing")
	}
	for _, code := range []rune{'l', 'L'} {
		stylePress(t, m, code)
		if !m.editor.LayerSelectionHidden() {
			t.Fatal("writing allowed the toggle")
		}
	}
	if strings.Contains(ansi.Strip(m.View().Content), "L Show layers") {
		t.Fatal("writing advertised the toggle")
	}
	updateModel(t, m, cmd())
	if !presentation(m).HideLayerSelection {
		t.Fatal("writing reset layer visibility")
	}
}

func TestLayerSelectionWidthAllocation(t *testing.T) {
	for _, tt := range []struct {
		width   int
		columns [4]int // Category, Features, Styling, Preview/Controls.
	}{
		{29, [4]int{5, 5, 3, 13}},
		{38, [4]int{6, 7, 4, 18}},
		{76, [4]int{12, 13, 8, 40}},
		{80, [4]int{14, 14, 9, 40}},
		{116, [4]int{20, 20, 13, 60}},
		{121, [4]int{21, 22, 14, 61}},
	} {
		t.Run(fmt.Sprint(tt.width), func(t *testing.T) {
			opts := panel.ScaffoldOptions{Width: tt.width, Height: 17, HideLayerSelection: true,
				PaneStyle: newStyles().pane, InactivePaneStyle: newStyles().inactivePane}
			view := ansi.Strip(panel.Scaffold(opts))
			tops := make([]string, 4)
			for i, width := range tt.columns {
				tops[i] = "╭" + strings.Repeat("─", width-2) + "╮"
			}
			if got := strings.Split(view, "\n")[0]; got != strings.Join(tops, " ") {
				t.Fatalf("top borders = %q, want widths %v", got, tt.columns)
			}
			if lipgloss.Width(view) != tt.width || lipgloss.Height(view) != 17 || strings.Contains(view, "Layer selection") {
				t.Fatal("hidden Layers overflowed or remained visible")
			}
		})
	}
	for width := 1; width <= 30; width++ {
		for height := 1; height <= 12; height++ {
			opts := panel.ScaffoldOptions{Width: width, Height: height, HideLayerSelection: true,
				PaneStyle: newStyles().pane, InactivePaneStyle: newStyles().inactivePane}
			view := panel.Scaffold(opts)
			if lipgloss.Width(view) != width || lipgloss.Height(view) != height {
				t.Fatalf("%dx%d: hidden Layers rendered %dx%d", width, height, lipgloss.Width(view), lipgloss.Height(view))
			}
		}
	}
}

func TestLayerSelectionPreservesPreviewAndControlsGeometry(t *testing.T) {
	for _, stack := range []bool{false, true} {
		for width := 1; width <= 220; width++ {
			for _, height := range []int{1, 3, 5, 17, 40} {
				opts := panel.ScaffoldOptions{Width: width, Height: height, ShowStack: stack,
					PaneStyle: newStyles().pane, InactivePaneStyle: newStyles().inactivePane}
				preview, controls := panel.PreviewRegion(opts), panel.ControlsRegion(opts)
				w, h := panel.ControlsDimensions(opts)
				opts.HideLayerSelection = true
				hiddenW, hiddenH := panel.ControlsDimensions(opts)
				if panel.PreviewRegion(opts) != preview || panel.ControlsRegion(opts) != controls || w != hiddenW || h != hiddenH {
					t.Fatalf("%dx%d stack=%v: toggle moved or resized Preview/Controls", width, height, stack)
				}
			}
		}
	}
	m, _ := writeReadyModel(t)
	selectProperty(t, m, "circle-color")
	m.editor.Focus(panel.ControlsPane)
	updateModel(t, m, tea.WindowSizeMsg{Width: 220, Height: 40})
	visibleDimensions := m.previewDimensions()
	for _, hidden := range []bool{false, true} {
		if hidden {
			stylePress(t, m, 'l')
		}
		if m.previewDimensions() != visibleDimensions {
			t.Fatal("toggle changed terminal preview placement")
		}
		opts, l := presentation(m), m.layout()
		region := panel.ControlsRegion(opts)
		geometry := panel.ColorPickerGeometry(*opts.ColorPicker, region.Width, region.Height)
		square := geometry.Controls[panel.PickerSquare]
		mouse := tea.Mouse{Button: tea.MouseLeft,
			X: l.frame.GetPaddingLeft() + region.X + square.X - geometry.OffsetX + square.Width - 1,
			Y: l.frame.GetPaddingTop() + l.headerHeight + l.gap + region.Y + square.Y - geometry.OffsetY}
		updateModel(t, m, tea.MouseClickMsg(mouse))
		picker := presentation(m).ColorPicker
		if !m.editor.Editing() || picker.Saturation != 1 || picker.Lightness != 1 {
			t.Fatalf("hidden=%v: mouse did not hit the rendered square's top right", hidden)
		}
		stylePress(t, m, 'L')
		if m.editor.LayerSelectionHidden() != hidden {
			t.Fatal("toggle intercepted active color adjustment")
		}
		updateModel(t, m, tea.MouseReleaseMsg(mouse))
		stylePress(t, m, tea.KeyEscape)
	}
}

func TestSessionLayerPresentation(t *testing.T) {
	layers := []app.Layer{
		{Name: "Nil"},
		{Name: "Empty", Data: &fgb.Fgb{}},
		{Name: "Single", Data: &fgb.Fgb{Features: make([]fgb.Feature, 1)}},
		{Name: "Multiple", Data: &fgb.Fgb{Features: make([]fgb.Feature, 12)}},
	}
	m := modelWithLayers(t, layers)
	updateModel(t, m, tea.WindowSizeMsg{Width: 180, Height: 40})
	if m.scaffoldOptions(m.layout()).SelectedLayer != 0 || len(m.scaffoldOptions(m.layout()).Layers) != len(layers) {
		t.Fatal("scaffold must initially select the first loaded layer")
	}
	for i, detail := range []string{"0 features", "0 features", "1 feature", "12 features"} {
		item := m.scaffoldOptions(m.layout()).Layers[i]
		if item.Name != layers[i].Name || item.Detail != detail || item.Status != panel.ListUnopened {
			t.Errorf("layer %d: unexpected presentation %+v", i, item)
		}
	}
	view := ansi.Strip(m.View().Content)
	last := -1
	for _, name := range []string{"○ Nil", "○ Empty", "○ Single", "○ Multiple"} {
		i := strings.Index(view, name)
		if i <= last {
			t.Fatalf("layer %q missing or out of source order:\n%s", name, view)
		}
		last = i
	}
	if !strings.Contains(view, "1 feature ") || !strings.Contains(view, "12 features") || strings.Count(view, "0 features") != 2 {
		t.Fatal("session feature counts not rendered")
	}
	buffer := uv.NewScreenBuffer(m.width, m.height)
	uv.NewStyledString(m.View().Content).Draw(buffer, buffer.Bounds())
	if !sameColor(buffer.CellAt(4, 6).Style.Bg, lipgloss.Color("236")) {
		t.Fatal("model did not pass the dark item background to the scaffold")
	}
}

func TestLayerSelectionAndScrolling(t *testing.T) {
	layers := make([]app.Layer, 10)
	for i := range layers {
		layers[i].Name = fmt.Sprintf("Layer %02d", i)
	}
	m := modelWithLayers(t, layers)
	updateModel(t, m, tea.WindowSizeMsg{Width: 180, Height: 24}) // Four complete items above the scroll hint.
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.scaffoldOptions(m.layout()).SelectedLayer != 0 || m.scaffoldOptions(m.layout()).FirstVisibleLayer != 0 {
		t.Fatal("Up must clamp at the first layer")
	}
	for i := 1; i <= 12; i++ {
		cmd := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
		want := min(i, len(layers)-1)
		if cmd != nil || m.scaffoldOptions(m.layout()).SelectedLayer != want || m.scaffoldOptions(m.layout()).FirstVisibleLayer != max(0, want-3) {
			t.Fatalf("Down %d: selected %d, first %d", i, m.scaffoldOptions(m.layout()).SelectedLayer, m.scaffoldOptions(m.layout()).FirstVisibleLayer)
		}
		view := ansi.Strip(m.View().Content)
		if !strings.Contains(view, layers[want].Name) || strings.Count(view, "○") != 4 || strings.Contains(view, "●") {
			t.Fatalf("Down %d: selected layer not visible or incorrect indicators", i)
		}
	}
	for i := 8; i >= -2; i-- {
		updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
		want := max(0, i)
		if m.scaffoldOptions(m.layout()).SelectedLayer != want || !strings.Contains(ansi.Strip(m.View().Content), layers[want].Name) {
			t.Fatalf("Up: selected %d, want visible %d", m.scaffoldOptions(m.layout()).SelectedLayer, want)
		}
	}
}

func TestLayerSelectionPreserved(t *testing.T) {
	layers := make([]app.Layer, 10)
	for i := range layers {
		layers[i].Name = fmt.Sprintf("Layer %02d", i)
	}
	m := modelWithLayers(t, layers)
	updateModel(t, m, tea.WindowSizeMsg{Width: 180, Height: 24})
	for i := 0; i < 7; i++ {
		updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	for _, pane := range []panel.Pane{panel.CategoryPane, panel.FeaturesPane, panel.FeatureStylingPane, panel.ControlsPane} {
		updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt})
		for _, code := range []rune{tea.KeyDown, tea.KeyUp, tea.KeyEnter} {
			updateModel(t, m, tea.KeyPressMsg{Code: code})
			if m.editor.Pane() != pane || m.scaffoldOptions(m.layout()).SelectedLayer != 7 || m.scaffoldOptions(m.layout()).FirstVisibleLayer != 4 {
				t.Fatal("another pane changed the layer selection/window")
			}
		}
	}
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt})
	before := m.View().Content
	updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	for _, code := range []rune{tea.KeyUp, tea.KeyDown} {
		updateModel(t, m, tea.KeyPressMsg{Code: code})
	}
	updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	if m.scaffoldOptions(m.layout()).SelectedLayer != 7 || m.scaffoldOptions(m.layout()).FirstVisibleLayer != 4 || m.View().Content != before {
		t.Fatal("help toggles changed selection or list window")
	}
	for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 12}, {Width: 1, Height: 1}, {Width: 180, Height: 40}, {Width: 180, Height: 24}} {
		updateModel(t, m, size)
		if m.scaffoldOptions(m.layout()).SelectedLayer != 7 || m.scaffoldOptions(m.layout()).FirstVisibleLayer != panel.LayerWindow(m.scaffoldOptions(m.layout())) {
			t.Fatal("resize lost selection or failed to recalculate window")
		}
		if size.Height >= 24 && !strings.Contains(ansi.Strip(m.View().Content), layers[7].Name) {
			t.Fatal("resize hid the selected layer")
		}
	}
	updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	updateModel(t, m, tea.WindowSizeMsg{Width: 120, Height: 12})
	updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	if m.scaffoldOptions(m.layout()).SelectedLayer != 7 || m.scaffoldOptions(m.layout()).FirstVisibleLayer != 7 {
		t.Fatal("returning from help must recalculate the shortened list window")
	}
}

func TestEmptyLayerSelection(t *testing.T) {
	for _, session := range []workflow.Session{nil, &testSession{layers: []app.Layer{}}} {
		m := NewModel(app.Options{})
		updateModel(t, m, workflow.PreparedMsg{Session: session})
		updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
		updateModel(t, m, tea.WindowSizeMsg{Width: 180, Height: 24})
		for _, code := range []rune{tea.KeyUp, tea.KeyDown, tea.KeyEnter} {
			updateModel(t, m, tea.KeyPressMsg{Code: code})
		}
		if m.editor.Pane() != panel.LayerPane || m.scaffoldOptions(m.layout()).SelectedLayer != 0 || m.scaffoldOptions(m.layout()).FirstVisibleLayer != 0 || len(m.scaffoldOptions(m.layout()).Layers) != 0 || !strings.Contains(ansi.Strip(m.View().Content), "No layers loaded") {
			t.Fatal("empty layer list did not stay stable")
		}
	}
}

func TestEnterLayerStatusAndFocus(t *testing.T) {
	m := modelWithLayers(t, []app.Layer{{Name: "Roads"}, {Name: "Stops"}, {Name: "Zones"}})
	updateModel(t, m, tea.WindowSizeMsg{Width: 120, Height: 24})
	if cmd := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter}); cmd != nil {
		t.Fatal("opening a layer must not start processing")
	}
	if m.editor.Pane() != panel.CategoryPane || m.scaffoldOptions(m.layout()).SelectedLayer != 0 || m.scaffoldOptions(m.layout()).Layers[0].Status != panel.ListIncomplete {
		t.Fatal("Enter must mark the current layer incomplete and focus Categories")
	}
	buffer := uv.NewScreenBuffer(m.width, m.height)
	uv.NewStyledString(m.View().Content).Draw(buffer, buffer.Bounds())
	indicator, name := buffer.CellAt(4, 6), buffer.CellAt(6, 6)
	if indicator.Content != "●" || !sameColor(indicator.Style.Fg, lipgloss.Color("3")) {
		t.Fatal("opened layer did not show a yellow filled circle")
	}
	if !sameColor(name.Style.Fg, lipgloss.Color("252")) || !sameColor(name.Style.Bg, lipgloss.Color("236")) || name.Style.Attrs&uv.AttrBold == 0 {
		t.Fatal("selected layer must keep its full highlight after focus leaves Layers")
	}
	for _, code := range []rune{tea.KeyEnter, tea.KeyUp, tea.KeyDown} {
		updateModel(t, m, tea.KeyPressMsg{Code: code})
		if m.editor.Pane() != panel.CategoryPane || m.scaffoldOptions(m.layout()).SelectedLayer != 0 || m.scaffoldOptions(m.layout()).Layers[1].Status != panel.ListUnopened {
			t.Fatal("Category inputs must remain inactive")
		}
	}
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt})
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.scaffoldOptions(m.layout()).Layers[0].Status != panel.ListIncomplete || m.scaffoldOptions(m.layout()).Layers[1].Status != panel.ListUnopened {
		t.Fatal("moving selection must not change readiness")
	}
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.editor.Pane() != panel.CategoryPane || m.scaffoldOptions(m.layout()).SelectedLayer != 1 || m.scaffoldOptions(m.layout()).Layers[0].Status != panel.ListIncomplete || m.scaffoldOptions(m.layout()).Layers[1].Status != panel.ListIncomplete || m.scaffoldOptions(m.layout()).Layers[2].Status != panel.ListUnopened {
		t.Fatal("multiple opened layers must remain incomplete independently")
	}
	buffer = uv.NewScreenBuffer(m.width, m.height)
	uv.NewStyledString(m.View().Content).Draw(buffer, buffer.Bounds())
	oldIndicator, oldName := buffer.CellAt(4, 6), buffer.CellAt(6, 6)
	newIndicator, newName := buffer.CellAt(4, 9), buffer.CellAt(6, 9)
	if !sameColor(oldIndicator.Style.Fg, lipgloss.Color("3")) || !sameColor(newIndicator.Style.Fg, lipgloss.Color("3")) || !sameColor(oldName.Style.Bg, lipgloss.Color("234")) || !sameColor(newName.Style.Bg, lipgloss.Color("236")) {
		t.Fatal("incomplete status must stay yellow independently of the selected fill")
	}
	// Reopening the first layer keeps its status and moves back to Categories.
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt})
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.editor.Pane() != panel.CategoryPane || m.scaffoldOptions(m.layout()).Layers[0].Status != panel.ListIncomplete || m.scaffoldOptions(m.layout()).Layers[1].Status != panel.ListIncomplete {
		t.Fatal("reopening a layer changed its readiness")
	}
}

func TestLayerStatusesPreserved(t *testing.T) {
	layers := make([]app.Layer, 10)
	for i := range layers {
		layers[i].Name = fmt.Sprintf("Layer %02d", i)
	}
	m := modelWithLayers(t, layers)
	updateModel(t, m, tea.WindowSizeMsg{Width: 120, Height: 24})
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt})
	for i := 0; i < 7; i++ {
		updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	for i := 0; i < 5; i++ {
		updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt})
	}
	updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	for _, code := range []rune{tea.KeyEnter, tea.KeyUp, tea.KeyDown} {
		updateModel(t, m, tea.KeyPressMsg{Code: code})
	}
	updateModel(t, m, tea.WindowSizeMsg{Width: 120, Height: 12})
	updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	for _, size := range []tea.WindowSizeMsg{{Width: 1, Height: 1}, {Width: 180, Height: 40}, {Width: 120, Height: 24}} {
		updateModel(t, m, size)
		m.View()
	}
	if m.editor.Pane() != panel.CategoryPane || m.scaffoldOptions(m.layout()).SelectedLayer != 7 || m.scaffoldOptions(m.layout()).FirstVisibleLayer != 4 {
		t.Fatal("focus, help, or resizing lost the selected layer/window")
	}
	for i, item := range m.scaffoldOptions(m.layout()).Layers {
		want := panel.ListUnopened
		if i == 0 || i == 7 {
			want = panel.ListIncomplete
		}
		if item.Status != want {
			t.Errorf("layer %d: readiness changed to %d, want %d", i, item.Status, want)
		}
	}
}

func TestEnterWithLayerHelpOpen(t *testing.T) {
	m := scaffoldModel(t)
	updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m.help.ShowAll || m.editor.Pane() != panel.LayerPane || m.scaffoldOptions(m.layout()).Layers[0].Status != panel.ListUnopened {
		t.Fatal("Enter while help is open must not open or mark a layer")
	}
	updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.editor.Pane() != panel.CategoryPane || m.scaffoldOptions(m.layout()).Layers[0].Status != panel.ListIncomplete {
		t.Fatal("Enter did not resume when help closed")
	}
}
