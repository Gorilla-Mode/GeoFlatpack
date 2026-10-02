package fgb

import (
	"fmt"
	"slices"

	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
	flatbuffers "github.com/google/flatbuffers/go"
)

// geometry retains every coordinate channel supported by the FGB schema.
type geometry struct {
	typ         flat.GeometryType
	ends        []uint32
	xy, z, m, t []float64
	tm          []uint64
	parts       []*geometry
}

func readGeometry(raw *flat.Geometry, fallback flat.GeometryType) (*geometry, error) {
	if raw == nil {
		return nil, nil
	}

	g := &geometry{typ: raw.Type()}
	if g.typ == flat.GeometryTypeUnknown {
		g.typ = fallback
	}

	switch g.typ {
	case flat.GeometryTypePoint, flat.GeometryTypeMultiPoint, flat.GeometryTypeLineString,
		flat.GeometryTypeMultiLineString, flat.GeometryTypePolygon, flat.GeometryTypeMultiPolygon:
	default:
		return nil, fmt.Errorf("unsupported geometry %s", g.typ)
	}

	if raw.XyLength()%2 != 0 {
		return nil, fmt.Errorf("odd XY coordinate count")
	}

	for i := 0; i < raw.XyLength(); i++ {
		g.xy = append(g.xy, raw.Xy(i))
	}

	for i := 0; i < raw.ZLength(); i++ {
		g.z = append(g.z, raw.Z(i))
	}

	for i := 0; i < raw.MLength(); i++ {
		g.m = append(g.m, raw.M(i))
	}

	for i := 0; i < raw.TLength(); i++ {
		g.t = append(g.t, raw.T(i))
	}

	for i := 0; i < raw.TmLength(); i++ {
		g.tm = append(g.tm, raw.Tm(i))
	}

	n := len(g.xy) / 2
	for _, count := range []int{len(g.z), len(g.m), len(g.t), len(g.tm)} {
		if count != 0 && count != n {
			return nil, fmt.Errorf("coordinate dimensions have inconsistent lengths")
		}
	}

	var previous uint32
	for i := 0; i < raw.EndsLength(); i++ {
		end := raw.Ends(i)
		if end < previous || uint64(end) > uint64(n) {
			return nil, fmt.Errorf("invalid geometry end %d", end)
		}
		g.ends = append(g.ends, end)
		previous = end
	}

	if len(g.ends) > 0 && uint64(previous) != uint64(n) {
		return nil, fmt.Errorf("geometry ends do not cover coordinates")
	}

	childType := g.typ
	switch g.typ {
	case flat.GeometryTypeMultiPolygon:
		childType = flat.GeometryTypePolygon
	case flat.GeometryTypeMultiLineString:
		childType = flat.GeometryTypeLineString
	case flat.GeometryTypeMultiPoint:
		childType = flat.GeometryTypePoint
	}

	for i := 0; i < raw.PartsLength(); i++ {
		var part flat.Geometry
		raw.Parts(&part, i)
		child, err := readGeometry(&part, childType)
		if err != nil {
			return nil, err
		}
		g.parts = append(g.parts, child)
	}

	return g, nil
}

func (g *geometry) vertices() (*geometry, error) {
	vertices := &geometry{typ: flat.GeometryTypeMultiPoint}
	var visit func(*geometry) error

	visit = func(part *geometry) error {
		n := len(part.xy) / 2
		// A MultiPoint has a single set of dimension vectors. Reject inconsistent
		// multipart channels rather than losing dimensions or inventing values.
		if n > 0 && len(vertices.xy) > 0 {
			if (len(part.z) > 0) != (len(vertices.z) > 0) || (len(part.m) > 0) != (len(vertices.m) > 0) ||
				(len(part.t) > 0) != (len(vertices.t) > 0) || (len(part.tm) > 0) != (len(vertices.tm) > 0) {
				return fmt.Errorf("multipart coordinate dimensions differ")
			}
		}

		ends := part.ends
		if len(ends) == 0 {
			ends = []uint32{uint32(n)}
		}

		start := 0
		for _, end := range ends {
			stop := int(end)
			if part.typ == flat.GeometryTypePolygon && stop-start > 1 && part.samePosition(start, stop-1) {
				stop--
			}

			for i := start; i < stop; i++ {
				vertices.xy = append(vertices.xy, part.xy[2*i:2*i+2]...)

				if len(part.z) > 0 {
					vertices.z = append(vertices.z, part.z[i])
				}

				if len(part.m) > 0 {
					vertices.m = append(vertices.m, part.m[i])
				}

				if len(part.t) > 0 {
					vertices.t = append(vertices.t, part.t[i])
				}

				if len(part.tm) > 0 {
					vertices.tm = append(vertices.tm, part.tm[i])
				}
			}

			start = int(end)
		}

		for _, child := range part.parts {
			if err := visit(child); err != nil {
				return err
			}
		}

		return nil
	}

	if err := visit(g); err != nil {
		return nil, err
	}

	return vertices, nil
}

func (g *geometry) samePosition(a, b int) bool {
	return g.xy[2*a] == g.xy[2*b] && g.xy[2*a+1] == g.xy[2*b+1] &&
		(len(g.z) == 0 || g.z[a] == g.z[b]) && (len(g.m) == 0 || g.m[a] == g.m[b]) &&
		(len(g.t) == 0 || g.t[a] == g.t[b]) && (len(g.tm) == 0 || g.tm[a] == g.tm[b])
}

func floatVector(b *flatbuffers.Builder, n int, at func(int) float64) flatbuffers.UOffsetT {
	if n == 0 {
		return 0
	}

	b.StartVector(8, n, 8)
	for i := n - 1; i >= 0; i-- {
		b.PrependFloat64(at(i))
	}

	return b.EndVector(n)
}

func offsetsVector(b *flatbuffers.Builder, offsets []flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	if len(offsets) == 0 {
		return 0
	}

	b.StartVector(4, len(offsets), 4)

	for _, offset := range slices.Backward(offsets) {
		b.PrependUOffsetT(offset)
	}

	return b.EndVector(len(offsets))
}

func (g *geometry) pack(b *flatbuffers.Builder) flatbuffers.UOffsetT {
	if g == nil {
		return 0
	}

	parts := make([]flatbuffers.UOffsetT, len(g.parts))
	for i, part := range g.parts {
		parts[i] = part.pack(b)
	}

	partsOffset := offsetsVector(b, parts)
	var ends, tm flatbuffers.UOffsetT
	if len(g.ends) > 0 {
		flat.GeometryStartEndsVector(b, len(g.ends))
		for _, v := range slices.Backward(g.ends) {
			b.PrependUint32(v)
		}
		ends = b.EndVector(len(g.ends))
	}

	if len(g.tm) > 0 {
		flat.GeometryStartTmVector(b, len(g.tm))
		for _, v := range slices.Backward(g.tm) {
			b.PrependUint64(v)
		}
		tm = b.EndVector(len(g.tm))
	}

	vectors := make([]flatbuffers.UOffsetT, 4)
	for i, values := range [][]float64{g.xy, g.z, g.m, g.t} {
		vectors[i] = floatVector(b, len(values), func(j int) float64 { return values[j] })
	}

	flat.GeometryStart(b)
	flat.GeometryAddType(b, g.typ)
	flat.GeometryAddEnds(b, ends)
	flat.GeometryAddXy(b, vectors[0])
	flat.GeometryAddZ(b, vectors[1])
	flat.GeometryAddM(b, vectors[2])
	flat.GeometryAddT(b, vectors[3])
	flat.GeometryAddTm(b, tm)
	flat.GeometryAddParts(b, partsOffset)
	
	return flat.GeometryEnd(b)
}
