package tui

import (
	"encoding/json"
	"fmt"
	"maps"
	"sync"

	"GeoFlatpack/internal/app"
	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/style/maplibre"
	tea "charm.land/bubbletea/v2"
)

type writeDoneMsg struct {
	operation *writeOperation
	err       error
}

// Like preparation, an operation owns queued/running work until cleanup. The
// result remains available even if the program stops receiving messages.
type writeOperation struct {
	mu                sync.Mutex
	started, stopping bool
	done              chan struct{}
	err               error
}

func (w *writeOperation) command(session loadedSession, selections []app.StyleSelection) tea.Cmd {
	return func() tea.Msg {
		w.mu.Lock()
		if w.started || w.stopping {
			w.mu.Unlock()
			return nil
		}
		w.started = true
		w.mu.Unlock()
		defer close(w.done)
		w.err = session.Write(selections)
		return writeDoneMsg{operation: w, err: w.err}
	}
}

func (w *writeOperation) finish() error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	w.stopping = true
	started := w.started
	w.mu.Unlock()
	if !started {
		return nil
	}
	<-w.done
	return w.err
}

func (m *Model) layersReady() bool {
	if m.session == nil || len(m.layers) == 0 {
		return false
	}
	for _, layer := range m.layers {
		if layer.Status != panel.ListComplete {
			return false
		}
	}
	return true
}

func (m *Model) canWrite() bool {
	return m.screen == scaffoldScreen && !m.help.ShowAll && !m.writing && !m.controlsEditing() &&
		(m.Options.WriteFGB || m.Options.WriteStyle) && m.layersReady()
}

// Snapshot source/category/group order independently of visible cursors. Stack
// order is insertion order, which the shared writer translates to draw order.
func (m *Model) styleSelections() ([]app.StyleSelection, error) {
	if m.session == nil || len(m.layers) != len(m.categories) || len(m.categories) != len(m.session.Layers()) {
		return nil, fmt.Errorf("style selections unavailable")
	}
	selections := make([]app.StyleSelection, len(m.categories))
	for i, categories := range m.categories {
		if categories.active < 0 || categories.active >= len(categories.features) || categories.active >= len(categories.fields) {
			return nil, fmt.Errorf("layer %q has no active category", m.layers[i].Name)
		}
		features := categories.features[categories.active]
		if len(features.groups) != len(features.styling) {
			return nil, fmt.Errorf("layer %q has incomplete styling groups", m.layers[i].Name)
		}
		selection := app.StyleSelection{CategoryField: categories.fields[categories.active], Styles: make(map[maplibre.StyleGroup][]maplibre.RenderLayerStyle)}
		for j, group := range features.groups {
			state := features.styling[j]
			if state.status() != panel.ListComplete {
				return nil, fmt.Errorf("layer %q has incomplete styling", m.layers[i].Name)
			}
			stack := make([]maplibre.RenderLayerStyle, 0, len(state.layers))
			for _, layer := range state.layers {
				render := layer.style
				render.Paint = maps.Clone(render.Paint)
				render.Layout = maps.Clone(render.Layout)
				sections := make(map[string]maplibre.StyleSection, len(layer.properties))
				for _, p := range layer.properties {
					sections[p.name] = p.section
				}
				for name, draft := range layer.options {
					if sections[name] == maplibre.LayoutSection {
						if !draft.included {
							delete(render.Layout, name)
							continue
						}
						if render.Layout == nil {
							render.Layout = make(map[string]any)
						}
						render.Layout[name] = draft.value
					} else {
						if !draft.included {
							delete(render.Paint, name)
							continue
						}
						if render.Paint == nil {
							render.Paint = make(maplibre.Paint)
						}
						render.Paint[name] = draft.value
					}
				}
				stack = append(stack, render)
			}
			// JSON is the writer's value format. Round-trip the stack to copy all
			// nested arrays/objects, leaving the mutable editor maps untouched.
			data, err := json.Marshal(stack)
			if err != nil {
				return nil, fmt.Errorf("copy layer %q styling: %w", m.layers[i].Name, err)
			}
			var snapshot []maplibre.RenderLayerStyle
			if err := json.Unmarshal(data, &snapshot); err != nil {
				return nil, err
			}
			selection.Styles[group] = snapshot
		}
		selections[i] = selection
	}
	return selections, nil
}

func (m *Model) startWrite() tea.Cmd {
	m.refreshReadiness()
	var err error
	if !m.Options.WriteFGB && !m.Options.WriteStyle {
		err = fmt.Errorf("No outputs enabled")
	} else if !m.layersReady() {
		err = fmt.Errorf("Complete all layers before writing")
	}
	var selections []app.StyleSelection
	if err == nil {
		selections, err = m.styleSelections()
	}
	if err != nil {
		m.writeErr = err
		m.writeStatus = err.Error()
		return nil
	}
	m.writeOperation = &writeOperation{done: make(chan struct{})}
	m.writing = true
	m.writeErr = nil
	m.writeStatus = "Writing files…"
	return m.writeOperation.command(m.session, selections)
}
