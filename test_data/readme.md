# Test data

This directory contains sample data for testing the [web app](../web/README.md).

## Sample obstacles

`sample-obstacles.gml` contains five features in EPSG:4326: the original street-light Point, power-line LineString, and building Polygon, plus two manually prepared MultiPoint companions. `Obstacle.4` copies the building's `kind`, `name`, and `height_m` attributes and stores its four corners without the repeated closing coordinate. `Obstacle.5` copies the power line's attributes and stores its three vertices. Each companion has a distinct feature ID. The street light has no companion.

The sample stylesheet uses the MultiPoint companions for four building stars and three power-line stars. It filters symbols by Point geometry and `kind`, with fills and lines beneath their corresponding stars. The street light retains its circle style. All five features are stored in the FGB and available in the browser's inspectors; clicking a star uses its companion's copied properties.

Regenerate the bundled FGB from the repository root with the installed GDAL converter:

```sh
sample_tmp_dir=$(mktemp -d) &&
ogr2ogr -f FlatGeobuf -oo WRITE_GFS=NO -nlt GEOMETRY -a_srs EPSG:4326 -nln sample-obstacles "$sample_tmp_dir/sample-obstacles.fgb" test_data/sample-obstacles.gml &&
mv "$sample_tmp_dir/sample-obstacles.fgb" test_data/sample-obstacles.fgb &&
rmdir "$sample_tmp_dir"
```

The temporary directory allows replacing the existing FGB only after conversion succeeds. `-nlt GEOMETRY` retains mixed geometry, and `-a_srs EPSG:4326` records the sample's existing coordinate reference system. This command converts the manually prepared features; it does not generate companions. Automatic, style-driven companion generation for arbitrary polygons and lines is deferred to a future CLI change.

## Files

| File                                                                      | Description                                         | How it is maintained                  |
|---------------------------------------------------------------------------|-----------------------------------------------------|---------------------------------------|
| `sample-obstacles.gml`                                                    | Sample obstacle data in GML format                  | Hard-coded                            |
| `sample-obstacles.maplibre.json`                                          | Sample obstacle data in MapLibre JSON format        | Hard-coded                            |
| `sample-obstacles.fgb`                                                    | The same sample data in FlatGeobuf format           | Generated from `sample-obstacles.gml` |
| `Samfunnssikkerhet_0000_Norge_25833_Brannstasjoner_GML.gml`               | Brannstasjon data from GeoNorge DSB in GML format   | Source data from GeoNorge DSB         |
| `Samfunnssikkerhet_0000_Norge_25833_Brannstasjoner_GML.fgb`               | Brannstasjon data in FlatGeobuf format              | Generated from the GML source         |
| `Samfunnssikkerhet_0000_Norge_25833_Brannstasjoner_GML.gen.maplibre.json` | Generated MapLibre stylesheet for Brannstasjon data | Generated from the GML source         |
| `Forurensning_0000_Norge_3035_Skytefelt_GML.SkyteOg_vingsfelt.fgb` | Skytefelt area data in FlatGeobuf format | Generated from the Skytefelt GML source |
| `Forurensning_0000_Norge_3035_Skytefelt_GML.Skytefeltgrense.fgb` | Skytefelt boundary data in FlatGeobuf format | Generated from the Skytefelt GML source |
| `Forurensning_0000_Norge_3035_Skytefelt_GML.gen.maplibre.json` | Shared MapLibre stylesheet for both generated Skytefelt files | Generated from the Skytefelt GML source |
