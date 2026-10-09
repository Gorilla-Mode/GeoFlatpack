package editor

import (
	"encoding/json"
	"maps"

	"GeoFlatpack/style/maplibre"
)

// Snapshot valid values independently of editor cursors and mutable arrays.
// Preview omits included options still awaiting their first valid value.
func (layer *styleLayerState) renderSnapshot(preview bool) (maplibre.RenderLayerStyle, error) {
	render := layer.style
	render.Paint = maps.Clone(render.Paint)
	render.Layout = maps.Clone(render.Layout)
	sections := make(map[string]maplibre.StyleSection, len(layer.properties))
	for _, p := range layer.properties {
		sections[p.name] = p.section
	}

	for name, draft := range layer.options {
		omit := !draft.included || (preview && !draft.hasValue)
		if sections[name] == maplibre.LayoutSection {
			if omit {
				delete(render.Layout, name)
				continue
			}

			if render.Layout == nil {
				render.Layout = make(map[string]any)
			}

			render.Layout[name] = draft.value
		} else {
			if omit {
				delete(render.Paint, name)
				continue
			}

			if render.Paint == nil {
				render.Paint = make(maplibre.Paint)
			}

			render.Paint[name] = draft.value
		}
	}

	raw, err := json.Marshal(render)
	if err != nil {
		return render, err
	}

	var snapshot maplibre.RenderLayerStyle
	err = json.Unmarshal(raw, &snapshot)

	return snapshot, err
}
