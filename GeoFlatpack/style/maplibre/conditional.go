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
	ID       string
	Operator string
	Value    any
	Paint    Paint
	Styles   map[GeometryType][]RenderLayerStyle
}

// ConditionalStyle replaces exact category grouping. An absent Otherwise
// geometry omits that fallback; rules remain available even without matches.
type ConditionalStyle struct {
	Field     string
	Rules     []ConditionalRule
	Otherwise map[GeometryType][]RenderLayerStyle
}

// DefinitionKey identifies matching semantics independently of style edits.
func (c *ConditionalStyle) DefinitionKey() string {
	if c == nil {
		return ""
	}

	rules := make([]ConditionalRule, len(c.Rules))
	for i, rule := range c.Rules {
		rules[i] = ConditionalRule{ID: rule.ID, Operator: rule.Operator, Value: rule.Value}
	}

	raw, _ := json.Marshal([]any{c.Field, rules})

	return string(raw)
}

// ValidateComparison uses the same typed comparison rules as generated filters
// and ConditionalPaint, without requiring a finished styling stack.
func ValidateComparison(field, operator string, value any) error {
	if strings.TrimSpace(field) == "" {
		return fmt.Errorf("empty field name")
	}

	_, err := conditionalComparison(field, operator, value)

	return err
}

func (c *ConditionalStyle) Validate() error {
	if strings.TrimSpace(c.Field) == "" {
		return fmt.Errorf("conditional style: empty field name")
	}

	seen := make(map[string]bool)
	for i, rule := range c.Rules {
		if rule.ID == "" || seen[rule.ID] {
			return fmt.Errorf("conditional rule %d: empty or duplicate ID %q", i+1, rule.ID)
		}

		seen[rule.ID] = true
		if err := ValidateComparison(c.Field, rule.Operator, rule.Value); err != nil {
			return fmt.Errorf("conditional rule %d: %w", i+1, err)
		}
	}

	return nil
}

// Match returns the first matching rule's ID, or the empty Otherwise ID.
// Convert numbers to double precision, as MapLibre's expression runtime does.
func (c *ConditionalStyle) Match(properties map[string]any) (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}

	actual, err := conditionalValue(properties[c.Field])
	if err != nil {
		return "", nil // Missing, null and incompatible values use Otherwise.
	}

	for _, rule := range c.Rules {
		expected, _ := conditionalValue(rule.Value)
		if actual.kind != expected.kind {
			continue
		}

		equal, less, greater := actual.value == expected.value, false, false
		if actual.kind == categoryNumber {
			a, _ := strconv.ParseFloat(actual.value, 64)
			b, _ := strconv.ParseFloat(expected.value, 64)
			equal, less, greater = a == b, a < b, a > b
		}

		matches := false
		switch rule.Operator {
		case "==":
			matches = equal
		case "!=":
			matches = !equal
		case ">":
			matches = greater
		case ">=":
			matches = greater || equal
		case "<":
			matches = less
		case "<=":
			matches = less || equal
		}
		if matches {
			return rule.ID, nil
		}
	}

	return "", nil
}

func (c *ConditionalStyle) Stack(group StyleGroup) []RenderLayerStyle {
	if group.RuleID == "" {
		return c.Otherwise[group.GeometryType]
	}

	for _, rule := range c.Rules {
		if rule.ID == group.RuleID {
			return rule.Styles[group.GeometryType]
		}
	}

	return nil
}

// Predicate includes exclusions for every preceding rule, including those
// whose styling uses a different render type or whose geometry has no matches.
func (c *ConditionalStyle) Predicate(id string) ([]any, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}

	filter := []any{"all"}
	for _, rule := range c.Rules {
		predicate, _ := conditionalComparison(c.Field, rule.Operator, rule.Value)
		if rule.ID == id {
			return append(filter, predicate), nil
		}
		filter = append(filter, []any{"!", predicate})
	}

	if id != "" {
		return nil, fmt.Errorf("unknown conditional rule %q", id)
	}

	return filter, nil
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
