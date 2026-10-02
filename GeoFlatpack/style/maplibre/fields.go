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

	addType := addType(types)
	addSchema := addSchema(addType, unavailable)

	addSchema(data.Header)
	for i := range data.Features {
		addSchema(&data.Features[i].Raw)
		for name := range data.Features[i].Properties {
			addType(name, "")
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

		decodeFeatures(data, name, field, inferred, seen)
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
		addExamples(keys, seen, field)

		fields = append(fields, field)
	}

	return fields, nil
}

func addExamples(keys []string, seen map[string]string, field CategoryField) {
	for _, key := range keys[:min(3, len(keys))] {
		label := []rune(strings.ReplaceAll(seen[key], "\n", `\n`))
		if len(label) > 60 {
			label = append(label[:57], '.', '.', '.')
		}
		field.Examples = append(field.Examples, string(label))
	}
}

func decodeFeatures(data *fgb.Fgb, name string, field CategoryField, inferred map[string]bool, seen map[string]string) {
	for _, feature := range data.Features {
		value := feature.Properties[name]
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

			continue
		}

		inferred[category.kind] = true
		seen[category.kind+":"+category.value] = category.String()
	}
}

func addSchema(addType func(name string, typ string), unavailable map[string]string) func(schema flatgeobuf.Schema) {
	return func(schema flatgeobuf.Schema) {
		for i := 0; i < schema.ColumnsLength(); i++ {
			var col flat.Column
			if !schema.Columns(&col, i) {
				continue
			}

			name := string(col.Name())
			addType(name, col.Type().String())
			if col.Type() == flat.ColumnTypeJson || col.Type() == flat.ColumnTypeBinary {
				unavailable[name] = "complex or binary fields cannot define styling categories"
			}
		}
	}
}

func addType(types map[string]map[string]bool) func(name string, typ string) {
	return func(name, typ string) {
		if types[name] == nil {
			types[name] = make(map[string]bool)
		}
		if typ != "" {
			types[name][typ] = true
		}
	}
}
