package fgb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gogama/flatgeobuf/flatgeobuf"
	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
	"github.com/gogama/flatgeobuf/packedrtree"
	flatbuffers "github.com/google/flatbuffers/go"
)

func optionalString(b *flatbuffers.Builder, value []byte) flatbuffers.UOffsetT {
	if value == nil {
		return 0
	}

	return b.CreateString(string(value))
}

func packColumn(b *flatbuffers.Builder, c *flat.Column) flatbuffers.UOffsetT {
	name := optionalString(b, c.Name())
	title := optionalString(b, c.Title())
	description := optionalString(b, c.Description())
	metadata := optionalString(b, c.Metadata())

	flat.ColumnStart(b)
	flat.ColumnAddName(b, name)
	flat.ColumnAddType(b, c.Type())
	flat.ColumnAddTitle(b, title)
	flat.ColumnAddDescription(b, description)
	flat.ColumnAddWidth(b, c.Width())
	flat.ColumnAddPrecision(b, c.Precision())
	flat.ColumnAddScale(b, c.Scale())
	flat.ColumnAddNullable(b, c.Nullable())
	flat.ColumnAddUnique(b, c.Unique())
	flat.ColumnAddPrimaryKey(b, c.PrimaryKey())
	flat.ColumnAddMetadata(b, metadata)

	return flat.ColumnEnd(b)
}

func packColumns(b *flatbuffers.Builder, schema flatgeobuf.Schema, marker string) flatbuffers.UOffsetT {
	columns := make([]flatbuffers.UOffsetT, 0, schema.ColumnsLength()+1)

	for i := 0; i < schema.ColumnsLength(); i++ {
		var column flat.Column
		schema.Columns(&column, i)
		columns = append(columns, packColumn(b, &column))
	}

	if marker != "" {
		name := b.CreateString(marker)
		flat.ColumnStart(b)
		flat.ColumnAddName(b, name)
		flat.ColumnAddType(b, flat.ColumnTypeString)
		flat.ColumnAddNullable(b, true)
		columns = append(columns, flat.ColumnEnd(b))
	}

	return offsetsVector(b, columns)
}

func packCRS(b *flatbuffers.Builder, c *flat.Crs) flatbuffers.UOffsetT {
	if c == nil {
		return 0
	}

	org := optionalString(b, c.Org())
	name := optionalString(b, c.Name())
	description := optionalString(b, c.Description())
	wkt := optionalString(b, c.Wkt())
	codeString := optionalString(b, c.CodeString())

	flat.CrsStart(b)
	flat.CrsAddOrg(b, org)
	flat.CrsAddCode(b, c.Code())
	flat.CrsAddName(b, name)
	flat.CrsAddDescription(b, description)
	flat.CrsAddWkt(b, wkt)
	flat.CrsAddCodeString(b, codeString)

	return flat.CrsEnd(b)
}

func packHeader(h *flat.Header, marker string, count int) *flat.Header {
	b := flatbuffers.NewBuilder(1024)
	name := optionalString(b, h.Name())
	title := optionalString(b, h.Title())
	description := optionalString(b, h.Description())
	metadata := optionalString(b, h.Metadata())
	envelope := floatVector(b, h.EnvelopeLength(), h.Envelope)
	columns := packColumns(b, h, marker)
	crs := packCRS(b, h.Crs(&flat.Crs{}))

	flat.HeaderStart(b)
	flat.HeaderAddName(b, name)
	flat.HeaderAddEnvelope(b, envelope)
	flat.HeaderAddGeometryType(b, flat.GeometryTypeUnknown)
	flat.HeaderAddHasZ(b, h.HasZ())
	flat.HeaderAddHasM(b, h.HasM())
	flat.HeaderAddHasT(b, h.HasT())
	flat.HeaderAddHasTm(b, h.HasTm())
	flat.HeaderAddColumns(b, columns)
	flat.HeaderAddFeaturesCount(b, uint64(count))

	nodeSize := h.IndexNodeSize()
	if nodeSize < 2 {
		nodeSize = 16 // The FGB schema's default index node size.
	}

	flat.HeaderAddIndexNodeSize(b, nodeSize)
	flat.HeaderAddCrs(b, crs)
	flat.HeaderAddTitle(b, title)
	flat.HeaderAddDescription(b, description)
	flat.HeaderAddMetadata(b, metadata)
	b.FinishSizePrefixed(flat.HeaderEnd(b))

	return flat.GetSizePrefixedRootAsHeader(b.FinishedBytes(), 0)
}

func packFeature(g *geometry, properties []byte, schema flatgeobuf.Schema, marker string) flat.Feature {
	b := flatbuffers.NewBuilder(1024)

	geometryOffset := g.pack(b)
	var props, columns flatbuffers.UOffsetT

	if len(properties) > 0 {
		props = b.CreateByteVector(properties)
	}

	if schema != nil {
		columns = packColumns(b, schema, marker)
	}

	flat.FeatureStart(b)
	flat.FeatureAddGeometry(b, geometryOffset)
	flat.FeatureAddProperties(b, props)
	flat.FeatureAddColumns(b, columns)
	b.FinishSizePrefixed(flat.FeatureEnd(b))

	return *flat.GetSizePrefixedRootAsFeature(b.FinishedBytes(), 0)
}

// WriteFgb writes an augmented dataset through a temporary sibling file. The
// original conversion path remains in convert.WriteFgb for unmodified datasets.
func WriteFgb(data *Fgb, output string) error {
	if data == nil || data.Header == nil || len(data.Features) == 0 || data.Header.IndexNodeSize() < 2 {
		return fmt.Errorf("augmented FGB requires a header, features, and an enabled spatial index")
	}

	// IndexData in the library only visits top-level XY coordinates. Recursing
	// here includes the parts of MultiPolygons when computing spatial bounds.
	refs := make([]packedrtree.Ref, len(data.Features))
	bounds := packedrtree.EmptyBox

	for i := range data.Features {
		refs[i] = featureIndexReference(&data.Features[i].Raw, i)
		bounds.Expand(&refs[i].Box)
	}

	packedrtree.HilbertSort(refs, bounds)
	features := make([]flat.Feature, len(refs))
	var offset int64

	for i := range refs {
		features[i] = data.Features[int(refs[i].Offset)].Raw
		offset = assignFeatureOffset(&refs[i], &features[i], offset)
	}

	index, err := packedrtree.New(refs, data.Header.IndexNodeSize())
	if err != nil {
		return err
	}

	dst, err := os.CreateTemp(filepath.Dir(output), ".gfp-*.fgb")
	if err != nil {
		return err
	}

	defer func() { _ = os.Remove(dst.Name()) }()
	writer := flatgeobuf.NewFileWriter(dst)
	_, err = writer.Header(data.Header)
	if err == nil {
		_, err = writer.Index(index)
	}
	if err == nil {
		_, err = writer.Data(features)
	}
	if err = errors.Join(err, writer.Close()); err != nil {
		return err
	}

	return os.Rename(dst.Name(), output)
}

func featureIndexReference(feature *flat.Feature, index int) packedrtree.Ref {
	box := packedrtree.EmptyBox
	expandBounds(&box, feature.Geometry(&flat.Geometry{}))
	if box == packedrtree.EmptyBox {
		// Null/empty geometries still need an index entry with finite bounds.
		box = packedrtree.Box{}
	}

	return packedrtree.Ref{Box: box, Offset: int64(index)}
}

func assignFeatureOffset(ref *packedrtree.Ref, feature *flat.Feature, offset int64) int64 {
	ref.Offset = offset

	return offset + int64(flatbuffers.GetUint32(feature.Table().Bytes)) + flatbuffers.SizeUint32
}

func expandBounds(box *packedrtree.Box, g *flat.Geometry) {
	if g == nil {
		return
	}

	for i := 0; i+1 < g.XyLength(); i += 2 {
		box.ExpandXY(g.Xy(i), g.Xy(i+1))
	}

	for i := 0; i < g.PartsLength(); i++ {
		var part flat.Geometry
		g.Parts(&part, i)
		expandBounds(box, &part)
	}
}
