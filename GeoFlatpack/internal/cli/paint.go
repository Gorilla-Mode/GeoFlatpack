package cli

import (
	"GeoFlatpack/style/maplibre"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

func promptLine(
	scanner *bufio.Scanner,
	out io.Writer,
	prompt string,
) (string, error) {
	line, err := promptRawLine(scanner, out, prompt)
	return strings.TrimSpace(line), err
}

func promptRawLine(
	scanner *bufio.Scanner,
	out io.Writer,
	prompt string,
) (string, error) {
	if _, err := fmt.Fprint(out, prompt); err != nil {
		return "", err
	}

	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("input ended while waiting for an answer: %w", io.EOF)
}

func PromptPaints(
	groups []maplibre.StyleGroup,
	scanner *bufio.Scanner,
	out io.Writer,
) (map[maplibre.StyleGroup]maplibre.Paint, error) {
	spec, err := maplibre.LoadSpec()
	if err != nil {
		return nil, err
	}

	layerTypes := map[maplibre.GeometryType]string{
		maplibre.Point:   "circle",
		maplibre.Line:    "line",
		maplibre.Polygon: "fill",
	}

	paints := make(map[maplibre.StyleGroup]maplibre.Paint)

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

		paint := make(maplibre.Paint)

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
					value, err = maplibre.ParseProperty(input, property)
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
