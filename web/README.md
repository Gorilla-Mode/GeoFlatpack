# GeoFlatpack web

A single fullscreen MapLibre map built with React, TypeScript, and Vite.

## Getting started

Install a current Node.js LTS release, then from the repository root:

```sh
cd web
npm install
npm run dev
```

Open the URL printed by Vite.

## Scripts

| Command           | Purpose                                                        |
|-------------------|----------------------------------------------------------------|
| `npm run dev`     | Start the development server                                   |
| `npm test`        | Check file decoding and stylesheet validation (Node.js 22.18+) |
| `npm run build`   | Check TypeScript and build the static app into `dist/`         |
| `npm run preview` | Serve the production build locally                             |

## Map data and styling

| File                                                                                                                                                              | Purpose                                                   |
|-------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------|
| [`sample-obstacles.fgb`](../test_data/sample-obstacles.fgb)                                                                                                       | Decoded in the browser into a GeoJSON source              |
| [`sample-obstacles.maplibre.json`](../test_data/sample-obstacles.maplibre.json)                                                                                   | Initial view, obstacle filters, and paint properties      |
| [`Samfunnssikkerhet_0000_Norge_25833_Brannstasjoner_GML.fgb`](../test_data/Samfunnssikkerhet_0000_Norge_25833_Brannstasjoner_GML.fgb)                             | Brannstasjoner data decoded into a GeoJSON source         |
| [`Samfunnssikkerhet_0000_Norge_25833_Brannstasjoner_GML.gen.maplibre.json`](../test_data/Samfunnssikkerhet_0000_Norge_25833_Brannstasjoner_GML.gen.maplibre.json) | Generated filters and paint properties for Brannstasjoner |

The two Skytefelt sources use `Forurensning_0000_Norge_3035_Skytefelt_GML.SkyteOg_vingsfelt.fgb` and `Forurensning_0000_Norge_3035_Skytefelt_GML.Skytefeltgrense.fgb`, with the shared `Forurensning_0000_Norge_3035_Skytefelt_GML.gen.maplibre.json` stylesheet.

All built-in files are imported directly from `test_data/`. Vite bundles the three stylesheets and emits all four FGB files as production assets. Only sample obstacles loads initially; other sources load when enabled and stay available in memory. Layers, uploads, visibility, and stylesheet assignments are not saved between page loads.

## Source, uploads, and inspectors

| Concept            | Notes                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
|--------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Source name**    | FGB filename without extension, e.g. `sample-obstacles.fgb` → `sample-obstacles`. Must match the stylesheet's `sources` key and each layer's `source`. Update both the import path and stylesheet references when changing files. Asset hashes are separate.                                                                                                                                                                                                         |
| **CLI output**     | Writes two files per input: `<output>/<input-name>.fgb` and `.gen.maplibre.json`. Default output dir uses the input name, e.g. `map.gml` → `map.fgb` + `map.gen.maplibre.json`. `-o /output/roads.fgb` writes to that dir as `roads.fgb` + `roads.gen.maplibre.json` (source `roads`). `-o .` writes named files to the current dir.                                                                                                                                 |
| **Upload files** | Pick one or more local `.fgb` files and one matching `.json` stylesheet, then **Add layers**. Every file in the batch receives that stylesheet. Later batches can use other stylesheets. Uploads add visible layers alongside existing ones, center on the new batch, select its first layer for inspection, and open the source list. Files stay in the browser. Invalid files or mismatched sources reject the whole batch without changing existing layers or styles. |
| **Select sources** | Check any combination of Sample obstacles, Brannstasjoner, Skytefelt areas, Skytefelt boundaries, and uploaded layers to show them together. Each row has a stylesheet selector listing compatible built-in and uploaded styles, plus an **Inspect** button. Changing a stylesheet updates only that layer; both Skytefelt layers initially share one style. Loading failures offer a retry for the affected source. Tab navigates controls, Space toggles checkboxes, and Escape closes the panel. |
| **Inspect Header** | The row’s **Inspect** button selects the layer used by all three inspectors. Header shows the parsed header and expandable features; GeoJSON shows decoded data; Map Style shows the complete assigned stylesheet. Hidden layers remain inspectable after loading. An unloaded source opens its style inspector. A parsed header stays inspectable if decoding later fails. |

Only style layers referencing the matching filename source render above the basemap; backgrounds and unrelated sources are skipped. Compatibility requires a matching GeoJSON source and at least one style layer using it. Runtime source and layer IDs are unique per dataset, so repeated filenames and shared styles can coexist. Uploaded layer and style labels include the batch number to distinguish them.

Enabling a layer centers on its bounds. **Center on bbox** fits all visible datasets. Click a feature to see its source label and properties, including when datasets have overlapping feature IDs. The toolbar inspectors remain focused on the layer selected using **Inspect**.

## Basemap

[OpenFreeMap Liberty](https://openfreemap.org/quick_start/) (street names and places) needs internet. The sample style's background layer is skipped so it stays visible; obstacles render above it, and MapLibre handles the attribution.

## GitHub Pages

URL: https://gorilla-mode.github.io/GeoFlatpack/

Once: repo **Settings → Pages → Source → GitHub Actions** ([docs](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages)).

The [workflow](../.github/workflows/deploy-pages.yml) runs on every push to `master`. Also runnable manually from **Actions → Deploy to GitHub Pages → Run workflow** with `master`. Uses Node.js 22, `npm ci`, a TS check, and a build. Before uploading `web/dist` it verifies the emitted obstacle and Brannstasjoner FGB files match their sources byte-for-byte and that CSS + MapLibre worker assets exist. Deployments run serially without cancelling active ones. The `github-pages` environment exposes the deployed URL.

Vite paths support the `/GeoFlatpack/` host. All three stylesheets are bundled into the app; the four FGB files, CSS, and worker are emitted as assets. The workflow also copies the two Skytefelt FGB files and their shared stylesheet to `test_data/` in the published site, preserving their filenames. The basemap still needs internet.
