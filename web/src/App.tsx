import { useEffect, useMemo, useRef, useState } from 'react';
import type { SubmitEvent } from 'react';
import { Map, NavigationControl, Popup, setWorkerUrl } from 'maplibre-gl';
import type { MapMouseEvent, StyleSpecification } from 'maplibre-gl';
import mapWorkerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url';
import HeaderTree from './HeaderTree';
import { createInspection } from './inspection';
import { decodeFgb, getBounds, getSourceName, readDataset } from './dataset';
import type { Dataset, DatasetHeader } from './dataset';
import sampleStyleJson from '../../test_data/sample-obstacles.maplibre.json';

// Use the original filename, not Vite's hashed asset URL, for source naming.
const [[samplePath, sampleUrl]] = Object.entries(import.meta.glob<string>(
  '../../test_data/sample-obstacles.fgb',
  { eager: true, query: '?url', import: 'default' },
));
const sampleStyle = sampleStyleJson as unknown as StyleSpecification;
const inspectors = { header: 'Header', geojson: 'GeoJSON', style: 'Map Style' } as const;
const fileFields = { fgb: ['FlatGeobuf file', '.fgb'], style: ['MapLibre stylesheet', '.json,application/json'] } as const;
setWorkerUrl(mapWorkerUrl);

export default function App() {
  const container = useRef<HTMLDivElement>(null);
  const mapRef = useRef<Map | null>(null);
  const sampleRequest = useRef<AbortController | null>(null);
  const uploadRequest = useRef(0);
  const uploadButton = useRef<HTMLButtonElement>(null);
  const [dataset, setDataset] = useState<(Dataset & { autoCenter?: boolean }) | null>(null);
  const [partialHeader, setPartialHeader] = useState<DatasetHeader | null>(null);
  const [mapReady, setMapReady] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [inspecting, setInspecting] = useState<keyof typeof inspectors | null>(null);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [files, setFiles] = useState<Partial<Record<keyof typeof fileFields, File>>>({});
  const filename = dataset?.filename ?? samplePath.slice(samplePath.lastIndexOf('/') + 1);
  const inspection = useMemo(() => createInspection(dataset ? dataset.header : partialHeader, dataset?.data), [dataset, partialHeader]);
  const views = { header: inspection, geojson: dataset?.data, style: dataset?.style ?? sampleStyle };
  const bounds = useMemo(() => dataset && getBounds(dataset.data), [dataset]);

  function centerMap(duration = 0) {
    if (bounds) mapRef.current?.fitBounds(bounds, { padding: 48, maxZoom: 18, duration, bearing: 0, pitch: 0 });
  }

  async function uploadFiles(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!files.fgb || !files.style || uploading) return;
    const request = ++uploadRequest.current;
    setUploading(true);
    setUploadError(null);
    try {
      const next = await readDataset(files.fgb, files.style);
      if (request !== uploadRequest.current) return;
      sampleRequest.current?.abort();
      setDataset({ ...next, autoCenter: true });
      setError(null);
      setUploadOpen(false);
    } catch (cause) {
      if (request === uploadRequest.current) setUploadError(cause instanceof Error ? cause.message : 'Unable to load these files.');
    } finally {
      if (request === uploadRequest.current) setUploading(false);
    }
  }

  useEffect(() => {
    if (!uploadOpen && !uploading && dataset?.autoCenter) uploadButton.current?.focus();
  }, [uploadOpen, uploading, dataset]);

  useEffect(() => {
    const controller = new AbortController();
    sampleRequest.current = controller;
    setMapReady(false);
    let map: Map;
    try {
      map = new Map({ container: container.current!, style: 'https://tiles.openfreemap.org/styles/liberty', center: sampleStyle.center, zoom: sampleStyle.zoom });
    } catch {
      setError('Unable to start the map. Check that WebGL is available in your browser.');
      return () => { controller.abort(); uploadRequest.current++; };
    }
    mapRef.current = map;
    map.addControl(new NavigationControl(), 'top-right');
    map.on('load', () => setMapReady(true));
    map.on('error', () => setError('Some map resources could not be loaded. Check your connection and reload.'));

    async function loadSample() {
      try {
        const response = await fetch(sampleUrl, { signal: controller.signal });
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const decoded = await decodeFgb(await response.arrayBuffer(), controller.signal, setPartialHeader);
        if (!controller.signal.aborted) setDataset({ filename: samplePath.split('/').pop()!, style: sampleStyle, ...decoded });
      } catch (cause) {
        if (!controller.signal.aborted) setError(`Unable to display the sample obstacles (${cause instanceof Error ? cause.message : cause}). Try loading the files again.`);
      }
    }
    void loadSample();
    return () => {
      controller.abort();
      uploadRequest.current++;
      map.remove();
      mapRef.current = null;
    };
  }, []);

  useEffect(() => {
    const map = mapRef.current;
    if (!map || !mapReady || !dataset) return;
    const { data, style } = dataset;
    const sourceId = getSourceName(dataset.filename);
    const layerIds: string[] = [];
    const popup = new Popup({ className: 'obstacle-popup', closeOnClick: false, maxWidth: 'min(360px, calc(100vw - 48px))' });
    let sourceAdded = false;

    function findFeature({ point: { x, y } }: MapMouseEvent) {
      const hit = map!.queryRenderedFeatures([[x - 4, y - 4], [x + 4, y + 4]], { layers: layerIds })[0];
      return hit && data.features.find(feature => feature.id === hit.id);
    }
    function showFeature(event: MapMouseEvent) {
      popup.remove();
      const feature = findFeature(event);
      if (!feature) return;
      const content = document.createElement('div');
      const title = Object.assign(document.createElement('h2'), { className: 'obstacle-popup-title', textContent: 'Feature properties' });
      const properties = Object.assign(document.createElement('pre'), {
        className: 'obstacle-properties', tabIndex: 0, textContent: JSON.stringify(feature.properties, null, 2),
      });
      properties.setAttribute('aria-label', 'Feature properties');
      content.append(title, properties);
      popup.setLngLat(event.lngLat).setDOMContent(content).addTo(map!);
    }
    function updateCursor(event: MapMouseEvent) {
      map!.getCanvas().style.cursor = findFeature(event) ? 'pointer' : '';
    }
    function resetCursor() { map!.getCanvas().style.cursor = ''; }

    try {
      if (map.getSource(sourceId)) throw new Error(`Source "${sourceId}" conflicts with the basemap.`);
      map.addSource(sourceId, { ...style.sources[sourceId], type: 'geojson', data });
      sourceAdded = true;
      // Only the matching source's layers belong above the street basemap.
      for (const layer of style.layers) {
        if ('source' in layer && layer.source === sourceId) {
          const id = `geoflatpack-${layer.id}`;
          map.addLayer({ ...layer, id, source: sourceId });
          layerIds.push(id);
        }
      }
      if (dataset.autoCenter) centerMap();
      map.on('click', showFeature);
      map.on('mousemove', updateCursor);
      map.getCanvas().addEventListener('mouseleave', resetCursor);
    } catch (cause) {
      setError(`Unable to display ${dataset.filename} (${cause instanceof Error ? cause.message : cause}). Try loading the files again.`);
    }
    return () => {
      popup.remove();
      map.off('click', showFeature);
      map.off('mousemove', updateCursor);
      map.getCanvas().removeEventListener('mouseleave', resetCursor);
      resetCursor();
      if (mapRef.current !== map) return; // Map teardown already removed its layers and source.
      for (const id of layerIds) if (map.getLayer(id)) map.removeLayer(id);
      if (sourceAdded && map.getSource(sourceId)) map.removeSource(sourceId);
    };
  }, [dataset, mapReady]);

  return (
    <main>
      <div ref={container} className="map" aria-label={`Map of ${filename}`} />
      <div className="map-overlay">
        <div className="inspector-controls">
          <button ref={uploadButton} type="button" className="inspector-toggle" disabled={uploading}
            aria-expanded={uploadOpen} aria-controls="upload-panel" onClick={() => {
              setFiles({});
              setUploadError(null);
              setUploadOpen(open => !open);
            }}>
            Upload files
          </button>
          <button type="button" className="inspector-toggle" disabled={!bounds || !mapReady}
            onClick={() => centerMap(window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 0 : 500)}>
            Center on bbox
          </button>
          {Object.entries(inspectors).map(([key, label]) => {
            const view = key as keyof typeof inspectors;
            return (
              <button key={view} type="button" className="inspector-toggle" disabled={!views[view]}
                aria-expanded={inspecting === view} aria-controls="inspector-panel"
                onClick={() => setInspecting(active => active === view ? null : view)}>
                {inspecting === view ? 'Hide' : 'Inspect'} {label}
              </button>
            );
          })}
        </div>
        {uploadOpen && (
          <form id="upload-panel" className="upload-panel" onSubmit={uploadFiles} aria-label="Upload map files">
            <p>Choose a FlatGeobuf file and its MapLibre JSON stylesheet. Files are read locally in your browser.</p>
            <p>The style source must match the FGB filename without its extension.</p>
            <fieldset disabled={uploading}>
              {Object.entries(fileFields).map(([key, [label, accept]]) => (
                <label key={key}>
                  {label}
                  <input type="file" accept={accept} required onChange={event => {
                    const file = event.target.files?.[0];
                    setFiles(current => ({ ...current, [key]: file }));
                    setUploadError(null);
                  }} />
                </label>
              ))}
              <button type="submit" className="inspector-toggle" disabled={!files.fgb || !files.style}>
                {uploading ? 'Reading files…' : 'Display on map'}
              </button>
            </fieldset>
            {uploading && <p role="status">Reading files…</p>}
            {uploadError && <p role="alert">{uploadError}</p>}
          </form>
        )}
        {(!dataset || !mapReady || error) && (
          <div className="map-status" role={error ? 'alert' : 'status'}>
            {error ?? `Loading map and ${filename}…`}
          </div>
        )}
        {inspecting && views[inspecting] && (
          <section key={inspecting} id="inspector-panel" className="inspector-panel" tabIndex={0}
            aria-label={inspecting === 'header' ? 'Parsed header and features' : inspecting === 'geojson' ? 'Decoded GeoJSON' : 'Map style JSON source'}>
            {inspecting === 'header' ? (
              <>
                <h2 className="header-panel-title">Parsed header and features</h2>
                <div className="header-tree"><HeaderTree value={views.header} /></div>
              </>
            ) : <pre className="inspector-json">{JSON.stringify(views[inspecting], null, 2)}</pre>}
          </section>
        )}
      </div>
    </main>
  );
}
