import { useEffect, useRef, useState } from 'react';
import { Map, NavigationControl, setWorkerUrl } from 'maplibre-gl';
import type { StyleSpecification } from 'maplibre-gl';
import mapWorkerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url';
import { deserialize } from 'flatgeobuf/lib/mjs/geojson.js';
import type { FeatureCollection } from 'geojson';
import sampleStyleJson from '../../test_data/sample-obstacles.maplibre.json';
import sampleUrl from '../../test_data/sample-obstacles.fgb?url';

// JSON imports widen literal types and coordinate tuples.
const sampleStyle = sampleStyleJson as unknown as StyleSpecification;
const sourceId = 'geoflatpack-obstacles';

// Let Vite bundle the worker and its imports for development and production.
setWorkerUrl(mapWorkerUrl);

export default function App() {
  const container = useRef<HTMLDivElement>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!container.current) return;

    const controller = new AbortController();
    let map: Map;
    setLoading(true);
    setError(null);

    try {
      map = new Map({
        container: container.current,
        style: 'https://tiles.openfreemap.org/styles/liberty',
        center: sampleStyle.center,
        zoom: sampleStyle.zoom,
      });
    } catch {
      setError('Unable to start the map. Check that WebGL is available in your browser.');
      setLoading(false);
      return;
    }

    map.addControl(new NavigationControl(), 'top-right');
    map.on('error', () => {
      if (!controller.signal.aborted) {
        setError('Some map resources could not be loaded. Check your connection and reload.');
      }
    });
    const ready = new Promise<void>((resolve) => map.once('load', () => resolve()));

    async function loadSample() {
      try {
        const response = await fetch(sampleUrl, { signal: controller.signal });
        if (!response.ok) throw new Error(`HTTP ${response.status}`);

        const bytes = new Uint8Array(await response.arrayBuffer());
        const data: FeatureCollection = { type: 'FeatureCollection', features: [] };
        for await (const feature of deserialize(bytes)) {
          if (controller.signal.aborted) return;
          data.features.push(feature);
        }

        await ready;
        if (controller.signal.aborted) return;

        map.addSource(sourceId, { type: 'geojson', data });
        // The sample background would hide the street basemap.
        for (const layer of sampleStyle.layers) {
          if ('source' in layer && layer.source === 'obstacles') {
            map.addLayer({ ...layer, id: `geoflatpack-${layer.id}`, source: sourceId });
          }
        }
        setLoading(false);
      } catch (cause) {
        if (controller.signal.aborted) return;
        const detail = cause instanceof Error ? ` (${cause.message})` : '';
        setError(`Unable to load the sample obstacles${detail}. Reload to try again.`);
        setLoading(false);
      }
    }

    void loadSample();
    return () => {
      controller.abort();
      map.remove();
    };
  }, []);

  return (
    <main>
      <div ref={container} className="map" aria-label="Map of sample obstacles in Oslo" />
      {(loading || error) && (
        <div className="map-status" role={error ? 'alert' : 'status'}>
          {error ?? 'Loading map and sample obstacles…'}
        </div>
      )}
    </main>
  );
}
