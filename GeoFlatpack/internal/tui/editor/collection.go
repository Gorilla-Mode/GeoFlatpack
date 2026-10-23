package editor

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"sort"
	"strings"

	"GeoFlatpack/internal/tui/panel"
	"GeoFlatpack/style/maplibre"
)

// Collections are retained by membership; one-field collections share the
// existing field draft. Multi-field collections own independent combinations.
type collectionState struct {
	fields       []int
	features     *featureState
	combinations []maplibre.ConditionalCombination
	global       *globalConditionRef
}

func (s *categoryState) markedFields() []int {
	var fields []int
	for i, item := range s.items {
		if item.Conditional {
			fields = append(fields, i)
		}
	}
	sort.Slice(fields, func(i, j int) bool { return s.fields[fields[i]] < s.fields[fields[j]] })
	return fields
}

func (s *categoryState) collectionKey(fields []int) string {
	key := s.normalCollectionKey(fields)
	if global := s.globalCondition(fields); global != nil {
		raw, _ := json.Marshal([]any{key, s.fields[global.field], global.ruleID})
		return string(raw)
	}
	return key
}

func (s *categoryState) normalCollectionKey(fields []int) string {
	names := make([]string, len(fields))
	for i, index := range fields {
		names[i] = s.fields[index]
	}
	raw, _ := json.Marshal(names)
	return string(raw)
}

func (s *categoryState) toggleConditional() {
	if s.fields[s.selected] == "" {
		return
	}
	s.items[s.selected].Conditional = !s.items[s.selected].Conditional
	s.conditional[s.selected].rebuild(s.fields[s.selected])
	if s.items[s.selected].Conditional {
		s.active, s.collectionActive = s.selected, true
	}
	if s.collectionActive {
		s.rebuildCollection(s.selected)
	}
	s.move(0)
}

func (s *categoryState) refreshActiveIndicators() {
	for i := range s.items {
		s.items[i].Status = panel.ListUnopened
		if (s.collectionActive && s.items[i].Conditional) || (!s.collectionActive && i == s.active) {
			s.items[i].Status = panel.ListActive
		}
	}
}

// Rebuild only replaces the visible product. Existing stacks are kept by tuple
// identity; extending a collection seeds new children from its previous parent.
func (s *categoryState) rebuildCollection(seedField int) {
	s.saveGlobalDraft()
	fields := s.markedFields()
	if len(fields) == 0 {
		s.collectionActive, s.active = false, -1
		s.refreshActiveIndicators()
		return
	}
	if s.collections == nil {
		s.collections = make(map[string]*collectionState)
	}
	key := s.collectionKey(fields)
	target := s.collections[key]
	previous := s.collection
	if target == nil {
		target = &collectionState{fields: slices.Clone(fields), features: &featureState{chosen: -1}}
		target.global = s.globalCondition(fields)
		if len(fields) == 1 && target.global == nil {
			s.conditional[fields[0]].rebuild(s.fields[fields[0]])
			target.features = &s.conditional[fields[0]].features
		}
		s.collections[key] = target
	}
	if target.global != nil {
		s.rebuildGlobalCollection(target, previous)
	} else if len(fields) > 1 {
		var seed *collectionState
		if previous != nil && len(previous.fields) < len(fields) && fieldSubset(previous.fields, fields) {
			seed = previous
		} else if previous == nil {
			if !slices.Contains(fields, seedField) {
				seedField = fields[0]
			}
			s.conditional[seedField].rebuild(s.fields[seedField])
			seed = &collectionState{fields: []int{seedField}, features: &s.conditional[seedField].features}
		}
		s.rebuildProduct(target, seed)
	}
	if target.global == nil && previous != nil && previous.global != nil && slices.Equal(fields, previous.fields) {
		shareSurvivingGroups(target.features, previous.features)
	}
	if previous != nil && previous.global != nil && (target.global == nil || *target.global != *previous.global) {
		s.restoreDemotedGlobal(target, *previous.global)
	}
	s.collection = target
	if s.collectionActive {
		if !s.items[s.active].Conditional {
			s.active = fields[0]
		}
		s.refreshActiveIndicators()
	}
}

func fieldSubset(subset, fields []int) bool {
	for _, field := range subset {
		if !slices.Contains(fields, field) {
			return false
		}
	}
	return true
}

func (s *categoryState) productDefinitions(fields []int) []maplibre.ConditionalCombination {
	tuples := []map[string]string{{}}
	for _, field := range fields {
		var next []map[string]string
		for _, tuple := range tuples {
			for _, rule := range s.conditional[field].rules {
				child := maps.Clone(tuple)
				child[s.fields[field]] = rule.id
				next = append(next, child)
			}
		}
		tuples = next
	}
	result := make([]maplibre.ConditionalCombination, len(tuples))
	for i, tuple := range tuples {
		result[i] = maplibre.ConditionalCombination{ID: maplibre.CombinationID(tuple), RuleIDs: tuple}
	}
	return result
}

func (s *categoryState) rebuildProduct(target, seed *collectionState) {
	old := *target.features
	drafts := make(map[maplibre.StyleGroup]int, len(old.groups))
	for i, group := range old.groups {
		drafts[group] = i
	}
	seeds := make(map[maplibre.StyleGroup]int)
	if seed != nil {
		for i, group := range seed.features.groups {
			seeds[group] = i
		}
	}
	next := featureState{chosen: -1}
	selectedRestored := false
	appendGroup := func(group maplibre.StyleGroup, tuple map[string]string, conditions []string) {
		name := strings.Join(conditions, " AND ")
		if group.RuleID == "" {
			name = "Otherwise"
			conditions = []string{"Otherwise"}
		}
		children := make([]string, 0, len(conditions)+1)
		for _, condition := range conditions {
			children = append(children, condition)
		}
		children = append(children, "Type: "+string(group.GeometryType))
		item := panel.ListItem{Name: name, Children: children}
		style := stylingState{active: -1}
		if index, exists := drafts[group]; exists {
			style = old.styling[index]
			item.Status, item.Omitted = old.items[index].Status, old.items[index].Omitted
			if index == old.selected {
				next.selected = len(next.items)
				selectedRestored = true
			}
			if index == old.chosen {
				next.chosen = len(next.items)
			}
		} else if seed != nil {
			parent := maplibre.StyleGroup{GeometryType: group.GeometryType}
			if group.RuleID != "" {
				if len(seed.fields) == 1 {
					parent.RuleID = tuple[s.fields[seed.fields[0]]]
				} else {
					projection := make(map[string]string, len(seed.fields))
					for _, field := range seed.fields {
						projection[s.fields[field]] = tuple[s.fields[field]]
					}
					parent.RuleID = maplibre.CombinationID(projection)
				}
			}
			if index, exists := seeds[parent]; exists {
				style = cloneStylingDraft(seed.features.styling[index])
				item.Status, item.Omitted = seed.features.items[index].Status, seed.features.items[index].Omitted
				// Keep the target on the first child of the previous selection.
				if index == seed.features.selected && !selectedRestored {
					next.selected = len(next.items)
					selectedRestored = true
				}
				if index == seed.features.chosen && next.chosen < 0 {
					next.chosen = len(next.items)
				}
			}
		}
		next.groups = append(next.groups, group)
		next.items = append(next.items, item)
		next.styling = append(next.styling, style)
	}
	target.combinations = s.productDefinitions(target.fields)
	for _, combination := range target.combinations {
		var labels []string
		for _, field := range target.fields {
			for _, rule := range s.conditional[field].rules {
				if rule.id == combination.RuleIDs[s.fields[field]] {
					labels = append(labels, s.conditional[field].label(s.fields[field], rule))
					break
				}
			}
		}
		for _, geometry := range s.features[0].groups {
			appendGroup(maplibre.StyleGroup{GeometryType: geometry.GeometryType, RuleID: combination.ID}, combination.RuleIDs, labels)
		}
	}
	for _, geometry := range s.features[0].groups {
		appendGroup(geometry, nil, nil)
	}
	next.selected = panel.MoveListSelection(next.selected, 0, len(next.items))
	next.firstVisible = min(old.firstVisible, next.selected)
	*target.features = next
}

// Changes to an inactive field are retained without activating its collection.
// Pruning every saved membership also removes drafts owned by deleted rules.
func (m *Editor) rebuildConditions() {
	m.conditionEditor.rebuild(m.conditionField)
	s := m.currentCategories()
	for _, collection := range s.collections {
		if len(collection.fields) > 1 && collection.global == nil {
			s.pruneCollection(collection)
		}
	}
	if s.collectionActive {
		s.rebuildCollection(s.active)
	}
}

// Hidden memberships keep only drafts that actually existed. New tuples are
// initialized when that membership becomes active, so extension can inherit
// from its parent rather than finding an artificial empty cached draft.
func (s *categoryState) pruneCollection(collection *collectionState) {
	valid := make(map[string]bool)
	var combinations []maplibre.ConditionalCombination
	for _, combination := range collection.combinations {
		keep := true
		for _, field := range collection.fields {
			found := false
			for _, rule := range s.conditional[field].rules {
				found = found || rule.id == combination.RuleIDs[s.fields[field]]
			}
			keep = keep && found
		}
		if keep {
			valid[combination.ID] = true
			combinations = append(combinations, combination)
		}
	}
	if len(combinations) == len(collection.combinations) {
		return
	}
	old := *collection.features
	next := featureState{chosen: -1}
	for i, group := range old.groups {
		if group.RuleID != "" && !valid[group.RuleID] {
			continue
		}
		if i == old.selected {
			next.selected = len(next.items)
		}
		if i == old.chosen {
			next.chosen = len(next.items)
		}
		next.groups = append(next.groups, group)
		next.items = append(next.items, old.items[i])
		next.styling = append(next.styling, old.styling[i])
	}
	next.firstVisible = min(old.firstVisible, next.selected)
	collection.combinations = combinations
	*collection.features = next
}

func (s *categoryState) collectionSnapshot(preview bool) (*maplibre.ConditionalStyle, error) {
	collection := s.collection
	if collection == nil {
		return nil, fmt.Errorf("no active conditional collection")
	}
	if collection.global != nil {
		compiled, err := s.compileGlobalCollection(preview)
		if err != nil {
			return nil, err
		}
		return compiled.style, nil
	}
	if len(collection.fields) == 1 {
		field := collection.fields[0]
		return s.conditional[field].snapshot(s.fields[field], preview)
	}
	result := &maplibre.ConditionalStyle{Otherwise: make(map[maplibre.GeometryType][]maplibre.RenderLayerStyle)}
	for _, field := range collection.fields {
		definition := maplibre.ConditionalField{Field: s.fields[field]}
		for _, rule := range s.conditional[field].rules {
			value, err := s.conditional[field].ruleValue(definition.Field, rule)
			if err != nil {
				return nil, fmt.Errorf("field %q condition %s: %w", definition.Field, rule.id, err)
			}
			definition.Rules = append(definition.Rules, maplibre.ConditionalComparison{ID: rule.id, Operator: rule.operator, Value: value})
		}
		result.Fields = append(result.Fields, definition)
	}
	indices := make(map[string]int)
	for _, combination := range collection.combinations {
		indices[combination.ID] = len(result.Combinations)
		result.Combinations = append(result.Combinations, maplibre.ConditionalCombination{ID: combination.ID, RuleIDs: maps.Clone(combination.RuleIDs), Styles: make(map[maplibre.GeometryType][]maplibre.RenderLayerStyle)})
	}
	features := collection.features
	for i, group := range features.groups {
		if features.items[i].Omitted {
			continue
		}
		stack, err := snapshotStack(features.styling[i], group.GeometryType, preview)
		if err != nil {
			return nil, fmt.Errorf("combination %s: %w", features.items[i].Name, err)
		}
		if group.RuleID == "" {
			result.Otherwise[group.GeometryType] = stack
		} else {
			result.Combinations[indices[group.RuleID]].Styles[group.GeometryType] = stack
		}
	}
	return result, nil
}

func snapshotStack(state stylingState, geometry maplibre.GeometryType, preview bool) ([]maplibre.RenderLayerStyle, error) {
	if !preview && state.status() != panel.ListComplete {
		return nil, fmt.Errorf("incomplete styling")
	}
	var stack []maplibre.RenderLayerStyle
	for _, layer := range state.layers {
		if preview && layer.style.Type == maplibre.RenderSymbol && layer.style.IconName == "" {
			continue
		}
		render, err := layer.renderSnapshot(preview)
		if err != nil {
			return nil, err
		}
		stack = append(stack, render)
	}
	if preview && len(stack) == 0 {
		stack = []maplibre.RenderLayerStyle{{Type: maplibre.RenderTypes[geometry][0]}}
	}
	return stack, nil
}

// Clone domain drafts, including excluded and unfinished options. Cached forms
// belong to the parent, so children recreate them from their own copied values.
func cloneStylingDraft(original stylingState) stylingState {
	result := original
	result.layers = make([]*styleLayerState, len(original.layers))
	for i, layer := range original.layers {
		copy := *layer
		copy.style.Paint = cloneDraftValue(layer.style.Paint).(maplibre.Paint)
		copy.style.Layout = cloneDraftValue(layer.style.Layout).(map[string]any)
		copy.options = make(map[string]optionDraft, len(layer.options))
		for name, draft := range layer.options {
			draft.value = cloneDraftValue(draft.value)
			copy.options[name] = draft
		}
		copy.controls = nil
		result.layers[i] = &copy
	}
	return result
}

func cloneDraftValue(value any) any {
	if value == nil {
		return nil
	}
	return cloneDraftReflect(reflect.ValueOf(value)).Interface()
}

func cloneDraftReflect(value reflect.Value) reflect.Value {
	switch value.Kind() {
	case reflect.Interface, reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		if value.Kind() == reflect.Pointer {
			copy := reflect.New(value.Type().Elem())
			copy.Elem().Set(cloneDraftReflect(value.Elem()))
			return copy
		}
		copy := reflect.New(value.Type()).Elem()
		copy.Set(cloneDraftReflect(value.Elem()))
		return copy
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copy := reflect.MakeMapWithSize(value.Type(), value.Len())
		iter := value.MapRange()
		for iter.Next() {
			copy.SetMapIndex(iter.Key(), cloneDraftReflect(iter.Value()))
		}
		return copy
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copy := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := 0; i < value.Len(); i++ {
			copy.Index(i).Set(cloneDraftReflect(value.Index(i)))
		}
		return copy
	case reflect.Array:
		copy := reflect.New(value.Type()).Elem()
		for i := 0; i < value.Len(); i++ {
			copy.Index(i).Set(cloneDraftReflect(value.Index(i)))
		}
		return copy
	case reflect.Struct:
		copy := reflect.New(value.Type()).Elem()
		copy.Set(value)
		for i := 0; i < value.NumField(); i++ {
			if copy.Field(i).CanSet() && value.Field(i).CanInterface() {
				copy.Field(i).Set(cloneDraftReflect(value.Field(i)))
			}
		}
		return copy
	default:
		return value
	}
}
