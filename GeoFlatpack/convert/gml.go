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

func GmlToFgb(input string, forceEPSG4326 bool, skipFailures bool) (*MemoryFGB, error) {
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

	args := []string{
		"-f", "FlatGeobuf",
		"-lco", "TEMPORARY_DIR=/vsimem/",
	}
	if forceEPSG4326 {
		args = append(args, "-t_srs", "EPSG:4326")
	}
	if skipFailures {
		args = append(args, "-skipfailures")
	}

	dst, err := src.VectorTranslate(path, args)
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

func (fgb *MemoryFGB) Close() error {
	return errors.Join(
		fgb.Dataset.Close(),
		godal.VSIUnlink(fgb.path),
	)
}

func (fgb *MemoryFGB) OpenReader() (io.ReadCloser, error) {
	return godal.VSIOpen(fgb.path)
}
