package svg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Svg struct {
	Key string
	Svg string
}

func ReadSvgs(dir string) (map[string]Svg, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var svgs = make(map[string]Svg)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if !file.Type().IsRegular() {
			continue
		}

		if filepath.Ext(file.Name()) != ".svg" {
			continue
		}

		svg, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read svg: %w", err)
		}

		name := strings.TrimSuffix(file.Name(), ".svg")
		svgs[name] = Svg{Key: name, Svg: string(svg)}
	}
	return svgs, nil
}
