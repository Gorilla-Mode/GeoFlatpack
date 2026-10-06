package tui

import (
	"GeoFlatpack/fgb"
	"GeoFlatpack/internal/app"
	"GeoFlatpack/internal/tui/panel"
	"fmt"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
)

func layerItems(layers []app.Layer) []panel.ListItem {
	items := make([]panel.ListItem, len(layers))
	for i, layer := range layers {
		count := 0
		if layer.Data != nil {
			count = len(layer.Data.Features)
		}
		detail := fmt.Sprintf("%d features", count)
		if count == 1 {
			detail = "1 feature"
		}
		items[i] = panel.ListItem{Name: layer.Name, Detail: detail, Children: sourceGeometryCounts(layer.Data)}
	}
	return items
}

// Count source features, never coordinates, multipart children, or companions
// that the output workflow may later generate for SVG styling.
func sourceGeometryCounts(data *fgb.Fgb) []string {
	if data == nil {
		return nil
	}
	counts := make(map[string]int)
	fallback := flat.GeometryTypeUnknown
	if data.Header != nil && len(data.Header.Table().Bytes) > 0 {
		fallback = data.Header.GeometryType()
	}
	for i := range data.Features {
		label := "No geometry"
		raw := &data.Features[i].Raw
		if len(raw.Table().Bytes) > 0 {
			if geometry := raw.Geometry(&flat.Geometry{}); geometry != nil {
				typ := geometry.Type()
				if typ == flat.GeometryTypeUnknown {
					typ = fallback
				}
				switch typ {
				case flat.GeometryTypePoint, flat.GeometryTypeMultiPoint:
					label = "Point"
				case flat.GeometryTypeLineString, flat.GeometryTypeMultiLineString:
					label = "LineString"
				case flat.GeometryTypePolygon, flat.GeometryTypeMultiPolygon:
					label = "Polygon"
				default:
					label = "Other"
				}
			}
		}
		counts[label]++
	}
	var children []string
	for _, label := range []string{"Point", "LineString", "Polygon", "No geometry", "Other"} {
		if counts[label] > 0 {
			children = append(children, fmt.Sprintf("%s: %d", label, counts[label]))
		}
	}
	return children
}
