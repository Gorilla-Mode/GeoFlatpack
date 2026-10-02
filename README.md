# GeoFlatpack

GeoFlatpack converts each layer in a `.gml` file into its own FlatGeobuf file and generates one shared MapLibre stylesheet. All layers are converted and parsed in memory before styling or writing outputs.

## Requirements

- Go 1.27 or later
- GDAL development libraries with GML and FlatGeobuf support (`gdal-config` available)

## Web example

### Live demo

> Live demo uses the .fgb and .json files in the `test_data` directory.

[Open the live demo](https://gorilla-mode.github.io/GeoFlatpack/)

### Further docs

The `web/` directory contains a browser-based MapLibre example. See the [web README](web/README.md) for setup and app details.


## Usage

### Compile

From the repository root, compile the CLI:

```zsh
cd GeoFlatpack/
go build -o .
```

Then run it:

```zsh
./GeoFlatpack -i path/to/map.gml -o path/to/output/ -f maplibre
```

### Flags

| Argument            | Description                                                                 | Default    | Required? |
|---------------------|-----------------------------------------------------------------------------|------------|-----------|
| `-i`                | Path to the input `.gml` file                                               | —          | Yes       |
| `-o`                | Output base file path or directory (use a trailing `/` for a new directory) | `./`       | No        |
| `-f`                | Stylesheet format: `maplibre` or `sld`                                      | `maplibre` | No        |
| `-h`                | Show help                                                                   | `false`    | No        |
| `-v`                | Verbose output                                                              | `false`    | No        |
| `--write-fgb`       | Write one FlatGeobuf output file per input layer                            | `true`     | No        |
| `--write-style`     | Prompt per layer and write one shared `.gen.maplibre.json` stylesheet       | `true`     | No        |
| `--force-epsg:4326` | Reproject coordinates and CRS metadata to EPSG:4326                         | `true`     | No        |
| `--skip-failures`   | Skip feature conversion failures; can produce incomplete output             | `false`    | No        |

Disable either output with a boolean flag:

```zsh
# Style only: writes roads.gen.maplibre.json; still converts and loads every layer
./GeoFlatpack -i path/to/map.gml -o path/to/roads.fgb --write-fgb=false

# FlatGeobuf only: writes roads.fgb (one layer) or roads.<layer>.fgb (multiple layers); no prompts
./GeoFlatpack -i path/to/map.gml -o path/to/roads.fgb --write-style=false

# Process the input in memory without creating output files or directories
# Combine with -v to inspect every layer as text
./GeoFlatpack -i path/to/map.gml -o path/to/output/ --write-fgb=false --write-style=false

# Preserve the input coordinate system instead of reprojecting to EPSG:4326
./GeoFlatpack -i path/to/map.gml -o path/to/roads.fgb --force-epsg:4326=false
```

Disabled outputs leave any existing files untouched. Both flags default to `true`; use `=false` to disable them.
Disabling style output skips all prompts. Disabling both outputs still converts and loads every layer but creates no
output files or directories.

To let GDAL skip feature conversion failures, enable `--skip-failures` (default: `false`). **Skipped failures can produce
incomplete output.**

```sh
./GeoFlatpack -i input.gml -o ./ --skip-failures
```


### Layer outputs and styling

A single-layer input produces `<base>.fgb`, multiple layers produce `<base>.<layer>.fgb`,
following [GDAL's single-layer FlatGeobuf model](https://gdal.org/en/stable/drivers/vector/flatgeobuf.html#multi-layer-support).

For example, from the `GeoFlatpack/` directory:

```zsh
./GeoFlatpack -i ../some/multi-layer.gml -o ./output/map.fgb
# output/map.Stops.fgb
# output/map.Routes.fgb
# output/map.Zones.fgb
# output/map.gen.maplibre.json
```

The stylesheet always uses the **`.gen` tag** and has one source per FGB, with IDs set to the final filenames without `.fgb`.
Sources are empty GeoJSON placeholders for the consuming application to populate.

## Dependencies

| Module                          | Version                 | Purpose                                                          |
|---------------------------------|-------------------------|------------------------------------------------------------------|
| `github.com/airbusgeo/godal`    | `v0.0.18`               | Provides GDAL access for GML conversion and FlatGeobuf handling. |
| `github.com/gogama/flatgeobuf`  | `v1.0.1`                | Reads and parses FlatGeobuf data.                                |
| `github.com/google/flatbuffers` | `v23.5.26+incompatible` | Underlying serialization dependency used by FlatGeobuf.          |


## Development status

- [x] Convert GML to FGB
- [x] Load FGB into memory
- [x] Parse FGB
- [x] Web demo
- [x] Generate MapLibre stylesheet
- [x] Cli user input for maplibre stylesheet
- [x] Layered GML support
- [ ] Tui user input for maplibre stylesheet
- [ ] Symbology support
