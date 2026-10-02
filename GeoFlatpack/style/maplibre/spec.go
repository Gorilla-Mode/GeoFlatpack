package maplibre

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed spec/v8.json
var specJSON []byte

type PropertySpec struct {
	Type       PropertyType               `json:"type"`
	Doc        string                     `json:"doc"`
	Default    json.RawMessage            `json:"default"`
	Minimum    *float64                   `json:"minimum"`
	Maximum    *float64                   `json:"maximum"`
	Length     *int                       `json:"length"`
	Values     map[string]json.RawMessage `json:"values"`
	Transition bool                       `json:"transition"`
}

type Spec map[string]json.RawMessage

func LoadSpec() (Spec, error) {
	var spec Spec
	err := json.Unmarshal(specJSON, &spec)

	return spec, err
}

func (s Spec) Properties(layerType RenderType, section StyleSection) (map[string]PropertySpec, error) {
	key := string(section) + "_" + string(layerType)
	data, ok := s[key]

	if !ok {
		return nil, fmt.Errorf("unknown property section %q", key)
	}

	var properties map[string]PropertySpec
	if err := json.Unmarshal(data, &properties); err != nil {
		return nil, err
	}

	// Transition properties are described through a flag in the reference.
	if section == PaintSection {
		extra := make(map[string]PropertySpec)
		for name, property := range properties {
			addTransitionProperty(extra, name, property)
		}

		for name, property := range extra {
			properties[name] = property
		}
	}

	return properties, nil
}

func addTransitionProperty(properties map[string]PropertySpec, name string, property PropertySpec) {
	if !property.Transition {
		return
	}

	properties[name+"-transition"] = PropertySpec{
		Type: TransitionType,
		Doc:  `JSON object, for example {"duration":300,"delay":0}`,
	}
}
