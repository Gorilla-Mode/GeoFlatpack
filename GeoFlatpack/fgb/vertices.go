package fgb

import (
	"encoding/binary"
	"fmt"
	"maps"

	"github.com/gogama/flatgeobuf/flatgeobuf"
	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
)

// WithVertexCompanions creates one MultiPoint per selected original feature.
// The selection maps feature indices to VertexLineString or VertexPolygon markers.
// The input is never modified. If all selected geometries are empty, the
// original dataset is returned along with the reserved marker name.
func WithVertexCompanions(data *Fgb, selected map[int]VertexKind) (*Fgb, string, error) {
	if data == nil || data.Header == nil {
		return nil, "", fmt.Errorf("missing FGB or header")
	}

	if len(selected) == 0 {
		return data, "", nil
	}

	names := make(map[string]bool)
	for i := 0; i < data.Header.ColumnsLength(); i++ {
		var column flat.Column
		data.Header.Columns(&column, i)
		names[string(column.Name())] = true
	}

	for i := range data.Features {
		for j := 0; j < data.Features[i].Raw.ColumnsLength(); j++ {
			var column flat.Column
			data.Features[i].Raw.Columns(&column, j)
			names[string(column.Name())] = true
		}

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
		kind, wanted := selected[i]
		g, vertices, err := prepareFeatureCompanion(&data.Features[i], data.Header.GeometryType(), kind, wanted, i)
		if err != nil {
			return nil, "", err
		}

		geometries[i] = g
		if vertices != nil {
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
		localSchema, markerIndex, err := resolveMarkerColumn(&feature.Raw, data.Header)
		if err != nil {
			return nil, "", fmt.Errorf("feature %d: %w", i+1, err)
		}

		payload := feature.Raw.PropertiesBytes()
		result.Features = append(result.Features, Feature{
			Raw:        packFeature(geometries[i], payload, localSchema, marker),
			Properties: feature.Properties,
		})

		if vertices := companions[i]; vertices != nil {
			result.Features = append(result.Features, buildCompanionFeature(feature, vertices, localSchema, markerIndex, marker, selected[i]))
		}
	}

	result.Header = packHeader(data.Header, marker, len(result.Features))

	return result, marker, nil
}

func prepareFeatureCompanion(feature *Feature, fallback flat.GeometryType, kind VertexKind, wanted bool, index int) (*geometry, *geometry, error) {
	g, err := readGeometry(feature.Raw.Geometry(&flat.Geometry{}), fallback)
	if err != nil {
		return nil, nil, fmt.Errorf("feature %d: %w", index+1, err)
	}

	if !wanted || g == nil {
		return g, nil, nil
	}

	if err := validateVertexSelection(g.typ, kind); err != nil {
		return nil, nil, fmt.Errorf("feature %d: %w", index+1, err)
	}

	vertices, err := g.vertices()
	if err != nil {
		return nil, nil, fmt.Errorf("feature %d vertices: %w", index+1, err)
	}

	if len(vertices.xy) == 0 {
		return g, nil, nil
	}

	return g, vertices, nil
}

func validateVertexSelection(typ flat.GeometryType, kind VertexKind) error {
	valid := kind == VertexLineString && (typ == flat.GeometryTypeLineString || typ == flat.GeometryTypeMultiLineString) ||
		kind == VertexPolygon && (typ == flat.GeometryTypePolygon || typ == flat.GeometryTypeMultiPolygon)
	if !valid {
		return fmt.Errorf("invalid vertex selection %q for %s", kind, typ)
	}

	return nil
}

func resolveMarkerColumn(feature *flat.Feature, header *flat.Header) (flatgeobuf.Schema, int, error) {
	var localSchema flatgeobuf.Schema
	markerIndex := header.ColumnsLength()
	if feature.ColumnsLength() > 0 {
		localSchema = feature
		markerIndex = feature.ColumnsLength()
	}

	if markerIndex >= 1<<16 {
		return nil, 0, fmt.Errorf("no column index available for vertex marker")
	}

	return localSchema, markerIndex, nil
}

func encodeMarkerPayload(payload []byte, markerIndex int, kind VertexKind) []byte {
	// Retain the original sparse property bytes exactly, including JSON,
	// binary, and integer payloads. Only the new marker entry is appended.
	properties := append([]byte(nil), payload...)
	properties = binary.LittleEndian.AppendUint16(properties, uint16(markerIndex))
	properties = binary.LittleEndian.AppendUint32(properties, uint32(len(kind)))
	properties = append(properties, string(kind)...)

	return properties
}

func buildCompanionFeature(feature Feature, vertices *geometry, schema flatgeobuf.Schema, markerIndex int, marker string, kind VertexKind) Feature {
	properties := encodeMarkerPayload(feature.Raw.PropertiesBytes(), markerIndex, kind)
	values := maps.Clone(feature.Properties)
	if values == nil {
		values = make(map[string]any)
	}

	values[marker] = string(kind)

	return Feature{
		Raw:        packFeature(vertices, properties, schema, marker),
		Properties: values,
	}
}
