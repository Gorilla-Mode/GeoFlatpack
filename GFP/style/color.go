package style

import (
	"fmt"
	"image/color"
)

type Hex string

func ToHex(color color.RGBA) Hex {
	return Hex(fmt.Sprintf("#%02x%02x%02x", color.R, color.G, color.B))
}
