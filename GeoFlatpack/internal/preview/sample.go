package preview

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"

	"GeoFlatpack/style/maplibre"
)

const earthRadius = 6371008.8

// sampleGeometry replaces only the preview's WGS84 geometry. Properties, style
// filters and the export generator's vertex marker retain their original meaning.
func sampleGeometry(data collection, typ maplibre.GeometryType, marker string) (collection, error) {
	var original *feature
	for i := range data.Features {
		if _, companion := data.Features[i].Properties[marker]; marker != "" && companion {
			continue
		}
		original = &data.Features[i]
		break
	}
	if original == nil {
		return collection{}, fmt.Errorf("No geometry to preview")
	}
	centre, err := geometryCentre(original.Geometry)
	if err != nil {
		return collection{}, err
	}
	// Keep the sample inside the same latitude limits as the map camera.
	centre[1] = max(-85.05, min(85.05, centre[1]))
	position := func(east, north float64) [2]float64 {
		lat := centre[1] * math.Pi / 180
		return [2]float64{centre[0] + east/earthRadius/math.Cos(lat)*180/math.Pi, centre[1] + north/earthRadius*180/math.Pi}
	}
	var vertices [][2]float64
	var geometryType string
	var coordinates any
	switch typ {
	case maplibre.Point:
		geometryType, coordinates = "Point", centre
	case maplibre.Line:
		for i := 0; i < 5; i++ {
			north := float64(25)
			if i%2 == 1 {
				north = -25
			}
			vertices = append(vertices, position(-75+37.5*float64(i), north))
		}
		geometryType, coordinates = "LineString", vertices
	case maplibre.Polygon:
		for i := 0; i < 6; i++ {
			angle := float64(i) * math.Pi / 3
			vertices = append(vertices, position(50*math.Cos(angle), 50*math.Sin(angle)))
		}
		ring := append(append([][2]float64(nil), vertices...), vertices[0])
		geometryType, coordinates = "Polygon", [][][2]float64{ring}
	default:
		return collection{}, fmt.Errorf("Unsupported sample geometry: %s", typ)
	}
	geometry, err := marshalGeometry(geometryType, coordinates)
	if err != nil {
		return collection{}, err
	}
	sample := *original
	sample.Geometry = geometry
	sample.Properties = maps.Clone(original.Properties)
	result := collection{Type: "FeatureCollection", Features: []feature{sample}}
	if marker != "" && typ != maplibre.Point {
		geometry, err := marshalGeometry("MultiPoint", vertices)
		if err != nil {
			return collection{}, err
		}
		properties := maps.Clone(sample.Properties)
		if properties == nil {
			properties = make(map[string]any)
		}
		properties[marker] = string(typ)
		result.Features = append(result.Features, feature{Type: "Feature", Geometry: geometry, Properties: properties})
	}
	return result, nil
}

func marshalGeometry(typ string, coordinates any) (json.RawMessage, error) {
	return json.Marshal(struct {
		Type        string `json:"type"`
		Coordinates any    `json:"coordinates"`
	}{typ, coordinates})
}

func geometryCentre(raw json.RawMessage) ([2]float64, error) {
	var geometry struct {
		Coordinates any `json:"coordinates"`
	}
	if err := json.Unmarshal(raw, &geometry); err != nil {
		return [2]float64{}, err
	}
	bounds := [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	var visit func(any)
	visit = func(value any) {
		xy, ok := value.([]any)
		if !ok {
			return
		}
		if len(xy) >= 2 {
			x, xOK := xy[0].(float64)
			y, yOK := xy[1].(float64)
			if xOK && yOK && !math.IsNaN(x) && !math.IsNaN(y) && !math.IsInf(x, 0) && !math.IsInf(y, 0) {
				bounds[0], bounds[1] = min(bounds[0], x), min(bounds[1], y)
				bounds[2], bounds[3] = max(bounds[2], x), max(bounds[3], y)
				return
			}
		}
		for _, part := range xy {
			visit(part)
		}
	}
	visit(geometry.Coordinates)
	if math.IsInf(bounds[0], 1) {
		return [2]float64{}, fmt.Errorf("No geometry to preview")
	}
	return [2]float64{(bounds[0] + bounds[2]) / 2, (bounds[1] + bounds[3]) / 2}, nil
}
