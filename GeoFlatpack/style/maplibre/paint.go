package maplibre

import (
	"GeoFlatpack/style"
	"bufio"
	"fmt"
	"io"
	"math"
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

func parsePaintAnswer(q paintQuestion, input string) (any, error) {
	if q.Color {
		return style.ParseRGB(input)
	}

	value, err := strconv.ParseFloat(input, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return nil, fmt.Errorf("enter a finite number greater than or equal to 0")
	}
	if q.Max > 0 && value > q.Max {
		return nil, fmt.Errorf("value must be between 0 and %g", q.Max)
	}
	return value, nil
}

func PromptPaints(
	groups []StyleGroup,
	in io.Reader,
	out io.Writer,
) (map[StyleGroup]Paint, error) {
	scanner := bufio.NewScanner(in)
	paints := make(map[StyleGroup]Paint)

	for _, group := range groups {
		questions, ok := paintQuestions[group.GeometryType]
		if !ok {
			return nil, fmt.Errorf("unsupported geometry: %s", group.GeometryType)
		}

		_, err := fmt.Fprintf(out, "\n%s (%s)\n", group.Category, group.GeometryType)
		if err != nil {
			return nil, err
		}
		paint := make(Paint)

		for _, question := range questions {
			for {
				_, err := fmt.Fprintf(out, "%s [%s]: ", question.Label, question.Default)
				if err != nil {
					return nil, err
				}

				if !scanner.Scan() {
					if err := scanner.Err(); err != nil {
						return nil, err
					}
					return nil, fmt.Errorf("input ended while configuring %q", group.Category)
				}

				input := strings.TrimSpace(scanner.Text())
				if input == "" {
					input = question.Default
				}

				value, err := parsePaintAnswer(question, input)
				if err != nil {
					_, err := fmt.Fprintln(out, err)
					if err != nil {
						return nil, err
					}
					continue
				}

				paint[question.Key] = value
				break
			}
		}

		paints[group] = paint
	}

	return paints, nil
}
