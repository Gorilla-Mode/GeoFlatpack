# GeoFlatpack

GeoFlatpack is a tool for converting `.gml` files into a FlatGeobuf file. With an associated stylesheet, and symbology support.

## Requirements

- Go 1.27 or later

## Usage

From the repository root, compile the CLI:

```zsh
go build -o gfp ./GFP
```

Then run it:

```zsh
./gfp -i path/to/map.gml -o path/to/output -f maplibre
```

| Argument | Description                              | Default    |
|----------|------------------------------------------|------------|
| `-i`     | Path to the input `.gml` file (required) | —          |
| `-o`     | Path to the output directory             | `.`        |
| `-f`     | Stylesheet format: `maplibre` or `sld`   | `maplibre` |
| `-h`     | Show help                                | —          |
| `-v`     | Verbose output                           | —          |

The output path is `<output-directory>/<input-name>.gfp`. For example, `map.gml` produces `map.gfp` in the chosen output directory.

## Web example

### Live demo

> Live demo uses the the .fgb and .json files in the `test_data` directory.

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
- [ ] Generate MapLibre stylesheet
- [ ] Symbology support
