package tui

import (
	"GeoFlatpack/fgb"
	"GeoFlatpack/internal/app"
	"GeoFlatpack/internal/tui/panel"
	"context"
	"errors"
	"fmt"
	"image/color"
	"io"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

type testSession struct {
	closed   int
	closeErr error
	layers   []app.Layer
}

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
	updateModel(t, m, preparedMsg{session: &testSession{}, elapsed: time.Second})
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
			m.activePane = panel.ControlsPane
			updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
			if m.screen != scaffoldScreen || m.activePane != panel.LayerPane {
				t.Fatal("Enter should open scaffold focused on Layer selection")
			}
			for _, tt := range []struct {
				key  rune
				want panel.Pane
			}{
				{tea.KeyRight, panel.CategoryPane}, {tea.KeyRight, panel.FeaturePane},
				{tea.KeyRight, panel.ControlsPane}, {tea.KeyRight, panel.LayerPane},
				{tea.KeyLeft, panel.ControlsPane}, {tea.KeyLeft, panel.FeaturePane},
				{tea.KeyLeft, panel.CategoryPane}, {tea.KeyLeft, panel.LayerPane},
			} {
				cmd := updateModel(t, m, tea.KeyPressMsg{Code: tt.key, Mod: mod})
				if cmd != nil || m.activePane != tt.want {
					t.Fatalf("focus = %v, want %v; command = %v", m.activePane, tt.want, cmd != nil)
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
			keys := decodeTerminalKeys(t, strings.Repeat(tt.right, 4)+strings.Repeat(tt.left, 4))
			if len(keys) != 8 {
				t.Fatalf("decoded %d events, want 8", len(keys))
			}
			m := scaffoldModel(t)
			for i, want := range []panel.Pane{
				panel.CategoryPane, panel.FeaturePane, panel.ControlsPane, panel.LayerPane,
				panel.ControlsPane, panel.FeaturePane, panel.CategoryPane, panel.LayerPane,
			} {
				before := m.View().Content
				updateModel(t, m, keys[i])
				if m.activePane != want || m.View().Content == before {
					t.Fatalf("input %q: focus = %d, want %d, or highlight failed to change", keys[i], m.activePane, want)
				}
			}
			updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
			for _, key := range keys {
				updateModel(t, m, key)
				if m.activePane != panel.LayerPane {
					t.Fatal("input stream changed focus while help was open")
				}
			}
			updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
			if m.activePane != panel.LayerPane || m.help.ShowAll {
				t.Fatal("closing help did not preserve focus")
			}
			for _, screen := range []screen{loadingScreen, completionScreen, failureScreen} {
				m.screen, m.activePane = screen, panel.ControlsPane
				for _, key := range keys {
					updateModel(t, m, key)
					if m.activePane != panel.ControlsPane || m.screen != screen {
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
		if m.activePane != panel.LayerPane {
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
			m.activePane = pane
			before := m.View().Content
			for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 40}, {Width: 1, Height: 1}, {Width: 80, Height: 24}} {
				updateModel(t, m, size)
				m.View()
				if m.activePane != pane {
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
					if m.activePane != pane {
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
				if m.activePane != pane {
					t.Fatal("resize changed focus")
				}
			}
			updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
			if m.help.ShowAll || m.activePane != pane || m.View().Content != before {
				t.Fatal("closing help did not restore the selected pane and scaffold")
			}
		})
	}
}

func TestScaffoldIgnoresUnsupportedContentKeys(t *testing.T) {
	m := scaffoldModel(t)
	before := m.View().Content
	for _, code := range []rune{tea.KeyRight, tea.KeyLeft, tea.KeyEnter, tea.KeyPgDown, tea.KeyPgUp, tea.KeyHome, tea.KeyEnd} {
		if cmd := updateModel(t, m, tea.KeyPressMsg{Code: code}); cmd != nil {
			t.Fatalf("inactive key %d produced a command", code)
		}
		if m.View().Content != before || m.activePane != panel.LayerPane || m.screen != scaffoldScreen {
			t.Fatalf("inactive key %d changed scaffold", code)
		}
	}
	if m.viewport.GetContent() != "" || m.viewport.YOffset() != 0 || strings.Contains(ansi.Strip(before), "Scroll") {
		t.Fatal("scaffold entered the shared viewport or showed scroll hints")
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
		{2, 3, 20, "Layer selection"}, {23, 3, 20, "Category option"},
		{44, 3, 30, "Feature option"}, {75, 19, 43, "Controls/color picker"},
		{75, 3, 43, "Preview"},
	}
	for active := panel.LayerPane; active < panel.PaneCount; active++ {
		if m.activePane != active {
			t.Fatalf("focus = %d, want %d", m.activePane, active)
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
			if pane.heading == "Controls/color picker" {
				points = append(points, [2]int{pane.x, 35}, [2]int{pane.x + pane.width/2, 35}, [2]int{pane.x + pane.width - 1, 35})
			} else if pane.heading == "Preview" {
				points = append(points, [2]int{pane.x, 18}, [2]int{pane.x + pane.width - 1, 18})
			}
			for _, point := range points {
				if !sameColor(buffer.CellAt(point[0], point[1]).Style.Fg, paneColor) {
					t.Errorf("active %d: %s border at %v has wrong color", active, pane.heading, point)
				}
			}
			headingX := pane.x + 2
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
		if m.help.ShowAll || m.activePane != active || m.View().Content != before {
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
		if m.screen != loadingScreen || m.activePane != panel.LayerPane {
			t.Fatal("scaffold keys changed loading state")
		}
	}
	updateModel(t, m, spinner.TickMsg{Time: m.startedAt.Add(time.Second)})
	if m.elapsed != time.Second {
		t.Fatal("loading elapsed time no longer updates")
	}
	session := &testSession{}
	updateModel(t, m, preparedMsg{session: session, elapsed: 2 * time.Second})
	complete := ansi.Strip(m.View().Content)
	if m.screen != completionScreen || m.session != session || !strings.Contains(complete, "Processing complete") || !strings.Contains(complete, "Roads") || !strings.Contains(complete, "Continue") {
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
	updateModel(t, m, preparedMsg{err: failure})
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
	cmd = updateModel(t, m, preparedMsg{session: session})
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
			m := newModel(app.Options{}, func(app.Options, io.Writer) (loadedSession, error) {
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
	updateModel(t, m, preparedMsg{session: &testSession{layers: layers}})
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	return m
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
	if m.selectedLayer != 0 || len(m.layers) != len(layers) {
		t.Fatal("scaffold must initially select the first loaded layer")
	}
	for i, count := range []int{0, 0, 1, 12} {
		item := m.layers[i]
		if item.Name != layers[i].Name || item.FeatureCount != count || item.Complete {
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
	if !sameColor(buffer.CellAt(3, 5).Style.Bg, lipgloss.Color("236")) {
		t.Fatal("model did not pass the dark item background to the scaffold")
	}
}

func TestLayerSelectionAndScrolling(t *testing.T) {
	layers := make([]app.Layer, 10)
	for i := range layers {
		layers[i].Name = fmt.Sprintf("Layer %02d", i)
	}
	m := modelWithLayers(t, layers)
	updateModel(t, m, tea.WindowSizeMsg{Width: 120, Height: 24}) // Seven complete items.
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	if m.selectedLayer != 0 || m.firstVisibleLayer != 0 {
		t.Fatal("Up must clamp at the first layer")
	}
	for i := 1; i <= 12; i++ {
		cmd := updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
		want := min(i, len(layers)-1)
		if cmd != nil || m.selectedLayer != want || m.firstVisibleLayer != max(0, want-6) {
			t.Fatalf("Down %d: selected %d, first %d", i, m.selectedLayer, m.firstVisibleLayer)
		}
		view := ansi.Strip(m.View().Content)
		if !strings.Contains(view, layers[want].Name) || strings.Count(view, "○") != 7 || strings.Contains(view, "●") {
			t.Fatalf("Down %d: selected layer not visible or incorrect indicators", i)
		}
	}
	before := m.View().Content
	updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.View().Content != before || m.layers[9].Complete {
		t.Fatal("Enter must remain inactive on layer items")
	}
	for i := 8; i >= -2; i-- {
		updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
		want := max(0, i)
		if m.selectedLayer != want || !strings.Contains(ansi.Strip(m.View().Content), layers[want].Name) {
			t.Fatalf("Up: selected %d, want visible %d", m.selectedLayer, want)
		}
	}
}

func TestLayerSelectionPreserved(t *testing.T) {
	layers := make([]app.Layer, 10)
	for i := range layers {
		layers[i].Name = fmt.Sprintf("Layer %02d", i)
	}
	m := modelWithLayers(t, layers)
	updateModel(t, m, tea.WindowSizeMsg{Width: 120, Height: 24})
	for i := 0; i < 7; i++ {
		updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	for _, pane := range []panel.Pane{panel.CategoryPane, panel.FeaturePane, panel.ControlsPane} {
		updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt})
		for _, code := range []rune{tea.KeyDown, tea.KeyUp, tea.KeyEnter} {
			updateModel(t, m, tea.KeyPressMsg{Code: code})
			if m.activePane != pane || m.selectedLayer != 7 || m.firstVisibleLayer != 1 {
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
	if m.selectedLayer != 7 || m.firstVisibleLayer != 1 || m.View().Content != before {
		t.Fatal("help toggles changed selection or list window")
	}
	for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 12}, {Width: 1, Height: 1}, {Width: 180, Height: 40}, {Width: 120, Height: 24}} {
		updateModel(t, m, size)
		if m.selectedLayer != 7 || m.firstVisibleLayer != panel.LayerWindow(m.scaffoldOptions(m.layout())) {
			t.Fatal("resize lost selection or failed to recalculate window")
		}
		if size.Height >= 24 && !strings.Contains(ansi.Strip(m.View().Content), layers[7].Name) {
			t.Fatal("resize hid the selected layer")
		}
	}
	updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	updateModel(t, m, tea.WindowSizeMsg{Width: 120, Height: 12})
	updateModel(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	if m.selectedLayer != 7 || m.firstVisibleLayer != 7 {
		t.Fatal("returning from help must recalculate the shortened list window")
	}
}

func TestEmptyLayerSelection(t *testing.T) {
	for _, session := range []loadedSession{nil, &testSession{layers: []app.Layer{}}} {
		m := NewModel(app.Options{})
		updateModel(t, m, preparedMsg{session: session})
		updateModel(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
		updateModel(t, m, tea.WindowSizeMsg{Width: 120, Height: 24})
		for _, code := range []rune{tea.KeyUp, tea.KeyDown, tea.KeyEnter} {
			updateModel(t, m, tea.KeyPressMsg{Code: code})
		}
		if m.selectedLayer != 0 || m.firstVisibleLayer != 0 || len(m.layers) != 0 || !strings.Contains(ansi.Strip(m.View().Content), "No layers loaded") {
			t.Fatal("empty layer list did not stay stable")
		}
	}
}
