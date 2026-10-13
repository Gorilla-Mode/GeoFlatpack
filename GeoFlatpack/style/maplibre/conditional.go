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
	Value    any // nil explicitly compares with Missing value (absent or null).
	Paint    Paint
	Styles   map[GeometryType][]RenderLayerStyle
}

// ConditionalStyle replaces exact category grouping. Use Field/Rules for a
// single field, or Fields/Combinations for a collection, never both. An absent
// Otherwise geometry omits the shared fallback; empty matches remain groups.
type ConditionalStyle struct {
	Field        string
	Rules        []ConditionalRule
	Fields       []ConditionalField
	Combinations []ConditionalCombination
	Otherwise    map[GeometryType][]RenderLayerStyle
}

// DefinitionKey identifies matching semantics independently of style edits.
func (c *ConditionalStyle) DefinitionKey() string {
	if c == nil {
		return ""
	}
	if len(c.Fields) > 0 {
		combinations := make([]ConditionalCombination, len(c.Combinations))
		for i, combination := range c.Combinations {
			combinations[i] = ConditionalCombination{ID: combination.ID, RuleIDs: combination.RuleIDs}
		}
		raw, _ := json.Marshal([]any{c.Fields, combinations})
		return string(raw)
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
	if len(c.Fields) > 0 {
		return c.validateCollection()
	}
	if len(c.Combinations) > 0 {
		return fmt.Errorf("conditional combinations require participating fields")
	}
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

// Match returns a rule or combination ID, or the empty shared Otherwise ID.
// Convert numbers to double precision, as MapLibre's expression runtime does.
func (c *ConditionalStyle) Match(properties map[string]any) (string, error) {
	match, err := c.Matcher()
	if err != nil {
		return "", err
	}
	return match(properties), nil
}

func (c *ConditionalStyle) Stack(group StyleGroup) []RenderLayerStyle {
	if group.RuleID == "" {
		return c.Otherwise[group.GeometryType]
	}
	if len(c.Fields) > 0 {
		for _, combination := range c.Combinations {
			if combination.ID == group.RuleID {
				return combination.Styles[group.GeometryType]
			}
		}
		return nil
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
	if len(c.Fields) > 0 {
		return c.collectionPredicate(id)
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
	if category.kind == categoryMissing {
		return []any{operator, get, nil}, nil
	}
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
