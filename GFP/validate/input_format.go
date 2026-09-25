package validate

import (
	"fmt"
	"path/filepath"
)

func Gml(file string) error {
	extension := filepath.Ext(file)
	if extension != ".gml" {
		return fmt.Errorf("input file must have a .gml extension")
	}

	return nil
}
