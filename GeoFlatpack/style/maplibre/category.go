package maplibre

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
)

// CategoryValue is comparable so it can be used as part of a StyleGroup key.
// Its display label is separate from its typed MapLibre filter value.
type CategoryValue struct {
	kind  categoryKind
	value string
}

func NewCategoryValue(value any) (CategoryValue, error) {
	switch v := value.(type) {
	case nil:
		return CategoryValue{kind: categoryMissing}, nil
	case string:
		return CategoryValue{kind: categoryString, value: v}, nil
	case bool:
		return CategoryValue{kind: categoryBoolean, value: strconv.FormatBool(v)}, nil
	}

	return numericCategoryValue(value)
}

func numericCategoryValue(value any) (CategoryValue, error) {
	// FlatGeobuf exposes all signed/unsigned integer and floating point widths.
	v := reflect.ValueOf(value)
	var number string

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		number = strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		number = strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		f := v.Float()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return CategoryValue{}, fmt.Errorf("non-finite numbers cannot be styling categories")
		}

		if f == 0 { // Normalize negative zero.
			f = 0
		}

		number = strconv.FormatFloat(f, 'f', -1, v.Type().Bits())
	default:
		return CategoryValue{}, fmt.Errorf("complex or binary values (%T) cannot be styling categories", value)
	}

	return CategoryValue{kind: categoryNumber, value: number}, nil
}

func (c CategoryValue) String() string {
	switch c.kind {
	case categoryAll:
		return "All features"
	case categoryMissing:
		return "Missing value"
	case categoryString:
		return strconv.Quote(c.value)
	default:
		return c.value
	}
}

func (c CategoryValue) FilterValue() any {
	switch c.kind {
	case categoryString:
		return c.value
	case categoryNumber:
		return json.Number(c.value)
	case categoryBoolean:
		return c.value == "true"
	default:
		return nil
	}
}

func categoryLess(a, b CategoryValue) bool {
	if a.kind != b.kind {
		return a.kind < b.kind
	}
	return a.value < b.value
}
