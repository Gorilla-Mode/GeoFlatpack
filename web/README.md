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

## GitHub Pages

The expected site URL is https://gorilla-mode.github.io/GeoFlatpack/.

One-time setup: in the repository, select **Settings → Pages → Source → GitHub Actions**, as described in the [GitHub Pages documentation](https://docs.github.com/en/pages/getting-started-with-github-pages/using-custom-workflows-with-github-pages).

The [deployment workflow](../.github/workflows/deploy-pages.yml) runs on every push to `master`, including changes to either sample data file. Once the workflow is on `master`, you can also run it manually from **Actions → Deploy to GitHub Pages → Run workflow**, selecting `master`.

The workflow uses Node.js 22 and `npm ci`, checks TypeScript, and builds the app. Before uploading only `web/dist`, it verifies that the emitted FGB matches the source byte-for-byte and that CSS and MapLibre worker assets exist. Deployments run serially without cancelling an active deployment, and the `github-pages` environment exposes the deployed URL.

Vite's relative asset paths support the `/GeoFlatpack/` project path. The sample style is bundled into the app, while the FGB, CSS, and bundled worker are emitted as assets. The basemap still requires internet access.
