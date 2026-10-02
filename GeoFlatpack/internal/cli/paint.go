package cli

import (
	"GeoFlatpack/style/maplibre"
	"GeoFlatpack/style/maplibre/svg"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
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

type StyleOptions struct {
	Icons    map[string]svg.Svg
	WriteFGB bool
}

func PromptStyles(
	groups []maplibre.StyleGroup,
	scanner *bufio.Scanner,
	out io.Writer,
	options ...StyleOptions,
) (map[maplibre.StyleGroup][]maplibre.RenderLayerStyle, error) {
	var opts StyleOptions
	if len(options) > 0 {
		opts = options[0]
	}

	spec, err := maplibre.LoadSpec()
	if err != nil {
		return nil, err
	}

	styles := make(map[maplibre.StyleGroup][]maplibre.RenderLayerStyle)
	if len(opts.Icons) > 0 {
		if _, err := fmt.Fprintln(out, "SVG icon layers draw above the other layers, in icon selection order."); err != nil {
			return nil, err
		}
	}
	for _, group := range groups {
		types := append([]string(nil), maplibre.RenderTypes[group.GeometryType]...)
		if len(types) == 0 {
			return nil, fmt.Errorf("unsupported geometry: %s", group.GeometryType)
		}
		
		if len(opts.Icons) > 0 {
			if group.GeometryType == maplibre.Point || opts.WriteFGB {
				types = append(types, "SVG icon")
			} else if _, err := fmt.Fprintln(out, "SVG icons for lines and polygons require writing companion geometry; enable --write-fgb to use them."); err != nil {
				return nil, err
			}
		}

		for {
			stack := make([]string, 0, len(styles[group]))
			var symbols []string
			for _, layer := range styles[group] {
				if layer.Type == "symbol" {
					symbols = append(symbols, fmt.Sprintf("SVG icon (%s)", layer.IconName))
				} else {
					stack = append(stack, layer.Type)
				}
			}
			stack = append(stack, symbols...)

			if _, err := fmt.Fprintf(out, "\n%s (%s)\nLayers (bottom to top): [%s]\n",
				group.Category, group.GeometryType, strings.Join(stack, ", "),
			); err != nil {
				return nil, err
			}

			for i, layerType := range types {
				if _, err := fmt.Fprintf(out, "%d. %s\n", i+1, layerType); err != nil {
					return nil, err
				}
			}

			choice, err := promptLine(scanner, out, "Layer type to add (Enter to finish): ")
			if err != nil {
				return nil, err
			}

			if choice == "" {
				if len(styles[group]) > 0 {
					break
				}

				if _, err := fmt.Fprintln(out, "Add at least one layer before finishing."); err != nil {
					return nil, err
				}

				continue
			}

			layerType := ""
			for i, candidate := range types {
				if choice == candidate || choice == strconv.Itoa(i+1) {
					layerType = candidate
					break
				}
			}
			if layerType == "" {
				if _, err := fmt.Fprintln(out, "Choose a listed layer type or number."); err != nil {
					return nil, err
				}

				continue
			}

			layer := maplibre.RenderLayerStyle{Type: layerType}
			if layerType == "SVG icon" {
				layer.Type = "symbol"
				layer.IconName, layer.Layout, err = promptIcon(opts.Icons, scanner, out)
				if err != nil {
					return nil, err
				}
			}

			paint, err := promptPaint(spec, layer.Type, scanner, out)
			if err != nil {
				return nil, err
			}

			layer.Paint = paint
			styles[group] = append(styles[group], layer)
		}
	}
	return styles, nil
}

func promptPaint(spec maplibre.Spec, layerType string, scanner *bufio.Scanner, out io.Writer) (maplibre.Paint, error) {
	properties, err := spec.Properties(layerType, "paint")
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)

	if _, err := fmt.Fprintf(out, "\n%s properties\n%s\n",
		layerType, strings.Join(names, "\n"),
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

	return paint, nil
}

func promptIcon(icons map[string]svg.Svg, scanner *bufio.Scanner, out io.Writer) (string, map[string]any, error) {
	names := make([]string, 0, len(icons))
	for name := range icons {
		names = append(names, name)
	}

	sort.Strings(names)
	for i, name := range names {
		if _, err := fmt.Fprintf(out, "%d. %s\n", i+1, name); err != nil {
			return "", nil, err
		}
	}

	var name string
	for {
		input, err := promptRawLine(scanner, out, "SVG icon name or number: ")
		if err != nil {
			return "", nil, err
		}

		// As with category fields, exact names take precedence over numbers.
		if _, exists := icons[input]; exists {
			name = input
			break
		}

		if n, err := strconv.Atoi(strings.TrimSpace(input)); err == nil && n > 0 && n <= len(names) {
			name = names[n-1]
			break
		}

		if _, err := fmt.Fprintln(out, "Choose a listed icon name or number."); err != nil {
			return "", nil, err
		}
	}

	size := 0.5
	for {
		input, err := promptLine(scanner, out, "Icon size [0.5]: ")
		if err != nil {
			return "", nil, err
		}

		if input == "" {
			break
		}

		value, err := strconv.ParseFloat(input, 64)
		if err == nil && value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0) {
			size = value
			break
		}

		if _, err := fmt.Fprintln(out, "Enter a positive finite size."); err != nil {
			return "", nil, err
		}
	}

	overlap := false
	for {
		input, err := promptLine(scanner, out, "WARNING: More performance intensive.\nAllow icon overlap [false]:")
		if err != nil {
			return "", nil, err
		}

		if input == "" {
			break
		}

		value, err := strconv.ParseBool(input)
		if err == nil {
			overlap = value
			break
		}

		if _, err := fmt.Fprintln(out, "Enter true or false."); err != nil {
			return "", nil, err
		}
	}

	ignorePlacement := false
	for {
		input, err := promptLine(scanner, out, "WARNING: More performance intensive .\nIgnore icon placement [false]:")
		if err != nil {
			return "", nil, err
		}

		if input == "" {
			break
		}

		value, err := strconv.ParseBool(input)
		if err == nil {
			ignorePlacement = value
			break
		}

		if _, err := fmt.Fprintln(out, "Enter true or false."); err != nil {
			return "", nil, err
		}
	}

	return name, map[string]any{
		"icon-image":            name,
		"icon-size":             size,
		"icon-allow-overlap":    overlap,
		"icon-ignore-placement": ignorePlacement,
		"symbol-placement":      "point",
	}, nil
}
