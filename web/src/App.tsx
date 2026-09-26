import { useEffect, useRef, useState } from 'react';
import { Map, NavigationControl, Popup, setWorkerUrl } from 'maplibre-gl';
import type { MapMouseEvent, StyleSpecification } from 'maplibre-gl';
import mapWorkerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url';
import { deserialize } from 'flatgeobuf/lib/mjs/geojson.js';
import type { FeatureCollection } from 'geojson';
import sampleStyleJson from '../../test_data/sample-obstacles.maplibre.json';
import sampleUrl from '../../test_data/sample-obstacles.fgb?url';

// JSON imports widen literal types and coordinate tuples.
const sampleStyle = sampleStyleJson as unknown as StyleSpecification;
const sourceId = 'geoflatpack-obstacles';

type Bounds = [[number, number], [number, number]];

function getFeatureCollectionBounds(data: FeatureCollection): Bounds | null {
  let west = Infinity;
  let south = Infinity;
  let east = -Infinity;
  let north = -Infinity;

  function visit(value: unknown): void {
    if (!Array.isArray(value)) return;
    if (value.length >= 2 && typeof value[0] === 'number' && typeof value[1] === 'number') {
      const [longitude, latitude] = value;
      if (Number.isFinite(longitude) && Number.isFinite(latitude)) {
        west = Math.min(west, longitude);
        south = Math.min(south, latitude);
        east = Math.max(east, longitude);
        north = Math.max(north, latitude);
      }
      return;
    }
    for (const child of value) visit(child);
  }

  function visitGeometry(geometry: unknown): void {
    if (!geometry || typeof geometry !== 'object') return;
    const candidate = geometry as { type?: string; coordinates?: unknown; geometries?: unknown[] };
    if (candidate.type === 'GeometryCollection') {
      candidate.geometries?.forEach(visitGeometry);
    } else {
      visit(candidate.coordinates);
    }
  }

  for (const feature of data.features) visitGeometry(feature.geometry);
  return Number.isFinite(west) ? [[west, south], [east, north]] : null;
}

// Let Vite bundle the worker and its imports for development and production.
setWorkerUrl(mapWorkerUrl);

export default function App() {
  const container = useRef<HTMLDivElement>(null);
  const mapRef = useRef<Map | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [geojson, setGeojson] = useState<FeatureCollection | null>(null);
  const [bounds, setBounds] = useState<Bounds | null>(null);
  const [inspecting, setInspecting] = useState<'geojson' | 'style' | null>(null);

  useEffect(() => {
    if (!container.current) return;

    const controller = new AbortController();
    let map: Map;
    setLoading(true);
    setError(null);
    setGeojson(null);
    setBounds(null);

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
    mapRef.current = map;

    map.addControl(new NavigationControl(), 'top-right');
    const popup = new Popup({
      className: 'obstacle-popup',
      closeOnClick: false,
      maxWidth: 'min(360px, calc(100vw - 48px))',
    });
    let removeObstacleInteractions = () => {};
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
        if (controller.signal.aborted) return;
        setGeojson(data);

        await ready;
        if (controller.signal.aborted) return;

        map.addSource(sourceId, { type: 'geojson', data });
        const obstacleLayerIds: string[] = [];
        // The sample background would hide the street basemap.
        for (const layer of sampleStyle.layers) {
          if ('source' in layer && layer.source === 'obstacles') {
            map.addLayer({ ...layer, id: `geoflatpack-${layer.id}`, source: sourceId });
            obstacleLayerIds.push(`geoflatpack-${layer.id}`);
          }
        }
        const dataBounds = getFeatureCollectionBounds(data);
        if (obstacleLayerIds.length > 0 && dataBounds) setBounds(dataBounds);

        function findObstacle(event: MapMouseEvent) {
          const { x, y } = event.point;
          const hit = map.queryRenderedFeatures(
            [[x - 4, y - 4], [x + 4, y + 4]],
            { layers: obstacleLayerIds },
          )[0];
          return hit && data.features.find((feature) => feature.id === hit.id);
        }

        function showObstacle(event: MapMouseEvent) {
          popup.remove();
          const feature = findObstacle(event);
          if (!feature) return;

          const content = document.createElement('div');
          const title = document.createElement('h2');
          title.className = 'obstacle-popup-title';
          title.textContent = 'Obstacle properties';
          const properties = document.createElement('pre');
          properties.className = 'obstacle-properties';
          properties.tabIndex = 0;
          properties.setAttribute('aria-label', 'Obstacle properties');
          properties.textContent = JSON.stringify(feature.properties, null, 2);
          content.append(title, properties);
          popup.setLngLat(event.lngLat).setDOMContent(content).addTo(map);
        }

        function updateCursor(event: MapMouseEvent) {
          map.getCanvas().style.cursor = findObstacle(event) ? 'pointer' : '';
        }

        function resetCursor() {
          map.getCanvas().style.cursor = '';
        }

        map.on('click', showObstacle);
        map.on('mousemove', updateCursor);
        map.getCanvas().addEventListener('mouseleave', resetCursor);
        removeObstacleInteractions = () => {
          map.off('click', showObstacle);
          map.off('mousemove', updateCursor);
          map.getCanvas().removeEventListener('mouseleave', resetCursor);
          resetCursor();
        };
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
      removeObstacleInteractions();
      popup.remove();
      map.remove();
      if (mapRef.current === map) mapRef.current = null;
    };
  }, []);

  return (
    <main>
      <div ref={container} className="map" aria-label="Map of sample obstacles in Oslo" />
      <div className="map-overlay">
        <div className="inspector-controls">
          <button
            type="button"
            className="inspector-toggle"
            disabled={bounds === null || mapRef.current === null}
            onClick={() => {
              const map = mapRef.current;
              if (!map || !bounds) return;
              const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
              map.fitBounds(bounds, {
                padding: 48,
                maxZoom: 18,
                duration: reducedMotion ? 0 : 500,
                bearing: 0,
                pitch: 0,
              });
            }}
          >
            Center on bbox
          </button>
          <button
            type="button"
            className="inspector-toggle"
            disabled={geojson === null}
            aria-expanded={inspecting === 'geojson'}
            aria-controls="inspector-panel"
            onClick={() => setInspecting((active) => active === 'geojson' ? null : 'geojson')}
          >
            {inspecting === 'geojson' ? 'Hide GeoJSON' : 'Inspect GeoJSON'}
          </button>
          <button
            type="button"
            className="inspector-toggle"
            aria-expanded={inspecting === 'style'}
            aria-controls="inspector-panel"
            onClick={() => setInspecting((active) => active === 'style' ? null : 'style')}
          >
            {inspecting === 'style' ? 'Hide Map Style' : 'Inspect Map Style'}
          </button>
        </div>
        {(loading || error) && (
          <div className="map-status" role={error ? 'alert' : 'status'}>
            {error ?? 'Loading map and sample obstacles…'}
          </div>
        )}
        {(inspecting === 'style' || (inspecting === 'geojson' && geojson !== null)) && (
          <pre
            key={inspecting}
            id="inspector-panel"
            className="inspector-panel"
            tabIndex={0}
            aria-label={inspecting === 'geojson' ? 'Decoded GeoJSON' : 'Map style JSON source'}
          >
            {JSON.stringify(inspecting === 'geojson' ? geojson : sampleStyleJson, null, 2)}
          </pre>
        )}
      </div>
    </main>
  );
}
