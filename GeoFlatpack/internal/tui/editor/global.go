package editor

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"GeoFlatpack/fgb"
	"GeoFlatpack/internal/preview"
	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/style/maplibre"
)

// Global is an editor choice. Export lowers it to ordinary, mutually exclusive
// collection buckets so the styling engine needs no special override semantics.
type globalConditionRef struct {
	field  int
	ruleID string
}

type conditionGroupDraft struct {
	style stylingState
	item  panel.ListItem
}

func (s *categoryState) globalCondition(fields []int) *globalConditionRef {
	for _, field := range fields {
		for _, rule := range s.conditional[field].rules {
			if rule.global {
				return &globalConditionRef{field: field, ruleID: rule.id}
			}
		}
	}
	return nil
}

func (s *categoryState) globalID(ref globalConditionRef) string {
	raw, _ := json.Marshal([]string{s.fields[ref.field], ref.ruleID})
	return "global:" + string(raw)
}

func (s *categoryState) saveGlobalDraft() {
	if s.collection == nil || s.collection.global == nil {
		return
	}
	if s.globalDrafts == nil {
		s.globalDrafts = make(map[maplibre.StyleGroup]conditionGroupDraft)
	}
	id := s.globalID(*s.collection.global)
	f := s.collection.features
	for i, group := range f.groups {
		if group.RuleID == id {
			s.globalDrafts[group] = conditionGroupDraft{style: f.styling[i], item: f.items[i]}
		}
	}
}

func (m *Editor) toggleGlobalCondition(index int) {
	s, c := m.currentCategories(), m.conditionEditor
	s.saveGlobalDraft()
	global := !c.rules[index].global
	for field, conditions := range s.conditional {
		changed := false
		for i := range conditions.rules {
			if conditions.rules[i].global {
				conditions.rules[i].global = false
				changed = true
			}
		}
		if changed {
			conditions.rebuild(s.fields[field])
		}
	}
	c.rules[index].global = global
	m.rebuildConditions()
}

// A global-only field is a guard, rather than an extra product dimension.
func (s *categoryState) normalFields(collection *collectionState) []int {
	var fields []int
	for _, field := range collection.fields {
		if field == collection.global.field && len(s.conditional[field].rules) == 1 {
			continue
		}
		fields = append(fields, field)
	}
	return fields
}

func (s *categoryState) normalGroupID(fields []int, tuple map[string]string) string {
	if len(fields) == 1 {
		return tuple[s.fields[fields[0]]]
	}
	return maplibre.CombinationID(tuple)
}

// Reuse complete draft objects, including dormant form inputs, when a logical
// group survives a change between normal and global collection views.
func shareSurvivingGroups(target, previous *featureState) {
	for i, group := range target.groups {
		if index := slices.Index(previous.groups, group); index >= 0 {
			target.styling[i] = previous.styling[index]
			target.items[i].Status, target.items[i].Omitted = previous.items[index].Status, previous.items[index].Omitted
			if previous.selected == index {
				target.selected = i
			}
			if previous.chosen == index {
				target.chosen = i
			}
		}
	}
	target.firstVisible = min(previous.firstVisible, target.selected)
}

func (s *categoryState) restoreDemotedGlobal(target *collectionState, ref globalConditionRef) {
	if !slices.Contains(target.fields, ref.field) {
		return
	}
	ids := make(map[string]bool)
	if len(target.fields) == 1 && target.global == nil {
		ids[ref.ruleID] = true
	}
	for _, combination := range target.combinations {
		if combination.RuleIDs[s.fields[ref.field]] == ref.ruleID {
			ids[combination.ID] = true
		}
	}
	for i, group := range target.features.groups {
		if !ids[group.RuleID] || len(target.features.styling[i].layers) != 0 {
			continue
		}
		original := maplibre.StyleGroup{GeometryType: group.GeometryType, RuleID: s.globalID(ref)}
		if saved, ok := s.globalDrafts[original]; ok {
			style := saved.style
			if len(target.fields) > 1 {
				style = cloneStylingDraft(style)
			}
			target.features.styling[i] = style
			target.features.items[i].Status = saved.item.Status
		}
	}
}

func (s *categoryState) rebuildGlobalCollection(target, previous *collectionState) {
	old := *target.features
	next := featureState{chosen: -1, selected: old.selected, firstVisible: old.firstVisible}
	fields := s.normalFields(target)
	var baseline *featureState
	if saved := s.collections[s.normalCollectionKey(fields)]; saved != nil {
		baseline = saved.features
	} else if len(fields) == 1 {
		baseline = &s.conditional[fields[0]].features
	}
	appendGroup := func(group maplibre.StyleGroup, labels []string) {
		item := panel.ListItem{Name: strings.Join(labels, " AND "), Children: append(slices.Clone(labels), "Type: "+string(group.GeometryType))}
		style := stylingState{active: -1}
		if index := slices.Index(old.groups, group); index >= 0 {
			style, item.Status, item.Omitted = old.styling[index], old.items[index].Status, old.items[index].Omitted
			if index == old.selected {
				next.selected = len(next.items)
			}
			if index == old.chosen {
				next.chosen = len(next.items)
			}
		} else if group.RuleID == s.globalID(*target.global) {
			if saved, ok := s.globalDrafts[group]; ok {
				style, item.Status = saved.style, saved.item.Status
			} else {
				individual := &s.conditional[target.global.field].features
				original := maplibre.StyleGroup{GeometryType: group.GeometryType, RuleID: target.global.ruleID}
				if index := slices.Index(individual.groups, original); index >= 0 {
					style = individual.styling[index]
					item.Status = individual.items[index].Status
				}
			}
		} else if baseline != nil {
			if index := slices.Index(baseline.groups, group); index >= 0 {
				style = cloneStylingDraft(baseline.styling[index])
				item.Status, item.Omitted = baseline.items[index].Status, baseline.items[index].Omitted
			}
		}
		// Global drafts follow their condition across collection memberships.
		if saved, ok := s.globalDrafts[group]; ok && group.RuleID == s.globalID(*target.global) {
			style, item.Status = saved.style, saved.item.Status
		}
		next.groups = append(next.groups, group)
		next.items = append(next.items, item)
		next.styling = append(next.styling, style)
	}
	ref := target.global
	for _, rule := range s.conditional[ref.field].rules {
		if rule.id == ref.ruleID {
			for _, geometry := range s.features[0].groups {
				appendGroup(maplibre.StyleGroup{GeometryType: geometry.GeometryType, RuleID: s.globalID(*ref)}, []string{s.conditional[ref.field].label(s.fields[ref.field], rule)})
			}
		}
	}
	var tuples []map[string]string
	if len(fields) > 0 {
		tuples = []map[string]string{{}}
	}
	for _, field := range fields {
		var product []map[string]string
		for _, tuple := range tuples {
			for _, rule := range s.conditional[field].rules {
				if rule.global {
					continue
				}
				child := maps.Clone(tuple)
				child[s.fields[field]] = rule.id
				product = append(product, child)
			}
		}
		tuples = product
	}
	target.combinations = nil
	for _, tuple := range tuples {
		id := s.normalGroupID(fields, tuple)
		target.combinations = append(target.combinations, maplibre.ConditionalCombination{ID: id, RuleIDs: tuple})
		var labels []string
		for _, field := range fields {
			for _, rule := range s.conditional[field].rules {
				if rule.id == tuple[s.fields[field]] {
					labels = append(labels, s.conditional[field].label(s.fields[field], rule))
				}
			}
		}
		for _, geometry := range s.features[0].groups {
			appendGroup(maplibre.StyleGroup{GeometryType: geometry.GeometryType, RuleID: id}, labels)
		}
	}
	for _, geometry := range s.features[0].groups {
		appendGroup(geometry, []string{"Otherwise"})
	}
	if previous != nil {
		if slices.Equal(target.fields, previous.fields) {
			shareSurvivingGroups(&next, previous.features)
		} else {
			// Match existing parent groups when adding a global-only field.
			for i, group := range next.groups {
				if slices.Index(old.groups, group) < 0 {
					if index := slices.Index(previous.features.groups, group); index >= 0 {
						next.styling[i] = cloneStylingDraft(previous.features.styling[index])
						next.items[i].Status, next.items[i].Omitted = previous.features.items[index].Status, previous.features.items[index].Omitted
					}
				}
			}
		}
	}
	next.selected = panel.MoveListSelection(next.selected, 0, len(next.items))
	next.firstVisible = min(next.firstVisible, next.selected)
	*target.features = next
}

type compiledGlobalCollection struct {
	style      *maplibre.ConditionalStyle
	logicalIDs map[string]string // compiled tuple ID -> visible group ID
}

const (
	globalMissingBucket = "global-fallback:missing"
	globalPresentBucket = "global-fallback:present"
)

func (s *categoryState) compileGlobalCollection(previewMode bool) (compiledGlobalCollection, error) {
	collection := s.collection
	ref := collection.global
	result := compiledGlobalCollection{style: &maplibre.ConditionalStyle{}, logicalIDs: make(map[string]string)}
	// Explicit final buckets make every field exhaustive, allowing the global
	// style to win even if another field is missing, complex, or unmatched.
	for _, field := range collection.fields {
		definition := maplibre.ConditionalField{Field: s.fields[field]}
		var global *maplibre.ConditionalComparison
		for _, rule := range s.conditional[field].rules {
			value, err := s.conditional[field].ruleValue(definition.Field, rule)
			if err != nil {
				return result, fmt.Errorf("field %q condition %s: %w", definition.Field, rule.id, err)
			}
			comparison := maplibre.ConditionalComparison{ID: rule.id, Operator: rule.operator, Value: value}
			if rule.global {
				global = &comparison
			} else {
				definition.Rules = append(definition.Rules, comparison)
			}
		}
		if global != nil {
			definition.Rules = append([]maplibre.ConditionalComparison{*global}, definition.Rules...)
		}
		definition.Rules = append(definition.Rules,
			maplibre.ConditionalComparison{ID: globalMissingBucket, Operator: "==", Value: nil},
			maplibre.ConditionalComparison{ID: globalPresentBucket, Operator: "!=", Value: nil})
		result.style.Fields = append(result.style.Fields, definition)
	}
	stacks := make(map[maplibre.StyleGroup][]maplibre.RenderLayerStyle)
	features := collection.features
	for i, group := range features.groups {
		if features.items[i].Omitted {
			continue
		}
		stack, err := snapshotStack(features.styling[i], group.GeometryType, previewMode)
		if err != nil {
			return result, fmt.Errorf("condition %s: %w", features.items[i].Name, err)
		}
		stacks[group] = stack
	}
	fields := s.normalFields(collection)
	var expand func(int, map[string]string)
	expand = func(index int, tuple map[string]string) {
		if index < len(result.style.Fields) {
			field := result.style.Fields[index]
			for _, rule := range field.Rules {
				tuple[field.Field] = rule.ID
				expand(index+1, tuple)
			}
			return
		}
		logicalID := ""
		if tuple[s.fields[ref.field]] == ref.ruleID {
			logicalID = s.globalID(*ref)
		} else if len(fields) > 0 {
			normal := make(map[string]string)
			for _, field := range fields {
				id := tuple[s.fields[field]]
				if id == globalMissingBucket || id == globalPresentBucket {
					normal = nil
					break
				}
				normal[s.fields[field]] = id
			}
			if normal != nil {
				logicalID = s.normalGroupID(fields, normal)
			}
		}
		id := maplibre.CombinationID(tuple)
		combination := maplibre.ConditionalCombination{ID: id, RuleIDs: maps.Clone(tuple), Styles: make(map[maplibre.GeometryType][]maplibre.RenderLayerStyle)}
		for _, geometry := range s.features[0].groups {
			group := maplibre.StyleGroup{GeometryType: geometry.GeometryType, RuleID: logicalID}
			stack := stacks[group]
			if len(stack) == 0 {
				// The existing engine requires a stack for every explicit tuple.
				// Hide compiled Otherwise aliases when the editor omits Otherwise.
				stack = []maplibre.RenderLayerStyle{{Type: maplibre.RenderTypes[geometry.GeometryType][0], Layout: map[string]any{"visibility": "none"}}}
			}
			combination.Styles[geometry.GeometryType] = stack
		}
		result.style.Combinations = append(result.style.Combinations, combination)
		result.logicalIDs[id] = logicalID
	}
	expand(0, make(map[string]string))
	return result, result.style.Validate()
}

type globalPreviewChoice struct {
	data       *fgb.Fgb
	definition string
	logical    maplibre.StyleGroup
	resolved   maplibre.StyleGroup
	err        error
}

func (s *categoryState) resolveGlobalPreview(data *fgb.Fgb, compiled compiledGlobalCollection, logical maplibre.StyleGroup) (maplibre.StyleGroup, error) {
	definition := compiled.style.DefinitionKey()
	if cached := s.globalPreview; cached != nil && cached.data == data && cached.definition == definition && cached.logical == logical {
		return cached.resolved, cached.err
	}
	choice := &globalPreviewChoice{data: data, definition: definition, logical: logical, err: fmt.Errorf("No matching geometry to preview")}
	first := int(^uint(0) >> 1)
	for _, combination := range compiled.style.Combinations {
		if compiled.logicalIDs[combination.ID] != logical.RuleID {
			continue
		}
		group := maplibre.StyleGroup{GeometryType: logical.GeometryType, RuleID: combination.ID}
		if index, err := preview.ConditionalRepresentativeIndex(data, compiled.style, group); err == nil && index < first {
			first, choice.resolved, choice.err = index, group, nil
		}
	}
	s.globalPreview = choice
	return choice.resolved, choice.err
}
