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

The `web/` directory contains a browser-based MapLibre example that displays the sample obstacle data from `test_data/`. To run it locally, install a current Node.js LTS release and, from the repository root, run:

```sh
cd web
npm install
npm run dev
```

Open the local URL printed by Vite. The map uses an online basemap, so it requires internet access. See the [web example README](web/README.md) for details about its scripts and sample data.

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
- [ ] Generate MapLibre stylesheet
- [ ] Symbology support
