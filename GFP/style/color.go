package style

import "fmt"

type RGB struct {
	R uint8
	G uint8
	B uint8
}

type Hex string

func (col RGB) Hex() Hex {
	return Hex(fmt.Sprintf("#%02x%02x%02x", col.R, col.G, col.B))
}
