package fgb

import (
	"encoding/binary"
	"fmt"
	"maps"

	"github.com/gogama/flatgeobuf/flatgeobuf"
	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
)

// WithVertexCompanions creates one MultiPoint per selected original feature.
// The selection maps feature indices to "LineString" or "Polygon" markers.
// The input is never modified. If all selected geometries are empty, the
// original dataset is returned along with the reserved marker name.
func WithVertexCompanions(data *Fgb, selected map[int]string) (*Fgb, string, error) {
	if data == nil || data.Header == nil {
		return nil, "", fmt.Errorf("missing FGB or header")
	}

	if len(selected) == 0 {
		return data, "", nil
	}

	names := make(map[string]bool)
	collectNames := func(schema flatgeobuf.Schema) {
		for i := 0; i < schema.ColumnsLength(); i++ {
			var c flat.Column
			schema.Columns(&c, i)
			names[string(c.Name())] = true
		}
	}

	collectNames(data.Header)
	for i := range data.Features {
		collectNames(&data.Features[i].Raw)
		for name := range data.Features[i].Properties {
			names[name] = true
		}
	}

	const base = "__geoflatpack_vertices"
	marker := base
	for suffix := 1; names[marker]; suffix++ {
		marker = fmt.Sprintf("%s_%d", base, suffix)
	}

	geometries := make([]*geometry, len(data.Features))
	companions := make(map[int]*geometry)
	for i := range data.Features {
		g, err := readGeometry(data.Features[i].Raw.Geometry(&flat.Geometry{}), data.Header.GeometryType())
		if err != nil {
			return nil, "", fmt.Errorf("feature %d: %w", i+1, err)
		}

		geometries[i] = g
		kind, wanted := selected[i]
		if !wanted || g == nil {
			continue
		}

		valid := kind == "LineString" && (g.typ == flat.GeometryTypeLineString || g.typ == flat.GeometryTypeMultiLineString) ||
			kind == "Polygon" && (g.typ == flat.GeometryTypePolygon || g.typ == flat.GeometryTypeMultiPolygon)
		if !valid {
			return nil, "", fmt.Errorf("feature %d: invalid vertex selection %q for %s", i+1, kind, g.typ)
		}

		vertices, err := g.vertices()
		if err != nil {
			return nil, "", fmt.Errorf("feature %d vertices: %w", i+1, err)
		}
		if len(vertices.xy) > 0 {
			companions[i] = vertices
		}
	}
	if len(companions) == 0 {
		return data, marker, nil
	}

	if data.Header.ColumnsLength() >= 1<<16 {
		return nil, "", fmt.Errorf("no column index available for vertex marker")
	}

	result := &Fgb{Features: make([]Feature, 0, len(data.Features)+len(companions))}
	for i, feature := range data.Features {
		var localSchema flatgeobuf.Schema
		markerIndex := data.Header.ColumnsLength()

		if feature.Raw.ColumnsLength() > 0 {
			localSchema = &feature.Raw
			markerIndex = feature.Raw.ColumnsLength()
		}

		if markerIndex >= 1<<16 {
			return nil, "", fmt.Errorf("feature %d: no column index available for vertex marker", i+1)
		}
		payload := feature.Raw.PropertiesBytes()
		result.Features = append(result.Features, Feature{
			Raw:        packFeature(geometries[i], payload, localSchema, marker),
			Properties: feature.Properties,
		})

		if vertices := companions[i]; vertices != nil {
			// Retain the original sparse property bytes exactly, including JSON,
			// binary, and integer payloads. Only the new marker entry is appended.
			properties := append([]byte(nil), payload...)
			properties = binary.LittleEndian.AppendUint16(properties, uint16(markerIndex))
			properties = binary.LittleEndian.AppendUint32(properties, uint32(len(selected[i])))
			properties = append(properties, selected[i]...)
			values := maps.Clone(feature.Properties)

			if values == nil {
				values = make(map[string]any)
			}

			values[marker] = selected[i]
			result.Features = append(result.Features, Feature{
				Raw: packFeature(vertices, properties, localSchema, marker), Properties: values,
			})
		}
	}
	result.Header = packHeader(data.Header, marker, len(result.Features))

	return result, marker, nil
}
