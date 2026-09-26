# GeoFlatpack web

A single fullscreen MapLibre map built with React, TypeScript, and Vite.

## Getting started

Install a current Node.js LTS release, then run from the repository root:

```sh
cd web
npm install
npm run dev
```

Open the local URL printed by Vite.

## Scripts

| Command           | Purpose                                                |
|-------------------|--------------------------------------------------------|
| `npm run dev`     | Start the development server                           |
| `npm run build`   | Check TypeScript and build the static app into `dist/` |
| `npm run preview` | Serve the production build locally                     |

## Map data and styling

| File                                                                            | Use                                                               |
|---------------------------------------------------------------------------------|-------------------------------------------------------------------|
| [`sample-obstacles.fgb`](../test_data/sample-obstacles.fgb)                     | Decoded in the browser into a GeoJSON source                      |
| [`sample-obstacles.maplibre.json`](../test_data/sample-obstacles.maplibre.json) | Supplies the initial view, obstacle filters, and paint properties |

The sample files are imported directly from `test_data/`. Vite bundles the style and copies the FGB into the production assets; no manual copying or backend is needed.

The [OpenFreeMap Liberty basemap](https://openfreemap.org/quick_start/) provides streets and place names and requires internet access. The sample style's background layer is skipped so the basemap stays visible. Obstacle layers appear above the basemap, and MapLibre displays the basemap attribution.
