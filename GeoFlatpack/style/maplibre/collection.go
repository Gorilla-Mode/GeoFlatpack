package maplibre

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ConditionalComparison describes a bucket in one participating field.
// Ordering is significant: each field independently uses its first match.
type ConditionalComparison struct {
	ID       string
	Operator string
	Value    any // nil explicitly compares with Missing value (absent or null).
}

type ConditionalField struct {
	Field string
	Rules []ConditionalComparison
}

// ConditionalCombination owns a stack for a tuple of explicit field buckets.
// Per-field Otherwise buckets are excluded; the collection has one fallback.
type ConditionalCombination struct {
	ID      string
	RuleIDs map[string]string
	Styles  map[GeometryType][]RenderLayerStyle
}

// CombinationID encodes a tuple without delimiter collisions. Map keys are
// sorted by encoding/json, so field ordering never changes its identity.
func CombinationID(ruleIDs map[string]string) string {
	raw, _ := json.Marshal(ruleIDs)
	return string(raw)
}

func (c *ConditionalStyle) validateCollection() error {
	if c.Field != "" || len(c.Rules) > 0 {
		return fmt.Errorf("cannot mix single-field and collection conditional definitions")
	}
	fields := make(map[string]map[string]bool, len(c.Fields))
	empty := false
	for _, field := range c.Fields {
		if strings.TrimSpace(field.Field) == "" {
			return fmt.Errorf("conditional collection: empty field name")
		}
		if _, exists := fields[field.Field]; exists {
			return fmt.Errorf("duplicate conditional field %q", field.Field)
		}
		rules := make(map[string]bool, len(field.Rules))
		for i, rule := range field.Rules {
			if rule.ID == "" || rules[rule.ID] {
				return fmt.Errorf("field %q rule %d: empty or duplicate ID %q", field.Field, i+1, rule.ID)
			}
			rules[rule.ID] = true
			if err := ValidateComparison(field.Field, rule.Operator, rule.Value); err != nil {
				return fmt.Errorf("field %q rule %d: %w", field.Field, i+1, err)
			}
		}
		fields[field.Field] = rules
		empty = empty || len(rules) == 0
	}
	expected := 0
	if !empty {
		expected = 1
		for _, field := range c.Fields {
			if expected > int(^uint(0)>>1)/len(field.Rules) {
				return fmt.Errorf("conditional combination count exceeds addressable size")
			}
			expected *= len(field.Rules)
		}
	}
	ids, tuples := make(map[string]bool), make(map[string]bool)
	for _, combination := range c.Combinations {
		if combination.ID == "" || ids[combination.ID] {
			return fmt.Errorf("empty or duplicate combination ID %q", combination.ID)
		}
		ids[combination.ID] = true
		if len(combination.RuleIDs) != len(fields) {
			return fmt.Errorf("combination %q must reference every participating field", combination.ID)
		}
		for name, rules := range fields {
			if !rules[combination.RuleIDs[name]] {
				return fmt.Errorf("combination %q references an unknown rule in field %q", combination.ID, name)
			}
		}
		key := CombinationID(combination.RuleIDs)
		if tuples[key] {
			return fmt.Errorf("duplicate conditional combination %s", key)
		}
		tuples[key] = true
	}
	if len(c.Combinations) != expected {
		return fmt.Errorf("conditional collection has %d combinations, expected %d", len(c.Combinations), expected)
	}
	return nil
}

func comparisonMatches(actual, expected CategoryValue, operator string) bool {
	if expected.kind == categoryMissing {
		if operator == "==" {
			return actual.kind == categoryMissing
		}
		return operator == "!=" && actual.kind != categoryMissing
	}
	if actual.kind != expected.kind {
		return false
	}
	equal, less, greater := actual.value == expected.value, false, false
	if actual.kind == categoryNumber {
		a, _ := strconv.ParseFloat(actual.value, 64)
		b, _ := strconv.ParseFloat(expected.value, 64)
		equal, less, greater = a == b, a < b, a > b
	}
	switch operator {
	case "==":
		return equal
	case "!=":
		return !equal
	case ">":
		return greater
	case ">=":
		return greater || equal
	case "<":
		return less
	case "<=":
		return less || equal
	}
	return false
}

// Matcher validates once and snapshots matching definitions for a source scan.
// Geometry scans do not repeatedly validate the entire Cartesian product.
func (c *ConditionalStyle) Matcher() (func(map[string]any) string, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	type compiledRule struct {
		id, operator string
		expected     CategoryValue
	}
	type compiledField struct {
		name  string
		rules []compiledRule
	}
	var fields []compiledField
	compile := func(name string, rules []ConditionalComparison) {
		field := compiledField{name: name}
		for _, rule := range rules {
			expected, _ := conditionalValue(rule.Value)
			field.rules = append(field.rules, compiledRule{id: rule.ID, operator: rule.Operator, expected: expected})
		}
		fields = append(fields, field)
	}
	collection := len(c.Fields) > 0
	if collection {
		for _, field := range c.Fields {
			compile(field.Field, field.Rules)
		}
	} else {
		rules := make([]ConditionalComparison, len(c.Rules))
		for i, rule := range c.Rules {
			rules[i] = ConditionalComparison{ID: rule.ID, Operator: rule.Operator, Value: rule.Value}
		}
		compile(c.Field, rules)
	}
	combinations := make(map[string]string, len(c.Combinations))
	for _, combination := range c.Combinations {
		tuple := make([]string, len(fields))
		for i, field := range fields {
			tuple[i] = combination.RuleIDs[field.name]
		}
		combinations[conditionalTupleKey(tuple)] = combination.ID
	}
	return func(properties map[string]any) string {
		tuple := make([]string, len(fields))
		for i, field := range fields {
			actual, err := conditionalValue(properties[field.name])
			for _, rule := range field.rules {
				// Complex values cannot match typed scalar comparisons, but are
				// still present values for != Missing value.
				if err != nil && rule.expected.kind != categoryMissing {
					continue
				}
				if comparisonMatches(actual, rule.expected, rule.operator) {
					tuple[i] = rule.id
					break
				}
			}
			if tuple[i] == "" {
				return ""
			}
		}
		if !collection {
			return tuple[0]
		}
		return combinations[conditionalTupleKey(tuple)]
	}, nil
}

func conditionalTupleKey(tuple []string) string {
	raw, _ := json.Marshal(tuple)
	return string(raw)
}

func (c *ConditionalStyle) collectionPredicate(id string) ([]any, error) {
	if id == "" {
		everyFieldMatches := []any{"all"}
		for _, field := range c.Fields {
			anyRule := []any{"any"}
			for _, rule := range field.Rules {
				predicate, _ := conditionalComparison(field.Field, rule.Operator, rule.Value)
				anyRule = append(anyRule, predicate)
			}
			everyFieldMatches = append(everyFieldMatches, anyRule)
		}
		return []any{"!", everyFieldMatches}, nil
	}
	for _, combination := range c.Combinations {
		if combination.ID != id {
			continue
		}
		filter := []any{"all"}
		for _, field := range c.Fields {
			for _, rule := range field.Rules {
				predicate, _ := conditionalComparison(field.Field, rule.Operator, rule.Value)
				if rule.ID == combination.RuleIDs[field.Field] {
					filter = append(filter, predicate)
					break
				}
				filter = append(filter, []any{"!", predicate})
			}
		}
		return filter, nil
	}
	return nil, fmt.Errorf("unknown conditional combination %q", id)
}

func (c *ConditionalStyle) targetIDs() []string {
	var ids []string
	if len(c.Fields) > 0 {
		for _, combination := range c.Combinations {
			ids = append(ids, combination.ID)
		}
	} else {
		for _, rule := range c.Rules {
			ids = append(ids, rule.ID)
		}
	}
	return ids
}
