package validate

import (
	"fmt"
	"path/filepath"
)

type VectorFormat int

const (
	NOFORMAT VectorFormat = iota
	FGB
	FGDB
)

func InputFormat(file string) (VectorFormat, error) {
	extension := filepath.Ext(file)

	Format := NOFORMAT
	if extension == ".gml" {
		Format = FGB
	} else if extension == ".gdb" {
		Format = FGDB
	}

	if extension != ".gml" && extension != ".gdb" {
		return Format, fmt.Errorf("input file has an invalid extension: %s", extension)
	}

	return Format, nil
}
