package maplibre

import (
	"GeoFlatpack/style"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

type paintQuestion struct {
	Key, Label, Default string
	Color               bool
	Max                 float64 // Numeric maximum; 0 means no upper limit.
}

var paintQuestions = map[GeometryType][]paintQuestion{
	Point: {
		{"circle-color", "Point color (R, G, B)", "245, 158, 11", true, 0},
		{"circle-radius", "Radius in pixels", "6", false, 0},
		{"circle-stroke-color", "Outline color (R, G, B)", "120, 53, 15", true, 0},
		{"circle-stroke-width", "Outline width", "1", false, 0},
	},
	Line: {
		{"line-color", "Line color (R, G, B)", "220, 38, 38", true, 0},
		{"line-width", "Line width in pixels", "3", false, 0},
	},
	Polygon: {
		{"fill-color", "Fill color (R, G, B)", "37, 99, 235", true, 0},
		{"fill-opacity", "Fill opacity", "0.55", false, 1},
		{"fill-outline-color", "Outline color (R, G, B)", "30, 58, 138", true, 0},
	},
}

func parseProperty(input string, property PropertySpec) (any, error) {
	input = strings.TrimSpace(input)

	// Advanced input: preserve JSON expressions or other complex values.
	// Complete validation happens against the finished stylesheet.
	if strings.HasPrefix(input, "json:") {
		var value any
		err := json.Unmarshal([]byte(strings.TrimSpace(input[5:])), &value)
		return value, err
	}

	switch property.Type {
	case "color":
		return style.ParseRGB(input)

	case "string", "resolvedImage", "formatted":
		return input, nil

	case "enum":
		if _, ok := property.Values[input]; !ok {
			return nil, fmt.Errorf("invalid choice %q", input)
		}
		return input, nil

	case "boolean":
		return strconv.ParseBool(input)

	case "number":
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

	case "array":
		var values []any
		if err := json.Unmarshal([]byte(input), &values); err != nil || values == nil {
			return nil, fmt.Errorf("enter a JSON array, for example [2, 1]")
		}
		if property.Length != nil && len(values) != *property.Length {
			return nil, fmt.Errorf("expected %d elements", *property.Length)
		}
		return values, nil

	default:
		// Other reference types, such as padding and transition objects.
		var value any
		err := json.Unmarshal([]byte(input), &value)
		return value, err
	}
}

func PromptPaints(
	groups []StyleGroup,
	in io.Reader,
	out io.Writer,
) (map[StyleGroup]Paint, error) {
	spec, err := LoadSpec()
	if err != nil {
		return nil, err
	}

	layerTypes := map[GeometryType]string{
		Point:   "circle",
		Line:    "line",
		Polygon: "fill",
	}

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 4096), 1024*1024)

	paints := make(map[StyleGroup]Paint)

	for _, group := range groups {
		layerType, ok := layerTypes[group.GeometryType]
		if !ok {
			return nil, fmt.Errorf("unsupported geometry: %s", group.GeometryType)
		}

		properties, err := spec.Properties(layerType, "paint")
		if err != nil {
			return nil, err
		}

		names := make([]string, 0, len(properties))
		for name := range properties {
			names = append(names, name)
		}
		sort.Strings(names)

		if _, err := fmt.Fprintf(out, "\n%s (%s)\n%s\n",
			group.Category, group.GeometryType, strings.Join(names, "\n"),
		); err != nil {
			return nil, err
		}

		paint := make(Paint)

		for {
			name, err := promptLine(scanner, out, "\nProperty (Enter to finish): ")
			if err != nil {
				return nil, err
			}
			if name == "" {
				break
			}

			property, ok := properties[name]
			if !ok {
				if _, err := fmt.Fprintln(out, "Choose a property from the list."); err != nil {
					return nil, err
				}
				continue
			}

			if _, err := fmt.Fprintf(out, "%s\nType: %s\n",
				property.Doc, property.Type,
			); err != nil {
				return nil, err
			}

			if len(property.Values) > 0 {
				choices := make([]string, 0, len(property.Values))
				for choice := range property.Values {
					choices = append(choices, choice)
				}
				sort.Strings(choices)

				if _, err := fmt.Fprintln(out, "Choices:", strings.Join(choices, ", ")); err != nil {
					return nil, err
				}
			}

			prompt := "Value (Enter to skip): "
			if len(property.Default) > 0 {
				prompt = fmt.Sprintf("Value [%s] (Enter for default): ", property.Default)
			}

			for {
				input, err := promptLine(scanner, out, prompt)
				if err != nil {
					return nil, err
				}
				if input == "" && len(property.Default) == 0 {
					break
				}

				var value any
				if input == "" {
					// Defaults are already encoded as valid JSON values.
					err = json.Unmarshal(property.Default, &value)
				} else {
					value, err = parseProperty(input, property)
				}

				if err != nil {
					if _, writeErr := fmt.Fprintln(out, err); writeErr != nil {
						return nil, writeErr
					}
					continue
				}

				paint[name] = value
				break
			}
		}

		paints[group] = paint
	}

	return paints, nil
}

func promptLine(
	scanner *bufio.Scanner,
	out io.Writer,
	prompt string,
) (string, error) {
	if _, err := fmt.Fprint(out, prompt); err != nil {
		return "", err
	}

	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", io.EOF
}
