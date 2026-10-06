package preview

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"GeoFlatpack/fgb"
	"GeoFlatpack/style/maplibre"
	"GeoFlatpack/style/maplibre/svg"
	"github.com/airbusgeo/godal"
)

const Basemap = "https://tiles.openfreemap.org/styles/liberty"

// Request owns immutable styling maps. The session's FGB buffers are read-only.
type Request struct {
	Input         maplibre.LayerStyle
	Icons         map[string]svg.Svg
	Group         *maplibre.StyleGroup
	Width, Height int
	Basemap       string
	Sample        bool
}

type document struct {
	Style   *maplibre.StyleHeader `json:"style"`
	Data    collection            `json:"data"`
	Basemap string                `json:"basemap"`
}

type collection struct {
	Type     string    `json:"type"`
	Features []feature `json:"features"`
}

type feature struct {
	Type       string          `json:"type"`
	Geometry   json.RawMessage `json:"geometry"`
	Properties map[string]any  `json:"properties"`
	ID         any             `json:"id,omitempty"`
}

// Cache one representative and its conversion across paint edits. The source,
// category, chosen group, and SVG companion requirements identify the dataset.
type dataCache struct {
	source     *fgb.Fgb
	field      string
	group      maplibre.StyleGroup
	index      int
	companions bool
	marker     string
	data       collection
}

func prepare(ctx context.Context, request Request) (document, error) {
	return prepareCached(ctx, request, nil)
}

func prepareCached(ctx context.Context, request Request, cache *dataCache) (document, error) {
	if err := ctx.Err(); err != nil {
		return document{}, err
	}
	input := request.Input
	if request.Group == nil {
		return document{}, fmt.Errorf("Choose a feature to preview")
	}
	if input.Data == nil || input.Data.Header == nil || len(input.Data.Features) == 0 {
		return document{}, fmt.Errorf("No geometry to preview")
	}
	group := *request.Group
	styles := make(map[maplibre.StyleGroup][]maplibre.RenderLayerStyle, 1)
	if stack, configured := input.Styles[group]; configured {
		styles[group] = stack
	}
	input.Styles = styles
	companions := false
	for _, layer := range styles[group] {
		companions = companions || (layer.Type == maplibre.RenderSymbol && group.GeometryType != maplibre.Point)
	}
	matching := cache != nil && cache.source == input.Data && cache.field == input.CategoryField && cache.group == group
	index := -1
	if matching {
		index = cache.index
	} else {
		var err error
		index, err = RepresentativeIndex(input.Data, input.CategoryField, group)
		if err != nil {
			return document{}, err
		}
	}
	var data collection
	if matching && cache.companions == companions {
		data, input.VertexMarker = cache.data, cache.marker
	} else {
		sample, err := representativeData(input.Data, index)
		if err != nil {
			return document{}, err
		}
		sampleInput := input
		sampleInput.Data = sample
		output, err := maplibre.PrepareVertexCompanions(&sampleInput)
		if err != nil {
			return document{}, err
		}
		input.VertexMarker = sampleInput.VertexMarker
		data, err = geoJSON(ctx, output)
		if err != nil {
			return document{}, err
		}
		if cache != nil {
			*cache = dataCache{source: input.Data, field: input.CategoryField, group: group, index: index, companions: companions, marker: input.VertexMarker, data: data}
		}
	}
	if request.Sample {
		var err error
		data, err = sampleGeometry(data, group.GeometryType, input.VertexMarker)
		if err != nil {
			return document{}, err
		}
	}
	style, err := maplibre.BuildMapLibreCollectionStyle("preview", []maplibre.LayerStyle{input}, request.Icons)
	if err != nil {
		return document{}, err
	}
	groups, err := maplibre.CollectStyleGroups(input.Data, input.CategoryField)
	if err != nil {
		return document{}, err
	}
	prefix := ""
	for i, candidate := range groups {
		if candidate == group {
			prefix = fmt.Sprintf("%s-%s-%d", input.SourceID, group.GeometryType, i)
			break
		}
	}
	layers := style.Layers[:0]
	for _, layer := range style.Layers {
		if prefix != "" && (layer.ID == prefix || strings.HasPrefix(layer.ID, prefix+"-")) {
			layers = append(layers, layer)
		}
	}
	style.Layers = layers
	if len(data.Features) == 0 {
		return document{}, fmt.Errorf("No geometry to preview")
	}
	return document{Style: style, Data: data, Basemap: request.Basemap}, nil
}

// GDAL reads an independent temporary dataset. Preview reprojection never
// changes the prepared session or the files written by the application.
func geoJSON(ctx context.Context, data *fgb.Fgb) (collection, error) {
	if err := ctx.Err(); err != nil {
		return collection{}, err
	}
	dir, err := os.MkdirTemp("", "gfp-preview-data-")
	if err != nil {
		return collection{}, err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "source.fgb")
	if err := fgb.WriteFgb(data, path); err != nil {
		return collection{}, err
	}
	godal.RegisterAll()
	source, err := godal.Open(path, godal.VectorOnly())
	if err != nil {
		return collection{}, err
	}
	defer source.Close()
	if layers := source.Layers(); len(layers) == 0 || layers[0].SpatialRef() == nil {
		return collection{}, fmt.Errorf("Preview unavailable: unknown CRS")
	} else if wkt, err := layers[0].SpatialRef().WKT(); err != nil || wkt == "" {
		return collection{}, fmt.Errorf("Preview unavailable: unknown CRS")
	}
	jsonPath := filepath.Join(dir, "source.json")
	translated, err := source.VectorTranslate(jsonPath, []string{"-f", "GeoJSON", "-t_srs", "EPSG:4326", "-lco", "RFC7946=YES"})
	if err != nil {
		return collection{}, fmt.Errorf("Preview reprojection: %w", err)
	}
	if err := translated.Close(); err != nil {
		return collection{}, err
	}
	if err := ctx.Err(); err != nil {
		return collection{}, err
	}
	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		return collection{}, err
	}
	var result collection
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber() // Retain exact int64 category values through reprojection.
	if err := decoder.Decode(&result); err != nil {
		return collection{}, err
	}
	return result, nil
}
