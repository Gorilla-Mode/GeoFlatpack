package convert

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/airbusgeo/godal"
	"github.com/gogama/flatgeobuf/flatgeobuf"
	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
)

type normalizedReader struct {
	io.Reader
	io.Closer
}

// OpenReader normalizes the list column metadata without copying the spatial
// index or features. GDAL writes String(JSON) payloads but may label them String.
func (fgb *MemoryFGB) OpenReader() (io.ReadCloser, error) {
	src, err := godal.VSIOpen(fgb.path)
	if err != nil {
		return nil, err
	}
	if len(fgb.ListColumns) == 0 {
		return src, nil
	}
	return normalizeListColumns(src, fgb.ListColumns)
}

func normalizeListColumns(src io.ReadCloser, names []string) (result io.ReadCloser, err error) {
	defer func() {
		// FlatBuffers accessors panic on malformed table offsets.
		if p := recover(); p != nil {
			result = nil
			err = fmt.Errorf("normalize FGB header: %v", p)
		}
		if err != nil {
			err = errors.Join(err, src.Close())
		}
	}()
	magic := make([]byte, 8)
	if _, err := io.ReadFull(src, magic); err != nil {
		return nil, fmt.Errorf("read FGB magic: %w", err)
	}
	// Header reads exactly the magic, length prefix and header table, leaving
	// src at the unchanged index. The library validates magic and header size.
	header, err := flatgeobuf.NewFileReader(io.MultiReader(bytes.NewReader(magic), src)).Header()
	if err != nil {
		return nil, fmt.Errorf("read FGB header: %w", err)
	}
	wanted := make(map[string]bool, len(names))
	for _, name := range names {
		wanted[name] = true
	}
	for i := 0; i < header.ColumnsLength(); i++ {
		var col flat.Column
		if !header.Columns(&col, i) || !wanted[string(col.Name())] {
			continue
		}
		if col.Type() != flat.ColumnTypeString && col.Type() != flat.ColumnTypeJson {
			return nil, fmt.Errorf("list column %q has unexpected type %s", col.Name(), col.Type())
		}
		if !col.MutateType(flat.ColumnTypeJson) {
			return nil, fmt.Errorf("cannot normalize list column %q", col.Name())
		}
		delete(wanted, string(col.Name()))
	}
	for _, name := range names {
		if wanted[name] {
			return nil, fmt.Errorf("list column %q missing from FGB header", name)
		}
	}
	return &normalizedReader{
		Reader: io.MultiReader(bytes.NewReader(magic), bytes.NewReader(header.Table().Bytes), src),
		Closer: src,
	}, nil
}
