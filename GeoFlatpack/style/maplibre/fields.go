package maplibre

import (
	"GeoFlatpack/fgb"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gogama/flatgeobuf/flatgeobuf"
	"github.com/gogama/flatgeobuf/flatgeobuf/flat"
)

type CategoryField struct {
	Name        string
	Types       []string
	Distinct    int
	Examples    []string
	Unavailable string
}

// DiscoverCategoryFields combines header/feature schemas with observed properties.
// Geometry is stored separately in FlatGeobuf and is never an attribute choice.
func DiscoverCategoryFields(data *fgb.Fgb) ([]CategoryField, error) {
	if data == nil || data.Header == nil {
		return nil, fmt.Errorf("missing FGB or header")
	}
	types := make(map[string]map[string]bool)
	unavailable := make(map[string]string)

	for i := 0; i < data.Header.ColumnsLength(); i++ {
		recordColumnType(data.Header, i, types, unavailable)
	}

	for i := range data.Features {
		for j := 0; j < data.Features[i].Raw.ColumnsLength(); j++ {
			recordColumnType(&data.Features[i].Raw, j, types, unavailable)
		}

		for name := range data.Features[i].Properties {
			recordFieldType(types, name, "")
		}
	}

	names := make([]string, 0, len(types))
	for name := range types {
		names = append(names, name)
	}

	sort.Strings(names)
	fields := make([]CategoryField, 0, len(names))

	for _, name := range names {
		field := CategoryField{Name: name, Unavailable: unavailable[name]}
		if name == "" {
			field.Unavailable = "empty attribute names cannot define styling categories"
		}

		seen := make(map[string]string)
		inferred := make(map[string]bool)

		for _, feature := range data.Features {
			observeCategoryValue(feature.Properties[name], &field, inferred, seen)
		}

		if len(types[name]) == 0 {
			types[name] = inferred
		}

		for typ := range types[name] {
			field.Types = append(field.Types, typ)
		}

		sort.Strings(field.Types)
		field.Distinct = len(seen)
		keys := make([]string, 0, len(seen))

		for key := range seen {
			keys = append(keys, key)
		}

		sort.Strings(keys)
		for _, key := range keys[:min(3, len(keys))] {
			field.Examples = append(field.Examples, formatCategoryExample(seen[key]))
		}

		fields = append(fields, field)
	}

	return fields, nil
}

func formatCategoryExample(value string) string {
	label := []rune(strings.ReplaceAll(value, "\n", `\n`))
	if len(label) > 60 {
		label = append(label[:57], '.', '.', '.')
	}

	return string(label)
}

func observeCategoryValue(value any, field *CategoryField, inferred map[string]bool, seen map[string]string) {
	category, err := NewCategoryValue(value)
	if err != nil {
		if field.Unavailable == "" {
			field.Unavailable = err.Error()
		}

		inferred[fmt.Sprintf("%T", value)] = true
		encoded, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			encoded = []byte(fmt.Sprint(value))
		}

		seen[fmt.Sprintf("%T:%s", value, encoded)] = string(encoded)

		return
	}

	inferred[category.kind] = true
	seen[category.kind+":"+category.value] = category.String()
}

func recordColumnType(schema flatgeobuf.Schema, index int, types map[string]map[string]bool, unavailable map[string]string) {
	var column flat.Column
	if !schema.Columns(&column, index) {
		return
	}

	name := string(column.Name())
	recordFieldType(types, name, column.Type().String())

	if column.Type() == flat.ColumnTypeJson || column.Type() == flat.ColumnTypeBinary {
		unavailable[name] = "complex or binary fields cannot define styling categories"
	}
}

func recordFieldType(types map[string]map[string]bool, name, typ string) {
	if types[name] == nil {
		types[name] = make(map[string]bool)
	}

	if typ != "" {
		types[name][typ] = true
	}
}
