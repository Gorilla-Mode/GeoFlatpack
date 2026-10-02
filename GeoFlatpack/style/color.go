package style

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

type Hex string

func ToHex(color color.RGBA) Hex {
	return Hex(fmt.Sprintf("#%02x%02x%02x", color.R, color.G, color.B))
}

func ParseRGB(input string) (Hex, error) {
	input = strings.TrimSpace(input)

	if strings.HasPrefix(strings.ToLower(input), "rgb(") &&
		strings.HasSuffix(input, ")") {
		input = input[4 : len(input)-1]
	}

	parts := strings.Split(input, ",")
	if len(parts) != 3 {
		return "", fmt.Errorf("enter RGB values such as 37, 99, 235")
	}

	var rgb [3]uint8
	for i, part := range parts {
		value, err := parseRGBChannel(part)
		if err != nil {
			return "", err
		}

		rgb[i] = value
	}

	return ToHex(color.RGBA{
		R: rgb[0],
		G: rgb[1],
		B: rgb[2],
		A: 255,
	}), nil
}

func parseRGBChannel(input string) (uint8, error) {
	value, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || value < 0 || value > 255 {
		return 0, fmt.Errorf("each RGB value must be an integer between 0 and 255")
	}

	return uint8(value), nil
}
