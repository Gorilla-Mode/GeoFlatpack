# GeoFlatpack web

A fullscreen MapLibre viewer built with React, TypeScript, and Vite.

[Live demo](https://gorilla-mode.github.io/GeoFlatpack/) · [CLI usage](../README.md#usage) · [Sample data](../test_data/)

| Command           | Purpose                                              |
|-------------------|------------------------------------------------------|
| `npm run dev`     | Start the development server                         |
| `npm test`        | Check decoding and style validation (Node.js 22.18+) |
| `npm run build`   | Check TypeScript and build into `dist/`              |
| `npm run preview` | Serve the production build locally                   |

## Using the map

Skytefelt areas loads and is selected by default. Other built-in sources are Sample obstacles, Brannstasjoner, and Skytefelt boundaries. Sources load when enabled and remain in memory.

- **Upload files:** Choose one or more `.fgb` files and one matching JSON stylesheet per batch. New layers are added and centered; invalid batches leave existing layers unchanged.
- **Select sources:** Toggle visibility, assign compatible stylesheets, inspect a source, or retry a failed load.
- **Layer order:** The bottom-left panel lists active sources only. Drag handles or use the arrows; top rows draw above lower rows.
- **Inspect:** Select a source to view its Header, GeoJSON, or Map Style. Loaded sources remain inspectable when hidden.
- **Center on bbox:** Fit all visible datasets. Enabling a source centers on its bounds.
- **Feature details:** Click a feature to see its source and properties.

Use Tab to navigate controls, Space to toggle checkboxes, and Escape to close panels. Files stay in your browser; uploads and settings reset on reload.

## Stylesheets

Each FGB filename without its extension must match a GeoJSON source key and at least one layer's `source` in the stylesheet. For example, `sample-obstacles.fgb` matches `sample-obstacles`.

Only layers using the matching source render; backgrounds and unrelated sources are skipped. Shared stylesheets and duplicate filenames can coexist. Upload labels include the batch number.

### SVG icons

See the [sample stylesheet](../test_data/sample-obstacles.maplibre.json) for a complete example.

- Store inline SVG strings by name in the stylesheet's `metadata["geoflatpack:icons"]`.
- Reference a name with a symbol layer's `layout["icon-image"]`, such as `"star"`. Use `icon-size` to scale it.
- For polygon vertices, set the symbol layer's `metadata["geoflatpack:placement"]` to `"vertices"` and `symbol-placement` to `"point"`. Its filters see Point geometry with the original properties; fills, outlines, and popups retain the original feature.

These metadata keys are GeoFlatpack extensions. The loader supports inline SVGs and literal icon names, not external paths, sprite atlases, or expression-based icon lookup. Failed icons report an error while geometry and valid icons remain visible.


## Usage

This single-dataset example follows the demo's loading steps. In a separate Vite page, add a map container and combine the TypeScript snippets below into `web/src/example.ts`. The imports use the existing demo dependencies and sample files.

```html
<div id="map" style="width: 100%; height: 100vh"></div>
<script type="module" src="./src/example.ts"></script>
```

### 1. Create the map

Load MapLibre's CSS and worker, then wait for the basemap before adding overlays. The helper functions called here are defined in the following steps.

```ts
import { Map as MapLibreMap, NavigationControl, setWorkerUrl } from 'maplibre-gl';
import type { LayerSpecification, StyleSpecification } from 'maplibre-gl';
import 'maplibre-gl/dist/maplibre-gl.css';
import workerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url';
import { deserialize } from 'flatgeobuf/lib/mjs/geojson.js';
import type { FeatureCollection, Geometry, MultiPoint, Position } from 'geojson';
import fgbUrl from '../../test_data/sample-obstacles.fgb?url';
import styleJson from '../../test_data/sample-obstacles.maplibre.json';

const stylesheet = styleJson as unknown as StyleSpecification;
const sourceId = 'sample-obstacles'; // Original filename without .fgb, not Vite's hashed URL.
setWorkerUrl(workerUrl);
const map = new MapLibreMap({
  container: 'map',
  style: 'https://tiles.openfreemap.org/styles/liberty',
  center: stylesheet.center,
  zoom: stylesheet.zoom,
});
map.addControl(new NavigationControl());
map.on('error', event => console.error(event.error));
map.once('load', () => { void addOverlay().catch(console.error); });
```

### 2. Decode the FGB into GeoJSON

MapLibre's GeoJSON source needs decoded features, rather than the FGB URL.

```ts
async function loadData(): Promise<FeatureCollection> {
  const response = await fetch(fgbUrl);
  if (!response.ok) throw new Error(`Unable to load FGB: HTTP ${response.status}`);
  const bytes = new Uint8Array(await response.arrayBuffer());
  const data: FeatureCollection = { type: 'FeatureCollection', features: [] };
  for await (const feature of deserialize(bytes)) {
    feature.id ??= data.features.length;
    data.features.push(feature);
  }
  return data;
}
```

### 3. Register the embedded SVGs

Decode each SVG through a browser image, then register it before adding symbol layers. Track failed icons so their references can be omitted while geometry still renders.

```ts
async function loadIcons(): Promise<Map<string, string | null>> {
  const ids = new Map<string, string | null>();
  const metadata = stylesheet.metadata;
  const icons = metadata && typeof metadata === 'object' && 'geoflatpack:icons' in metadata
    ? metadata['geoflatpack:icons'] : undefined;
  if (!icons || typeof icons !== 'object' || Array.isArray(icons)) return ids;

  for (const [name, svg] of Object.entries(icons)) {
    let url: string | undefined;
    try {
      if (typeof svg !== 'string') throw new Error('Expected an inline SVG string.');
      url = URL.createObjectURL(new Blob([svg], { type: 'image/svg+xml' }));
      const image = new Image();
      image.src = url;
      await image.decode();
      if (!image.naturalWidth || !image.naturalHeight) throw new Error('SVG has no size.');
      const id = `${sourceId}-icon-${encodeURIComponent(name)}`;
      map.addImage(id, image, { pixelRatio: 1, sdf: false });
      if (!map.hasImage(id)) throw new Error('Unable to register image.');
      ids.set(name, id);
    } catch (error) {
      ids.set(name, null);
      console.error(`Icon "${name}" could not load:`, error);
    } finally {
      if (url) URL.revokeObjectURL(url);
    }
  }
  return ids;
}
```

### 4. Add the sources and styled layers

The sample places stars at polygon vertices. Build a MultiPoint source for those symbols, keeping feature properties and omitting each ring's repeated closing coordinate. Their filters see Point geometry; fills and outlines use the original source.

```ts
function polygonVertices(geometry: Geometry | null): Position[] {
  if (geometry?.type === 'MultiPolygon') {
    return geometry.coordinates.flatMap(coordinates => polygonVertices({ type: 'Polygon', coordinates }));
  }
  if (geometry?.type === 'GeometryCollection') return geometry.geometries.flatMap(polygonVertices);
  if (geometry?.type !== 'Polygon') return [];
  return geometry.coordinates.flatMap(ring => {
    const first = ring[0], last = ring[ring.length - 1];
    return ring.length > 1 && first[0] === last[0] && first[1] === last[1] ? ring.slice(0, -1) : ring;
  });
}

async function addOverlay() {
  const data = await loadData();
  const icons = await loadIcons();
  const source = stylesheet.sources[sourceId];
  if (!source || source.type !== 'geojson') throw new Error('Expected a matching GeoJSON source.');
  map.addSource(sourceId, { ...source, data });

  const vertexSourceId = `${sourceId}-vertices`;
  const vertices: FeatureCollection<MultiPoint> = {
    type: 'FeatureCollection',
    features: data.features.flatMap((feature, index) => {
      const coordinates = polygonVertices(feature.geometry);
      return coordinates.length ? [{
        type: 'Feature' as const, id: index, properties: feature.properties,
        geometry: { type: 'MultiPoint' as const, coordinates },
      }] : [];
    }),
  };
  map.addSource(vertexSourceId, { type: 'geojson', data: vertices });

  for (const layer of stylesheet.layers) {
    if (!('source' in layer) || layer.source !== sourceId) continue;
    let rendered: LayerSpecification = { ...layer, id: `${sourceId}-${layer.id}` };
    if (rendered.type === 'symbol') {
      const metadata = rendered.metadata;
      if (metadata && typeof metadata === 'object' && 'geoflatpack:placement' in metadata &&
        metadata['geoflatpack:placement'] === 'vertices') {
        rendered = { ...rendered, source: vertexSourceId };
      }
      const layout = { ...rendered.layout };
      const name = layout['icon-image'];
      if (typeof name === 'string' && icons.has(name)) {
        const id = icons.get(name);
        if (id) layout['icon-image'] = id;
        else delete layout['icon-image'];
      }
      rendered = { ...rendered, layout };
    }
    map.addLayer(rendered);
  }
}
```

Layer order follows the stylesheet; backgrounds and other sources are skipped. SVG metadata and vertex placement are GeoFlatpack conventions implemented by this code, not built-in MapLibre behavior.

For the full implementation—including validation, cancellation, cleanup, multiple datasets, and popups—see [App.tsx](src/App.tsx), [dataset.ts](src/dataset.ts), [styleIcons.ts](src/styleIcons.ts), and [mapLayers.ts](src/mapLayers.ts).

## Local development
Install a current Node.js LTS release, then run from the repository root:

```sh
cd web
npm install
npm run dev
```

Open the URL printed by Vite.
