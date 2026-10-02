package svg

import (
	"fmt"
	"os"
)

type Svg struct {
	Key string
	Svg string
}
func readSvg(path string) (Svg, error) {
	svg, err := os.ReadFile(path)

	if err != nil {
		return Svg{}, fmt.Errorf("failed to read svg: %w", err)
	}

	return Svg{
		Key: path,
		Svg: string(svg),
	}, nil
}