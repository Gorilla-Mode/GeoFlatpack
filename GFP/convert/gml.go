package convert

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/airbusgeo/godal"
)

type MemoryFGB struct {
	Dataset *godal.Dataset // Opened for inspecting the finished FGB
	path    string         // Its /vsimem/ path
}

func GmlToFgb(input string) (*MemoryFGB, error) {
	godal.RegisterAll()

	src, err := godal.Open(input,
		godal.VectorOnly(),
		godal.DriverOpenOption("WRITE_GFS=NO"),
	)
	if err != nil {
		return nil, fmt.Errorf("open GML: %w", err)
	}
	defer func() { _ = src.Close() }()

	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/vsimem/gfp-%x.fgb", id[:])

	// Remove the virtual file if conversion or reopening fails.
	keep := false
	defer func() {
		if !keep {
			_ = godal.VSIUnlink(path)
		}
	}()

	dst, err := src.VectorTranslate(path, []string{
		"-f", "FlatGeobuf",
		"-lco", "TEMPORARY_DIR=/vsimem/",
	})
	if err != nil {
		if dst != nil {
			_ = dst.Close()
		}
		return nil, fmt.Errorf("convert GML to FGB: %w", err)
	}
	if err := dst.Close(); err != nil {
		return nil, fmt.Errorf("finish FGB: %w", err)
	}

	fgb, err := godal.Open(path, godal.VectorOnly())
	if err != nil {
		return nil, fmt.Errorf("reopen FGB: %w", err)
	}

	keep = true
	return &MemoryFGB{Dataset: fgb, path: path}, nil
}

func WriteFgb(fgb *MemoryFGB, output string) error {
	src, err := godal.VSIOpen(fgb.path)
	if err != nil {
		return fmt.Errorf("open in-memory FGB: %w", err)
	}

	// Write beside the destination so a failed copy leaves no partial
	// output file at the requested path.
	dst, err := os.CreateTemp(filepath.Dir(output), ".gfp-*.fgb")
	if err != nil {
		_ = src.Close()
		return err
	}
	defer func() { _ = os.Remove(dst.Name()) }()

	_, copyErr := io.Copy(dst, src)
	srcCloseErr := src.Close()
	dstCloseErr := dst.Close()
	if err := errors.Join(copyErr, srcCloseErr, dstCloseErr); err != nil {
		return fmt.Errorf("copy FGB: %w", err)
	}

	return os.Rename(dst.Name(), output)
}

func PrintFgb(fgb *MemoryFGB) error {
	ds := fgb.Dataset
	for _, layer := range ds.Layers() {
		fmt.Printf("Layer: %s\n", layer.Name())
		layer.ResetReading()

		for i := 1; ; i++ {
			feature := layer.NextFeature()
			if feature == nil {
				break
			}

			fmt.Printf("  Feature %d\n", i)
			if geometry := feature.Geometry(); geometry != nil {
				wkt, err := geometry.WKT()
				if err != nil {
					feature.Close()
					return fmt.Errorf("feature %d geometry: %w", i, err)
				}
				fmt.Println("    Geometry: ", wkt)
			}

			for name, field := range feature.Fields() {
				fmt.Printf("    %s: %s\n", name, field.String())
			}
			feature.Close()
		}
	}
	return nil
}

func (fgb *MemoryFGB) Close() error {
	return errors.Join(
		fgb.Dataset.Close(),
		godal.VSIUnlink(fgb.path),
	)
}
