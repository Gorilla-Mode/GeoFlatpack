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

The output path is `<output-directory>/<input-name>.gfp`. For example, `map.gml` produces `map.gfp` in the chosen output directory.

## Development status

- [ ] Convert GML to FGB
- [ ] Generate MapLibre stylesheet
- [ ] Symbology support
