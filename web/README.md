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

| Command | Purpose |
|---|---|
| `npm run dev` | Start the development server |
| `npm test` | Check file decoding and stylesheet validation (Node.js 22.18+) |
| `npm run build` | Check TypeScript and build the static app into `dist/` |
| `npm run preview` | Serve the production build locally |

## Map data and styling

| File | Purpose |
|---|---|
| [`sample-obstacles.fgb`](../test_data/sample-obstacles.fgb) | Decoded in the browser into a GeoJSON source |
| [`sample-obstacles.maplibre.json`](../test_data/sample-obstacles.maplibre.json) | Initial view, obstacle filters, and paint properties |

Imported directly from `test_data/`. Vite bundles the style and copies the FGB into production assets — no manual copying or backend.

## Source, uploads, and inspectors

| Concept | Notes |
|---|---|
| **Source name** | FGB filename without extension, e.g. `sample-obstacles.fgb` → `sample-obstacles`. Must match the stylesheet's `sources` key and each layer's `source`. Update both the import path and stylesheet references when changing files. Asset hashes are separate. |
| **Upload files** | Pick a local `.fgb` + matching `.json` stylesheet, then **Display on map** (e.g. `roads.fgb` + `roads.gen.maplibre.json` → `roads`). Both read in the browser. Replaces the sample, centers map on bounds, and updates the header, GeoJSON, and style inspectors. Invalid files or mismatched source names error without removing the current dataset. |
| **Inspect Header** | Shows parsed header: columns, geometry type, bounds, feature count, spatial index node size, CRS. Expandable tree — focus a branch and press Enter/Space to toggle. Scrolls long values; reopen re-expands all. Shares the panel with **Inspect GeoJSON** and **Inspect Map Style**. |

Only layers using the matching source render above the basemap; stylesheet backgrounds and unrelated sources are skipped. The header button appears once the decoder parses the header — no extra download or basemap wait — and stays available if feature decoding fails, alongside the error.

## Basemap

[OpenFreeMap Liberty](https://openfreemap.org/quick_start/) (street names and places) needs internet. The sample style's background layer is skipped so it stays visible; obstacles render above it, and MapLibre handles the attribution.

## GitHub Pages

URL: https://gorilla-mode.github.io/GeoFlatpack/

Once: repo **Settings → Pages → Source → GitHub Actions** ([docs](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages)).

The [workflow](../.github/workflows/deploy-pages.yml) runs on every push to `master`. Also runnable manually from **Actions → Deploy to GitHub Pages → Run workflow** with `master`. Uses Node.js 22, `npm ci`, a TS check, and a build. Before uploading `web/dist` it verifies the emitted FGB matches byte-for-byte and that CSS + MapLibre worker assets exist. Deployments run serially without cancelling active ones. The `github-pages` environment exposes the deployed URL.

Vite paths support the `/GeoFlatpack/` host. The style is bundled into the app; the FGB, CSS, and worker are emitted as assets. The basemap still needs internet.
