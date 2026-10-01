package fgb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gogama/flatgeobuf/flatgeobuf"
	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
)

// Properties are sparse index/value pairs, not a row with one value per column.
// Unset and null fields have no entry in the buffer or the resulting map.
func readProperties(data []byte, schema flatgeobuf.Schema) (map[string]any, error) {
	values := make(map[string]any)
	buf := bytes.NewReader(data)
	r := flatgeobuf.NewPropReader(buf)

	for buf.Len() > 0 {
		offset := len(data) - buf.Len()
		index, err := r.ReadUShort()

		if err != nil {
			return nil, fmt.Errorf("column index at byte %d: %w", offset, err)
		}

		var col flat.Column
		if int(index) >= schema.ColumnsLength() || !schema.Columns(&col, int(index)) {
			return nil, fmt.Errorf("column index %d out of range (%d columns)", index, schema.ColumnsLength())
		}

		value, err := readProperty(r, buf, col.Type())
		if err != nil {
			return nil, fmt.Errorf("column %d %q (%s): %w", index, col.Name(), col.Type(), err)
		}

		values[string(col.Name())] = value
	}
	return values, nil
}

func readProperty(r *flatgeobuf.PropReader, buf *bytes.Reader, typ flat.ColumnType) (any, error) {
	switch typ {
	case flat.ColumnTypeByte:
		return r.ReadByte()
	case flat.ColumnTypeUByte:
		return r.ReadUByte()
	case flat.ColumnTypeBool:
		return r.ReadBool()
	case flat.ColumnTypeShort:
		return r.ReadShort()
	case flat.ColumnTypeUShort:
		return r.ReadUShort()
	case flat.ColumnTypeInt:
		return r.ReadInt()
	case flat.ColumnTypeUInt:
		return r.ReadUInt()
	case flat.ColumnTypeLong:
		return r.ReadLong()
	case flat.ColumnTypeULong:
		return r.ReadULong()
	case flat.ColumnTypeFloat:
		return r.ReadFloat()
	case flat.ColumnTypeDouble:
		return r.ReadDouble()
	case flat.ColumnTypeString, flat.ColumnTypeDateTime, flat.ColumnTypeJson, flat.ColumnTypeBinary:
		size, err := r.ReadUInt()
		if err != nil {
			return nil, fmt.Errorf("read length: %w", err)
		}

		if uint64(size) > uint64(buf.Len()) {
			return nil, fmt.Errorf("length %d exceeds remaining %d bytes: %w", size, buf.Len(), io.ErrUnexpectedEOF)
		}

		data := make([]byte, int(size))
		if _, err := io.ReadFull(buf, data); err != nil {
			return nil, err
		}

		switch typ {
		case flat.ColumnTypeBinary:
			return data, nil
		case flat.ColumnTypeJson:
			decoder := json.NewDecoder(bytes.NewReader(data))
			decoder.UseNumber()

			var value any
			if err := decoder.Decode(&value); err != nil {
				return nil, fmt.Errorf("invalid JSON: %w", err)
			}

			var extra any
			if err := decoder.Decode(&extra); err != io.EOF {
				if err == nil {
					err = fmt.Errorf("multiple JSON values")
				}
				return nil, fmt.Errorf("invalid JSON: %w", err)
			}

			return value, nil
		default:
			return string(data), nil
		}
	default:
		return nil, fmt.Errorf("unknown column type %d", typ)
	}
}
