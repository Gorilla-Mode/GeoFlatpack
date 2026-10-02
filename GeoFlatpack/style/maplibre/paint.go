package maplibre

import (
	"GeoFlatpack/style"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func ParseProperty(input string, property PropertySpec) (any, error) {
	input = strings.TrimSpace(input)

	// Advanced input: preserve JSON expressions or other complex values.
	// Complete validation happens against the finished stylesheet.
	if strings.HasPrefix(input, "json:") {
		var value any
		err := json.Unmarshal([]byte(strings.TrimSpace(input[5:])), &value)

		return value, err
	}

	switch property.Type {
	case ColorType:
		return style.ParseRGB(input)

	case StringType, ReferenceType:
		return input, nil

	case EnumType:
		if _, ok := property.Values[input]; !ok {
			return nil, fmt.Errorf("invalid choice %q", input)
		}

		return input, nil

	case BooleanType:
		return strconv.ParseBool(input)

	case NumberType:
		return parseNumberProperty(input, property)

	case ArrayType:
		return parseArrayProperty(input, property)

	default:
		// Other reference types, such as padding and transition objects.
		var value any
		err := json.Unmarshal([]byte(input), &value)

		return value, err
	}
}

func parseNumberProperty(input string, property PropertySpec) (any, error) {
	var value float64
	if err := json.Unmarshal([]byte(input), &value); err != nil {
		return nil, fmt.Errorf("enter a number")
	}

	if input == "null" {
		return nil, fmt.Errorf("enter a number")
	}

	if property.Minimum != nil && value < *property.Minimum {
		return nil, fmt.Errorf("minimum is %g", *property.Minimum)
	}

	if property.Maximum != nil && value > *property.Maximum {
		return nil, fmt.Errorf("maximum is %g", *property.Maximum)
	}

	return value, nil
}

func parseArrayProperty(input string, property PropertySpec) (any, error) {
	var values []any

	if err := json.Unmarshal([]byte(input), &values); err != nil || values == nil {
		return nil, fmt.Errorf("enter a JSON array, for example [2, 1]")
	}

	if property.Length != nil && len(values) != *property.Length {
		return nil, fmt.Errorf("expected %d elements", *property.Length)
	}

	return values, nil
}
