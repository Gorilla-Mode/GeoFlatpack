import { useEffect, useMemo, useRef, useState } from 'react';
import type { SubmitEvent } from 'react';
import { Map, NavigationControl, Popup, setWorkerUrl } from 'maplibre-gl';
import type { MapMouseEvent, StyleSpecification } from 'maplibre-gl';
import mapWorkerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url';
import HeaderTree from './HeaderTree';
import { createInspection } from './inspection';
import { decodeFgb, getBounds, getSourceName, getStyleMismatch, readDataset, readStylesheet } from './dataset';
import type { Dataset, DatasetHeader } from './dataset';
import sampleUrl from '../../test_data/sample-obstacles.fgb?url';
import sampleStyleJson from '../../test_data/sample-obstacles.maplibre.json';
import stationUrl from '../../test_data/Samfunnssikkerhet_0000_Norge_25833_Brannstasjoner_GML.fgb?url';
import stationStyleJson from '../../test_data/Samfunnssikkerhet_0000_Norge_25833_Brannstasjoner_GML.gen.maplibre.json';
import skytefeltAreaUrl from '../../test_data/Forurensning_0000_Norge_3035_Skytefelt_GML.SkyteOg_vingsfelt.fgb?url';
import skytefeltBoundaryUrl from '../../test_data/Forurensning_0000_Norge_3035_Skytefelt_GML.Skytefeltgrense.fgb?url';
import skytefeltStyleJson from '../../test_data/Forurensning_0000_Norge_3035_Skytefelt_GML.gen.maplibre.json';

type Stylesheet = { id: string; label: string; style: StyleSpecification };
type SourceLayer = {
  id: string;
  label: string;
  filename: string;
  styleId: string;
  visible: boolean;
  url?: string;
  dataset?: Dataset;
  partialHeader?: DatasetHeader;
  loading?: boolean;
  error?: string;
};
const sampleStyle = sampleStyleJson as unknown as StyleSpecification;
const bundledStyles: Stylesheet[] = [
  { id: 'obstacles', label: 'Sample obstacles', style: sampleStyle },
  { id: 'stations', label: 'Brannstasjoner', style: stationStyleJson as unknown as StyleSpecification },
  { id: 'skytefelt', label: 'Skytefelt (shared)', style: skytefeltStyleJson as unknown as StyleSpecification },
];
// Use the original filenames, not Vite's hashed asset URLs, for style matching.
const bundledSources: SourceLayer[] = [
  { id: 'obstacles', label: 'Sample obstacles', filename: 'sample-obstacles.fgb', url: sampleUrl, styleId: 'obstacles', visible: true },
  {
    id: 'stations', label: 'Brannstasjoner',
    filename: 'Samfunnssikkerhet_0000_Norge_25833_Brannstasjoner_GML.fgb',
    url: stationUrl, styleId: 'stations', visible: false,
  },
  {
    id: 'skytefelt-area', label: 'Skytefelt areas',
    filename: 'Forurensning_0000_Norge_3035_Skytefelt_GML.SkyteOg_vingsfelt.fgb',
    url: skytefeltAreaUrl, styleId: 'skytefelt', visible: false,
  },
  {
    id: 'skytefelt-boundary', label: 'Skytefelt boundaries',
    filename: 'Forurensning_0000_Norge_3035_Skytefelt_GML.Skytefeltgrense.fgb',
    url: skytefeltBoundaryUrl, styleId: 'skytefelt', visible: false,
  },
];
const inspectors = [
  { key: 'header', label: 'Header', title: 'Parsed header and features' },
  { key: 'geojson', label: 'GeoJSON', title: 'Decoded GeoJSON' },
  { key: 'style', label: 'Map Style', title: 'Map style JSON source' },
] as const;
type ActivePanel = 'upload' | 'source' | (typeof inspectors)[number]['key'] | null;

function layerBounds(layers: SourceLayer[]) {
  return getBounds({ type: 'FeatureCollection', features: layers.flatMap(layer => layer.dataset?.data.features ?? []) });
}

setWorkerUrl(mapWorkerUrl);

export default function App() {
  const container = useRef<HTMLDivElement>(null);
  const mapRef = useRef<Map | null>(null);
  const sourceRequests = useRef<Record<string, AbortController>>({});
  const pendingCenters = useRef(new Set<string>());
  const uploadRequest = useRef(0);
  const nextBatch = useRef(0);
  const uploadButton = useRef<HTMLButtonElement>(null);
  const sourceButton = useRef<HTMLButtonElement>(null);
  const sourcePanel = useRef<HTMLDivElement>(null);
  const inspectorPanel = useRef<HTMLElement>(null);
  const [activePanel, setActivePanel] = useState<ActivePanel>(null);
  const [library, setLibrary] = useState({ layers: bundledSources, styles: bundledStyles });
  const [selectedId, setSelectedId] = useState(bundledSources[0].id);
  const [mapReady, setMapReady] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [displayErrors, setDisplayErrors] = useState<Record<string, string>>({});
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [fgbFiles, setFgbFiles] = useState<File[]>([]);
  const [styleFile, setStyleFile] = useState<File | null>(null);
  const { layers, styles } = library;
  const selected = layers.find(layer => layer.id === selectedId)!;
  const selectedStyle = styles.find(style => style.id === selected.styleId)!;
  const sourceOpen = activePanel === 'source';
  const uploadOpen = activePanel === 'upload';
  const inspecting = inspectors.find(view => view.key === activePanel);
  const inspection = useMemo(() => createInspection(selected.dataset?.header ?? selected.partialHeader ?? null, selected.dataset?.data), [selected]);
  const views = { header: inspection, geojson: selected.dataset?.data, style: selectedStyle.style };
  const visibleLayers = useMemo(() => layers.filter(layer => layer.visible && layer.dataset), [layers]);
  const bounds = useMemo(() => layerBounds(visibleLayers), [visibleLayers]);
  const pendingLayers = layers.filter(layer => layer.loading || layer.error || displayErrors[layer.id]);

  function updateLayer(id: string, changes: Partial<SourceLayer>) {
    setLibrary(current => ({ ...current, layers: current.layers.map(layer => layer.id === id ? { ...layer, ...changes } : layer) }));
  }

  function togglePanel(panel: Exclude<ActivePanel, null>) {
    setActivePanel(active => active === panel ? null : panel);
  }

  function centerMap(duration = 0) {
    if (bounds) mapRef.current?.fitBounds(bounds, { padding: 48, maxZoom: 18, duration, bearing: 0, pitch: 0 });
  }

  async function loadSource(source: SourceLayer, autoCenter = true) {
    if (!source.url) return;
    sourceRequests.current[source.id]?.abort();
    const controller = new AbortController();
    sourceRequests.current[source.id] = controller;
    updateLayer(source.id, { loading: true, error: undefined, partialHeader: undefined });
    try {
      const response = await fetch(source.url, { signal: controller.signal });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const decoded = await decodeFgb(await response.arrayBuffer(), controller.signal, header => {
        if (!controller.signal.aborted) updateLayer(source.id, { partialHeader: header });
      });
      if (controller.signal.aborted) return;
      if (autoCenter) pendingCenters.current.add(source.id);
      updateLayer(source.id, { dataset: { filename: source.filename, ...decoded }, loading: false });
    } catch (cause) {
      if (!controller.signal.aborted) {
        updateLayer(source.id, { loading: false, error: `Unable to load ${source.label}: ${cause instanceof Error ? cause.message : cause}` });
      }
    } finally {
      if (sourceRequests.current[source.id] === controller) delete sourceRequests.current[source.id];
    }
  }

  function setVisible(source: SourceLayer, visible: boolean) {
    if (!visible) {
      sourceRequests.current[source.id]?.abort();
      delete sourceRequests.current[source.id];
      pendingCenters.current.delete(source.id);
      updateLayer(source.id, { visible: false, loading: false, error: undefined });
      return;
    }
    if (source.dataset) pendingCenters.current.add(source.id);
    updateLayer(source.id, { visible: true });
    if (!source.dataset) void loadSource(source);
  }

  function assignStyle(source: SourceLayer, styleId: string) {
    const stylesheet = styles.find(style => style.id === styleId);
    if (!stylesheet || getStyleMismatch(source.filename, stylesheet.style)) return;
    updateLayer(source.id, { styleId });
  }

  async function uploadFiles(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!fgbFiles.length || !styleFile || uploading) return;
    const request = ++uploadRequest.current;
    setUploading(true);
    setUploadError(null);
    try {
      const style = await readStylesheet(styleFile);
      const datasets: Dataset[] = [];
      // Commit only after every file succeeds; existing layers survive a bad batch.
      for (const file of fgbFiles) {
        datasets.push(await readDataset(file, style));
        if (request !== uploadRequest.current) return;
      }
      const batch = ++nextBatch.current;
      const styleId = `upload-style-${batch}`;
      const added = datasets.map((dataset, index): SourceLayer => ({
        id: `upload-${batch}-${index}`, label: `${dataset.filename} (upload ${batch}, ${index + 1})`,
        filename: dataset.filename, styleId, visible: true, dataset,
      }));
      added.forEach(layer => pendingCenters.current.add(layer.id));
      setLibrary(current => ({
        layers: [...current.layers, ...added],
        styles: [...current.styles, { id: styleId, label: `${styleFile.name} (upload ${batch})`, style }],
      }));
      setSelectedId(added[0].id);
      setActivePanel('source');
    } catch (cause) {
      if (request === uploadRequest.current) setUploadError(cause instanceof Error ? cause.message : 'Unable to load these files.');
    } finally {
      if (request === uploadRequest.current) setUploading(false);
    }
  }

  useEffect(() => {
    if (sourceOpen) {
      const panel = sourcePanel.current;
      (panel?.querySelector<HTMLInputElement>('[data-selected="true"] input') ?? panel?.querySelector('input'))?.focus();
    }
  }, [sourceOpen]);

  useEffect(() => {
    if (inspecting) inspectorPanel.current?.focus();
  }, [inspecting, selectedId]);

  useEffect(() => {
    function cancelRequests() {
      Object.values(sourceRequests.current).forEach(controller => controller.abort());
      sourceRequests.current = {};
      uploadRequest.current++;
    }
    setMapReady(false);
    let map: Map;
    try {
      map = new Map({ container: container.current!, style: 'https://tiles.openfreemap.org/styles/liberty', center: sampleStyle.center, zoom: sampleStyle.zoom });
    } catch {
      setError('Unable to start the map. Check that WebGL is available in your browser.');
      return cancelRequests;
    }
    mapRef.current = map;
    map.addControl(new NavigationControl(), 'top-right');
    map.on('load', () => setMapReady(true));
    map.on('error', () => setError('Some map resources could not be loaded. Check your connection and reload.'));

    void loadSource(bundledSources[0], false);
    return () => {
      cancelRequests();
      map.remove();
      mapRef.current = null;
    };
  }, []);

  useEffect(() => {
    const map = mapRef.current;
    if (!map || !mapReady) return;
    const layerIds: string[] = [];
    const sourceIds: string[] = [];
    const owners: Record<string, { layer: SourceLayer; featureId: (index: number) => unknown }> = {};
    const failures: Record<string, string> = {};
    const rendered: SourceLayer[] = [];
    const popup = new Popup({ className: 'obstacle-popup', closeOnClick: false, maxWidth: 'min(360px, calc(100vw - 48px))' });
    const canvas = map.getCanvas();

    function findFeature({ point: { x, y } }: MapMouseEvent) {
      if (!layerIds.length) return;
      const hit = map!.queryRenderedFeatures([[x - 4, y - 4], [x + 4, y + 4]], { layers: layerIds })[0];
      if (!hit) return;
      const owner = owners[hit.source];
      const feature = owner?.layer.dataset?.data.features.find((_, index) => owner.featureId(index) === hit.id);
      return feature ? { feature, layer: owner.layer } : undefined;
    }
    function showFeature(event: MapMouseEvent) {
      popup.remove();
      const found = findFeature(event);
      if (!found) return;
      const content = document.createElement('div');
      const title = Object.assign(document.createElement('h2'), { className: 'obstacle-popup-title', textContent: found.layer.label });
      const properties = Object.assign(document.createElement('pre'), {
        className: 'obstacle-properties', tabIndex: 0, textContent: JSON.stringify(found.feature.properties, null, 2),
      });
      properties.setAttribute('aria-label', 'Feature properties');
      content.append(title, properties);
      popup.setLngLat(event.lngLat).setDOMContent(content).addTo(map!);
    }
    function updateCursor(event: MapMouseEvent) { canvas.style.cursor = findFeature(event) ? 'pointer' : ''; }
    function resetCursor() { canvas.style.cursor = ''; }

    for (const layer of visibleLayers) {
      const sourceId = `geoflatpack-${layer.id}`;
      const addedLayers: string[] = [];
      let sourceAdded = false;
      try {
        const style = styles.find(style => style.id === layer.styleId)!.style;
        const originalSource = getSourceName(layer.filename);
        const source = style.sources[originalSource];
        if (source.type !== 'geojson') throw new Error('The assigned stylesheet must use a GeoJSON source.');
        const data = layer.dataset!.data;
        map.addSource(sourceId, { ...source, data });
        sourceAdded = true;
        for (const styleLayer of style.layers) {
          if (!('source' in styleLayer) || styleLayer.source !== originalSource) continue;
          const id = `${sourceId}-${styleLayer.id}`;
          map.addLayer({ ...styleLayer, id, source: sourceId });
          if (!map.getLayer(id)) throw new Error(`Unable to add style layer "${styleLayer.id}".`);
          addedLayers.push(id);
        }
        owners[sourceId] = {
          layer,
          featureId: index => {
            const feature = data.features[index];
            if (typeof source.promoteId === 'string') return feature.properties?.[source.promoteId];
            return source.generateId ? index : feature.id;
          },
        };
        sourceIds.push(sourceId);
        layerIds.push(...addedLayers);
        rendered.push(layer);
      } catch (cause) {
        for (const id of addedLayers.reverse()) if (map.getLayer(id)) map.removeLayer(id);
        if (sourceAdded && map.getSource(sourceId)) map.removeSource(sourceId);
        failures[layer.id] = `Unable to display ${layer.label}: ${cause instanceof Error ? cause.message : cause}`;
      }
    }
    setDisplayErrors(failures);
    const newlyVisible = rendered.filter(layer => pendingCenters.current.has(layer.id));
    const nextBounds = layerBounds(newlyVisible);
    newlyVisible.forEach(layer => pendingCenters.current.delete(layer.id));
    if (nextBounds) map.fitBounds(nextBounds, { padding: 48, maxZoom: 18, duration: 0, bearing: 0, pitch: 0 });
    map.on('click', showFeature);
    map.on('mousemove', updateCursor);
    canvas.addEventListener('mouseleave', resetCursor);
    return () => {
      popup.remove();
      map.off('click', showFeature);
      map.off('mousemove', updateCursor);
      canvas.removeEventListener('mouseleave', resetCursor);
      resetCursor();
      if (mapRef.current !== map) return;
      for (const id of layerIds.reverse()) if (map.getLayer(id)) map.removeLayer(id);
      for (const id of sourceIds) if (map.getSource(id)) map.removeSource(id);
    };
  }, [visibleLayers, styles, mapReady]);

  return (
    <main>
      <div ref={container} className="map" aria-label={`Map layers: ${visibleLayers.map(layer => layer.label).join(', ') || 'basemap only'}`} />
      <div className="map-overlay" onKeyDown={event => {
        if (event.key === 'Escape' && activePanel) {
          event.preventDefault();
          setActivePanel(null);
          (uploadOpen ? uploadButton : sourceButton).current?.focus();
        }
      }}>
        <div className="map-toolbar">
          <div className="inspector-controls">
            <button ref={uploadButton} type="button" className="inspector-toggle" disabled={uploading}
              aria-expanded={uploadOpen} aria-controls="upload-panel" onClick={() => {
                setFgbFiles([]);
                setStyleFile(null);
                setUploadError(null);
                togglePanel('upload');
              }}>
              Upload files
            </button>
            <button ref={sourceButton} type="button" className="inspector-toggle" disabled={uploading}
              aria-expanded={sourceOpen} aria-controls="source-panel" onClick={() => togglePanel('source')}>
              Select sources
            </button>
            {inspectors.map(view => (
              <button key={view.key} type="button" className="inspector-toggle" disabled={!views[view.key]}
                aria-expanded={inspecting === view} aria-controls="inspector-panel"
                onClick={() => togglePanel(view.key)}>
                {inspecting === view ? 'Hide' : 'Inspect'} {view.label}
              </button>
            ))}
          </div>
          <button type="button" className="inspector-toggle" disabled={!bounds || !mapReady}
            onClick={() => centerMap(window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 0 : 500)}>
            Center on bbox
          </button>
        </div>
        {sourceOpen && (
          <div ref={sourcePanel} id="source-panel" className="source-panel" role="group" aria-label="Select sources">
            <p>Show layers together. Choose a compatible stylesheet for each layer.</p>
            {layers.map(layer => (
              <div key={layer.id} className="source-row" data-selected={selectedId === layer.id}>
                <label className="source-visibility" title={layer.filename}>
                  <input type="checkbox" checked={layer.visible} onChange={event => setVisible(layer, event.target.checked)} />
                  <span>{layer.label}</span>
                </label>
                <label className="source-style">
                  Stylesheet
                  <select value={layer.styleId} onChange={event => assignStyle(layer, event.target.value)}>
                    {styles.filter(style => !getStyleMismatch(layer.filename, style.style)).map(style => (
                      <option key={style.id} value={style.id}>{style.label}</option>
                    ))}
                  </select>
                </label>
                <button type="button" className="inspector-toggle" aria-label={`Inspect ${layer.label}`}
                  onClick={() => {
                    setSelectedId(layer.id);
                    setActivePanel(layer.dataset || layer.partialHeader ? 'header' : 'style');
                  }}>
                  Inspect{selectedId === layer.id ? ' (current)' : ''}
                </button>
                {layer.loading && <p role="status">Loading {layer.label}…</p>}
                {(layer.error || displayErrors[layer.id]) && <p role="alert">{layer.error || displayErrors[layer.id]}</p>}
                {layer.error && <button type="button" className="inspector-toggle" onClick={() => void loadSource(layer)}>Retry {layer.label}</button>}
              </div>
            ))}
          </div>
        )}
        {uploadOpen && (
          <form id="upload-panel" className="upload-panel" onSubmit={uploadFiles} aria-label="Upload map files">
            <p>Choose one or more FlatGeobuf files and one MapLibre JSON stylesheet. All files in this batch use that stylesheet.</p>
            <p>Each FGB filename without its extension must match a source in the stylesheet. Files stay in your browser.</p>
            <fieldset disabled={uploading}>
              <label>
                FlatGeobuf files
                <input type="file" accept=".fgb" multiple required onChange={event => {
                  setFgbFiles(Array.from(event.target.files ?? []));
                  setUploadError(null);
                }} />
              </label>
              <label>
                MapLibre stylesheet
                <input type="file" accept=".json,application/json" required onChange={event => {
                  setStyleFile(event.target.files?.[0] ?? null);
                  setUploadError(null);
                }} />
              </label>
              <button type="submit" className="inspector-toggle" disabled={!fgbFiles.length || !styleFile}>
                {uploading ? 'Reading files…' : 'Add layers'}
              </button>
            </fieldset>
            {uploading && <p role="status">Reading files…</p>}
            {uploadError && <p role="alert">{uploadError}</p>}
          </form>
        )}
        {(!mapReady || error) && <div className="map-status" role={error ? 'alert' : 'status'}>{error ?? 'Loading map…'}</div>}
        {!sourceOpen && pendingLayers.length > 0 && (
          <div className="map-status">
            {pendingLayers.map(layer => (
              <div key={layer.id}>
                <p role={layer.error || displayErrors[layer.id] ? 'alert' : 'status'}>
                  {layer.error || displayErrors[layer.id] || `Loading ${layer.label}…`}
                </p>
                {layer.error && <button type="button" className="inspector-toggle source-retry" onClick={() => void loadSource(layer)}>Retry {layer.label}</button>}
              </div>
            ))}
          </div>
        )}
        {inspecting && views[inspecting.key] && (
          <section ref={inspectorPanel} key={`${inspecting.key}-${selectedId}`} id="inspector-panel" className="inspector-panel" tabIndex={0}
            aria-label={`${inspecting.title}: ${selected.label}`}>
            <h2 className="header-panel-title">{inspecting.title}</h2>
            <p className="inspector-source">{selected.label}{inspecting.key === 'style' && <> — {selectedStyle.label}</>}</p>
            {inspecting.key === 'header'
              ? <div className="header-tree"><HeaderTree value={views.header} /></div>
              : <pre className="inspector-json">{JSON.stringify(views[inspecting.key], null, 2)}</pre>}
          </section>
        )}
      </div>
    </main>
  );
}
