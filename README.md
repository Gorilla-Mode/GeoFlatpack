# GeoFlatpack

GeoFlatpack is a tool for converting `.gml` files into a FlatGeobuf file. With an associated stylesheet, and symbology support.

## Requirements

- Go 1.27 or later

## Usage

From the repository root, compile the CLI:

```zsh
cd GeoFlatpack/
go build -o .
```

Then run it:

```zsh
./GeoFlatpack -i path/to/map.gml -o path/to/output/ -f maplibre
```

| Argument        | Description                                                            | Default    |
|-----------------|------------------------------------------------------------------------|------------|
| `-i`            | Path to the input `.gml` file (required)                               | —          |
| `-o`            | Output file path or directory (use a trailing `/` for a new directory) | empty      |
| `-f`            | Stylesheet format: `maplibre` or `sld`                                 | `maplibre` |
| `-h`            | Show help                                                              | —          |
| `-v`            | Verbose output                                                         | —          |
| `--write-fgb`   | Write the FlatGeobuf output file                                       | `true`     |
| `--write-style` | Write the generated stylesheet output file                             | `true`     |

Disable either output with a boolean flag:

```zsh
# Style only: writes roads.gen.maplibre.json with roads as the source name
./GeoFlatpack -i path/to/map.gml -o path/to/roads.fgb --write-fgb=false

# FlatGeobuf only: writes roads.fgb
./GeoFlatpack -i path/to/map.gml -o path/to/roads.fgb --write-style=false

# Process the input in memory without creating output files or directories
# can be combined with verbose output to display the fgb file as text
./GeoFlatpack -i path/to/map.gml -o path/to/output/ --write-fgb=false --write-style=false
```

Disabled outputs leave any existing files untouched. Both flags default to `true`; use `=false` to disable them.

## Web example

### Live demo

> Live demo uses the .fgb and .json files in the `test_data` directory.

[Open the live demo](https://gorilla-mode.github.io/GeoFlatpack/)

### Further docs

The `web/` directory contains a browser-based MapLibre example. See the [web README](web/README.md) for setup and app details.

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
- [ ] User input for maplibre stylesheet
- [ ] Symbology support
