package maplibre

import (
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"reflect"
	"strconv"
	"strings"
)

type ConditionalRule struct {
	Operator string
	Value    any
	Paint    Paint
}

func ConditionalPaint(field string, rules []ConditionalRule, fallback Paint) (Paint, error) {
	if strings.TrimSpace(field) == "" {
		return nil, fmt.Errorf("conditional paint: empty field name")
	}
	if _, err := json.Marshal(fallback); err != nil {
		return nil, fmt.Errorf("conditional paint fallback: %w", err)
	}

	conditions := make([]any, len(rules))
	changed := make(map[string]bool)

	for i, rule := range rules {
		condition, err := conditionalComparison(field, rule.Operator, rule.Value)
		if err != nil {
			return nil, fmt.Errorf("conditional paint rule %d: %w", i+1, err)
		}

		conditions[i] = condition
		if _, err := json.Marshal(rule.Paint); err != nil {
			return nil, fmt.Errorf("conditional paint rule %d: %w", i+1, err)
		}

		for property, value := range rule.Paint {
			base, ok := fallback[property]

			if !ok {
				return nil, fmt.Errorf("conditional paint rule %d: property %q has no fallback", i+1, property)
			}

			if !reflect.DeepEqual(value, base) {
				changed[property] = true
			}
		}
	}

	paint := make(Paint, len(fallback))
	maps.Copy(paint, fallback)

	for property := range changed {
		expression := make([]any, 1, 2*len(rules)+2)
		expression[0] = "case"

		for i, rule := range rules {
			value, ok := rule.Paint[property]
			if !ok {
				value = fallback[property]
			}

			expression = append(expression, conditions[i], conditionalLiteral(value))
		}

		paint[property] = append(expression, conditionalLiteral(fallback[property]))
	}

	return paint, nil
}

func conditionalComparison(field, operator string, value any) ([]any, error) {
	switch operator {
	case "==", "!=", ">", ">=", "<", "<=":
	default:
		return nil, fmt.Errorf("unsupported operator %q", operator)
	}

	category, err := conditionalValue(value)
	if err != nil {
		return nil, err
	}

	if operator != "==" && operator != "!=" && category.kind != categoryNumber {
		return nil, fmt.Errorf("operator %q requires a numeric comparison value", operator)
	}

	typ := string(category.kind)
	get := []any{"get", field}
	// Use a lazy case rather than relying on a failed type assertion, which would
	// make MapLibre fall back to the property's default instead of our fallback.
	return []any{
		"case",
		[]any{"==", []any{"typeof", get}, typ},
		[]any{operator, []any{typ, get}, category.FilterValue()},
		false,
	}, nil
}

func conditionalValue(value any) (CategoryValue, error) {
	if number, ok := value.(json.Number); ok {
		// json.Number is a string alias, so handle it before NewCategoryValue.
		if _, err := json.Marshal(number); err != nil {
			return CategoryValue{}, fmt.Errorf("invalid numeric comparison value: %w", err)
		}

		f, err := strconv.ParseFloat(string(number), 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return CategoryValue{}, fmt.Errorf("comparison value must be a finite number")
		}

		return CategoryValue{kind: categoryNumber, value: string(number)}, nil
	}
	category, err := NewCategoryValue(value)
	if err != nil {
		return CategoryValue{}, fmt.Errorf("invalid comparison value: %w", err)
	}
	if category.kind == categoryMissing {
		return CategoryValue{}, fmt.Errorf("comparison value must be a number, string, or boolean")
	}

	return category, nil
}

func conditionalLiteral(value any) any {
	// Inspect the JSON shape, since pointers and custom JSON types can also
	// serialize to arrays or objects. Serialization was validated above.
	raw, _ := json.Marshal(value)
	if len(raw) > 0 && (raw[0] == '[' || raw[0] == '{') {
		return []any{"literal", value}
	}

	return value
}
