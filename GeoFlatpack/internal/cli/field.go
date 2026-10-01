package cli

import (
	"GeoFlatpack/fgb"
	"GeoFlatpack/style/maplibre"
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// NewScanner creates the shared scanner for an entire styling session.
func NewScanner(in io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	return scanner
}

// PromptStyleWithScanner retains buffered answers between input layers.
func PromptStyleWithScanner(data *fgb.Fgb, scanner *bufio.Scanner, out io.Writer) (string, map[maplibre.StyleGroup][]maplibre.RenderLayerStyle, error) {
	if data != nil && data.Header != nil && len(data.Features) == 0 {
		return "", nil, nil
	}

	fields, err := maplibre.DiscoverCategoryFields(data)
	if err != nil {
		return "", nil, err
	}

	field, err := PromptCategoryField(fields, scanner, out)
	if err != nil {
		return "", nil, err
	}

	groups, err := maplibre.CollectStyleGroups(data, field)
	if err != nil {
		return "", nil, err
	}

	styles, err := PromptStyles(groups, scanner, out)

	return field, styles, err
}

func PromptCategoryField(fields []maplibre.CategoryField, scanner *bufio.Scanner, out io.Writer) (string, error) {
	if _, err := fmt.Fprintln(out, "Styling category fields:\n0. No category field (one style per geometry type)"); err != nil {
		return "", err
	}

	for i, field := range fields {
		status := ""
		if field.Unavailable != "" {
			status = "; unavailable: " + field.Unavailable
		}

		if _, err := fmt.Fprintf(out, "%d. %s (%s; %d distinct; examples: %s%s)\n",
			i+1, field.Name, strings.Join(field.Types, "/"), field.Distinct,
			strings.Join(field.Examples, ", "), status); err != nil {
			return "", err
		}
	}
	for {
		input, err := promptRawLine(scanner, out, "Which field should define the styling categories? ")

		if err != nil {
			return "", fmt.Errorf("select category field: %w", err)
		}

		// Exact names take precedence if a field itself has a numeric name.
		index := -1
		for i, field := range fields {
			if input == field.Name && input != "" {
				index = i
				break
			}
		}

		if index == -1 {
			input = strings.TrimSpace(input)
			if input == "0" || input == "No category field" {
				return "", nil
			}

			if n, err := strconv.Atoi(input); err == nil && n > 0 && n <= len(fields) {
				index = n - 1
			}
		}

		message := "Invalid selection. Enter a listed number or exact field name (0 for No category field)."
		if index >= 0 {
			field := fields[index]

			if field.Unavailable == "" {
				return field.Name, nil
			}

			message = fmt.Sprintf("Field %q is unavailable: %s. Choose another field or 0.", field.Name, field.Unavailable)
		}

		if _, err := fmt.Fprintln(out, message); err != nil {
			return "", err
		}
	}
}
