# Test data

This directory contains sample data for testing the [web app](../web/README.md).

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
