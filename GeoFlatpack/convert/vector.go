package convert

import (
	"GeoFlatpack/validate"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	godal "github.com/airbusgeo/godal"
)

// MemoryFGB owns one GDAL dataset and its virtual file. Call Close after use.
type MemoryFGB struct {
	LayerName string
	Dataset   *godal.Dataset
	// ListColumns records source list fields whose output values are JSON arrays.
	ListColumns []string
	path        string
}

type VectorGeometry struct {
	InputPath     string
	ForceEPSG4326 bool
	SkipFailures  bool
	Format        validate.VectorFormat
}

// VectorToFgb converts each input layer independently, preserving GDAL layer order.
// The caller owns every returned MemoryFGB; failures release partial results.
func VectorToFgb(geometry VectorGeometry) (files []*MemoryFGB, err error) {
	godal.RegisterAll()

	options := []godal.OpenOption{godal.VectorOnly()}

	if geometry.Format == validate.FGB {
		options = append(options, godal.DriverOpenOption("WRITE_GFS=NO"))
	}

	src, err := godal.Open(geometry.InputPath, options...)

	if err != nil {
		return nil, fmt.Errorf("open GML: %w", err)
	}

	defer func() {
		if closeErr := src.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close GML: %w", closeErr))
			for _, file := range files {
				err = errors.Join(err, file.Close())
			}
			files = nil
		}
	}()

	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, err
	}

	return convertLayers(src, fmt.Sprintf("/vsimem/gfp-%x", id[:]), geometry.ForceEPSG4326, geometry.SkipFailures)
}

func convertLayers(src *godal.Dataset, prefix string, forceEPSG4326, skipFailures bool) (files []*MemoryFGB, err error) {
	layers := src.Layers()
	if len(layers) == 0 {
		return nil, fmt.Errorf("input has no layers")
	}

	defer func() {
		if err != nil {
			for _, file := range files {
				err = errors.Join(err, file.Close())
			}
			files = nil
		}
	}()

	for i, layer := range layers {
		listColumns, schemaErr := layerListColumns(layer)
		if schemaErr != nil {
			return files, fmt.Errorf("layer %q schema: %w", layer.Name(), schemaErr)
		}
		file, convertErr := convertLayer(src, layer.Name(), fmt.Sprintf("%s-%d.fgb", prefix, i), forceEPSG4326, skipFailures)
		if convertErr != nil {
			return files, fmt.Errorf("layer %q: %w", layer.Name(), convertErr)
		}
		file.ListColumns = listColumns
		files = append(files, file)
	}

	return files, nil
}

func layerListColumns(layer godal.Layer) ([]string, error) {
	// A detached, empty feature exposes the layer definition without reading
	// any source features. Passing nil geometry does not insert a feature.
	definition, err := layer.NewFeature(nil)

	if err != nil {
		return nil, err
	}

	defer definition.Close()

	var names []string
	for name, field := range definition.Fields() {
		switch field.Type() {
		case godal.FTStringList, godal.FTIntList, godal.FTInt64List, godal.FTRealList:
			names = append(names, name)
		}
	}

	sort.Strings(names)

	return names, nil
}

func convertLayer(src *godal.Dataset, name, path string, forceEPSG4326, skipFailures bool) (*MemoryFGB, error) {
	keep := false
	defer func() {
		if !keep {
			_ = godal.VSIUnlink(path)
		}
	}()

	args := []string{"-f", "FlatGeobuf", "-lco", "TEMPORARY_DIR=/vsimem/",
		"-mapFieldType", "StringList=String(JSON),IntegerList=String(JSON),Integer64List=String(JSON),RealList=String(JSON)"}

	if forceEPSG4326 {
		args = append(args, "-t_srs", "EPSG:4326")
	}

	if skipFailures {
		args = append(args, "-skipfailures")
	}

	args = append(args, name)
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
	dataset, err := godal.Open(path, godal.VectorOnly())

	if err != nil {
		return nil, fmt.Errorf("reopen FGB: %w", err)
	}

	keep = true
	return &MemoryFGB{LayerName: name, Dataset: dataset, path: path}, nil
}

func WriteFgb(fgb *MemoryFGB, output string) error {
	src, err := fgb.OpenReader()
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
	var err error
	if fgb.Dataset != nil {
		err = fgb.Dataset.Close()
		fgb.Dataset = nil
	}

	if fgb.path != "" {
		err = errors.Join(err, godal.VSIUnlink(fgb.path))
		fgb.path = ""
	}
	if err != nil {
		return fmt.Errorf("close layer %q: %w", fgb.LayerName, err)
	}

	return nil
}
