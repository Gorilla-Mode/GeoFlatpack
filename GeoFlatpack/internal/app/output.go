package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolveOutput(input, output string) (string, error) {
	info, err := os.Stat(output)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	if output == "." || strings.HasSuffix(output, string(os.PathSeparator)) || (err == nil && info.IsDir()) {
		name := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
		output = filepath.Join(output, name+".fgb")
	} else if filepath.Ext(output) == "" {
		output += ".fgb"
	}

	return output, nil
}

func layerOutputPaths(output string, names []string) []string {
	if len(names) == 1 {
		return []string{output}
	}

	base := strings.TrimSuffix(output, filepath.Ext(output))
	paths := make([]string, len(names))
	used := make(map[string]bool)

	for i, name := range names {
		// A conservative portable filename alphabet also prevents path traversal.
		name = strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
				return r
			}
			return '_'
		}, name)
		if name == "" {
			name = "layer"
		}
		unique := name
		for suffix := 2; used[strings.ToLower(unique)]; suffix++ {
			unique = fmt.Sprintf("%s_%d", name, suffix)
		}
		used[strings.ToLower(unique)] = true
		paths[i] = base + "." + unique + ".fgb"
	}

	return paths
}
