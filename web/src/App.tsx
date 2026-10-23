import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { SubmitEvent } from 'react';
import { Map, NavigationControl, setWorkerUrl } from 'maplibre-gl';
import type { StyleSpecification } from 'maplibre-gl';
import mapWorkerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url';
import HeaderTree from './HeaderTree';
import ObjectDetails from './ObjectDetails';
import FeatureInspection from './FeatureInspection';
import { featureTitle } from './featureProperties';
import LayerOrderControl from './LayerOrderControl';
import type { LayerPlacement } from './LayerOrderControl';
import { createInspection } from './inspection';
import { decodeFgb, getBounds, getErrorMessage, getStyleMismatch, readDataset, readStylesheet } from './dataset';
import type { Dataset, SourceLayer, Stylesheet } from './dataset';
import { renderMapLayers } from './mapLayers';
import type { FeatureSelection } from './mapLayers';
import sampleUrl from '../../test_data/sample-obstacles.fgb?url';
import sampleStyleJson from '../../test_data/sample-obstacles.maplibre.json';
import stationUrl from '../../test_data/Samfunnssikkerhet_0000_Norge_25833_Brannstasjoner_GML.fgb?url';
import stationStyleJson from '../../test_data/Samfunnssikkerhet_0000_Norge_25833_Brannstasjoner_GML.gen.maplibre.json';
import skytefeltAreaUrl from '../../test_data/Forurensning_0000_Norge_3035_Skytefelt_GML.SkyteOg_vingsfelt.fgb?url';
import skytefeltBoundaryUrl from '../../test_data/Forurensning_0000_Norge_3035_Skytefelt_GML.Skytefeltgrense.fgb?url';
import skytefeltStyleJson from '../../test_data/Forurensning_0000_Norge_3035_Skytefelt_GML.gen.maplibre.json';

const sampleStyle = sampleStyleJson as unknown as StyleSpecification;
const bundledStyles: Stylesheet[] = [
  { id: 'obstacles', label: 'Sample obstacles', style: sampleStyle },
  { id: 'stations', label: 'Brannstasjoner', style: stationStyleJson as unknown as StyleSpecification },
  { id: 'skytefelt', label: 'Skytefelt (shared)', style: skytefeltStyleJson as unknown as StyleSpecification },
];
// Use the original filenames, not Vite's hashed asset URLs, for style matching.
const bundledSources: SourceLayer[] = [
  { id: 'obstacles', label: 'Sample obstacles', filename: 'sample-obstacles.fgb', url: sampleUrl, styleId: 'obstacles', visible: false },
  {
    id: 'stations', label: 'Brannstasjoner',
    filename: 'Samfunnssikkerhet_0000_Norge_25833_Brannstasjoner_GML.fgb',
    url: stationUrl, styleId: 'stations', visible: false,
  },
  {
    id: 'skytefelt-area', label: 'Skytefelt areas',
    filename: 'Forurensning_0000_Norge_3035_Skytefelt_GML.SkyteOg_vingsfelt.fgb',
    url: skytefeltAreaUrl, styleId: 'skytefelt', visible: true,
  },
  {
    id: 'skytefelt-boundary', label: 'Skytefelt boundaries',
    filename: 'Forurensning_0000_Norge_3035_Skytefelt_GML.Skytefeltgrense.fgb',
    url: skytefeltBoundaryUrl, styleId: 'skytefelt', visible: false,
  },
];
const initialSource = bundledSources.find(layer => layer.visible)!;
const initialStyle = bundledStyles.find(style => style.id === initialSource.styleId)!.style;
const inspectors = [
  { key: 'header', label: 'Header', title: 'Parsed header and features' },
  { key: 'geojson', label: 'GeoJSON', title: 'Decoded GeoJSON' },
  { key: 'style', label: 'Map Style', title: 'Map style JSON source' },
] as const;
type ActivePanel = 'object' | 'upload' | 'source' | (typeof inspectors)[number]['key'] | null;
const fitOptions = { padding: 48, maxZoom: 18, duration: 0, bearing: 0, pitch: 0 };

function layerBounds(layers: SourceLayer[]) {
  return getBounds({ type: 'FeatureCollection', features: layers.flatMap(layer => layer.dataset?.data.features ?? []) });
}

function LayerStatus({ layer, displayError, onRetry, compact = false }: {
  layer: SourceLayer;
  displayError?: string;
  onRetry: () => void;
  compact?: boolean;
}) {
  const error = layer.error || displayError;
  return (
    <>
      {layer.loading && !(compact && error) && <p role="status">Loading {layer.label}…</p>}
      {error && <p role="alert">{error}</p>}
      {layer.error && (
        <button type="button" className={`inspector-toggle${compact ? ' source-retry' : ''}`} onClick={onRetry}>
          Retry {layer.label}
        </button>
      )}
    </>
  );
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
  const panelHeading = useRef<HTMLHeadingElement>(null);
  const panelReturn = useRef<HTMLElement | null>(null);
  const objectReturn = useRef<HTMLElement | null>(null);
  const focusPanel = useRef(false);
  const [selection, setSelection] = useState<FeatureSelection | null>(null);
  const [activePanel, setActivePanel] = useState<ActivePanel>(null);
  const [library, setLibrary] = useState({ layers: bundledSources, styles: bundledStyles });
  const [selectedId, setSelectedId] = useState(initialSource.id);
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
  const inspection = useMemo(() => createInspection(selected.dataset?.header ?? selected.partialHeader ?? null), [selected]);
  const views = { header: inspection, geojson: selected.dataset?.data, style: selectedStyle.style };
  const visibleLayers = useMemo(() => layers.filter((layer): layer is SourceLayer & { dataset: Dataset } => layer.visible && !!layer.dataset), [layers]);
  const bounds = useMemo(() => layerBounds(visibleLayers), [visibleLayers]);
  const pendingLayers = layers.filter(layer => layer.loading || layer.error || displayErrors[layer.id]);
  const objectLayer = layers.find(layer => layer.id === selection?.layerId);
  const objectFeature = selection && objectLayer?.dataset?.data.features[selection.featureIndex];
  const objectTitle = objectFeature && selection ? featureTitle(objectFeature, selection.featureIndex) : '';
  const panelTitle = activePanel === 'object' ? objectTitle : sourceOpen ? 'Layers' : uploadOpen ? 'Add local files' : inspecting?.label ?? '';

  const selectFromMap = useCallback((next: FeatureSelection | null) => {
    if (next) {
      focusPanel.current = false;
      objectReturn.current = mapRef.current?.getCanvas() ?? null;
      panelReturn.current = objectReturn.current;
    }
    setSelection(next);
    setActivePanel(current => next ? 'object' : current === 'object' ? null : current);
  }, []);

  function closePanel() {
    if (activePanel === 'object') setSelection(null);
    setActivePanel(null);
    const target = panelReturn.current;
    const usable = target?.isConnected && !(target instanceof HTMLButtonElement && target.disabled);
    (usable ? target : sourceButton.current)?.focus();
  }

  function openPanel(panel: Exclude<ActivePanel, null>, trigger?: HTMLElement | null) {
    if (panel === 'object') {
      if (trigger) objectReturn.current = trigger;
      panelReturn.current = objectReturn.current;
    } else panelReturn.current = trigger ?? document.getElementById(`${panel}-trigger`);
    focusPanel.current = true;
    setActivePanel(panel);
  }

  function updateLayer(id: string, changes: Partial<SourceLayer>) {
    if (changes.dataset || changes.visible === false || changes.styleId) {
      setSelection(current => current?.layerId === id ? null : current);
    }
    setLibrary(current => ({ ...current, layers: current.layers.map(layer => layer.id === id ? { ...layer, ...changes } : layer) }));
  }

  function moveLayer(layerId: string, targetId: string, placement: LayerPlacement) {
    setLibrary(current => {
      if (layerId === targetId) return current;
      const ordered = [...current.layers].reverse();
      const from = ordered.findIndex(layer => layer.id === layerId);
      if (from < 0 || !ordered.some(layer => layer.id === targetId)) return current;
      const [moved] = ordered.splice(from, 1);
      const target = ordered.findIndex(layer => layer.id === targetId);
      const to = target + (placement === 'after' ? 1 : 0);
      if (from === to) return current;
      ordered.splice(to, 0, moved);
      return { ...current, layers: ordered.reverse() };
    });
  }

  function togglePanel(panel: Exclude<ActivePanel, null>) {
    if (activePanel === panel) closePanel();
    else openPanel(panel);
  }

  function centerMap(duration = 0) {
    if (bounds) mapRef.current?.fitBounds(bounds, { ...fitOptions, duration });
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
        updateLayer(source.id, { loading: false, error: `Unable to load ${source.label}: ${getErrorMessage(cause)}` });
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
      openPanel('source');
    } catch (cause) {
      if (request === uploadRequest.current) setUploadError(getErrorMessage(cause, 'Unable to load these files.'));
    } finally {
      if (request === uploadRequest.current) setUploading(false);
    }
  }

  useEffect(() => {
    if (activePanel && focusPanel.current) {
      panelHeading.current?.focus({ preventScroll: true });
      const panel = panelHeading.current?.closest<HTMLElement>('.workspace-panel');
      if (window.matchMedia('(min-width: 1024px)').matches) panel?.scrollTo(0, 0);
      else panelHeading.current?.scrollIntoView({ block: 'start' });
      focusPanel.current = false;
    }
  }, [activePanel, selectedId, selection]);

  useEffect(() => {
    if (activePanel === 'object' && !objectFeature) setActivePanel(null);
  }, [activePanel, objectFeature]);

  useEffect(() => {
    function cancelRequests() {
      Object.values(sourceRequests.current).forEach(controller => controller.abort());
      sourceRequests.current = {};
      uploadRequest.current++;
    }
    setMapReady(false);
    let map: Map;
    try {
      map = new Map({ container: container.current!, style: 'https://tiles.openfreemap.org/styles/liberty', center: initialStyle.center, zoom: initialStyle.zoom });
    } catch {
      setError('Unable to start the map. Check that WebGL is available in your browser.');
      return cancelRequests;
    }
    mapRef.current = map;
    const resize = new ResizeObserver(() => map.resize());
    resize.observe(container.current!);
    map.addControl(new NavigationControl(), 'top-right');
    map.on('load', () => setMapReady(true));
    map.on('error', () => setError('Some map resources could not be loaded. Check your connection and reload.'));

    void loadSource(initialSource);
    return () => {
      cancelRequests();
      resize.disconnect();
      map.remove();
      mapRef.current = null;
    };
  }, []);

  useEffect(() => {
    const map = mapRef.current;
    if (!map || !mapReady) return;
    const controller = new AbortController();
    let cleanup: ((mapActive: boolean) => void) | undefined;
    void renderMapLayers(map, visibleLayers.map(layer => ({
      layer, style: styles.find(style => style.id === layer.styleId)!.style,
    })), controller.signal, selectFromMap).then(result => {
      if (controller.signal.aborted || mapRef.current !== map) {
        result.cleanup(mapRef.current === map);
        return;
      }
      cleanup = result.cleanup;
      setDisplayErrors(result.errors);
      const newlyVisible = visibleLayers.filter(layer => result.renderedLayerIds.has(layer.id) && pendingCenters.current.has(layer.id));
      const nextBounds = layerBounds(newlyVisible);
      newlyVisible.forEach(layer => pendingCenters.current.delete(layer.id));
      if (nextBounds) map.fitBounds(nextBounds, fitOptions);
    }).catch(cause => {
      if (controller.signal.aborted || mapRef.current !== map) return;
      setDisplayErrors(Object.fromEntries(visibleLayers.map(layer => [
        layer.id, `Unable to display ${layer.label}: ${getErrorMessage(cause)}`,
      ])));
    });
    return () => {
      controller.abort();
      cleanup?.(mapRef.current === map);
    };
  }, [visibleLayers, styles, mapReady, selectFromMap]);

  return (
    <main className="app-shell">
      <a className="skip-link" href={activePanel ? '#workspace-panel' : '#map-view'}>{activePanel ? 'Skip to details' : 'Skip to map'}</a>
      <header className="app-header">
        <h1 className="brand"><span className="brand-mark" aria-hidden="true">◧</span>GeoFlatpack</h1>
        <nav className="primary-actions" aria-label="Map tools">
          <button ref={uploadButton} id="upload-trigger" type="button" className="tool-button" disabled={uploading}
            aria-expanded={uploadOpen} aria-controls="workspace-panel" onClick={() => {
              if (!uploadOpen) { setFgbFiles([]); setStyleFile(null); setUploadError(null); }
              togglePanel('upload');
            }}>Add files</button>
          <button ref={sourceButton} id="source-trigger" type="button" className="tool-button" disabled={uploading}
            aria-expanded={sourceOpen} aria-controls="workspace-panel" onClick={() => togglePanel('source')}>Layers</button>
          <div className="inspection-actions" role="group" aria-label="Inspect selected source">
            {inspectors.map(view => <button key={view.key} id={`${view.key}-trigger`} type="button" className="tool-button"
              disabled={!views[view.key]} aria-expanded={inspecting === view} aria-controls="workspace-panel"
              onClick={() => togglePanel(view.key)}>{view.label}</button>)}
          </div>
          {objectFeature && <button id="object-trigger" type="button" className="tool-button object-trigger"
            aria-expanded={activePanel === 'object'} aria-controls="workspace-panel"
            onClick={() => togglePanel('object')}>Selected object</button>}
        </nav>
        <button type="button" className="tool-button center-button" disabled={!bounds || !mapReady}
          onClick={() => centerMap(window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 0 : 500)}>
          <span aria-hidden="true">⌖</span> Center on layers
        </button>
      </header>
      <div className="visually-hidden" role="status" aria-atomic="true">
        {objectFeature ? `Selected ${objectTitle} from ${objectLayer?.label}. Use Selected object to read details.` : ''}
      </div>
      {(!mapReady || error || (!sourceOpen && pendingLayers.length > 0)) && <div className="status-strip">
        {(!mapReady || error) && <p role={error ? 'alert' : 'status'}>{error ?? 'Loading map…'}</p>}
        {!sourceOpen && pendingLayers.map(layer => <div key={layer.id}>
          <LayerStatus layer={layer} displayError={displayErrors[layer.id]} onRetry={() => void loadSource(layer)} compact />
        </div>)}
      </div>}
      <div className={`workspace${activePanel ? ' has-panel' : ''}`}>
        <section className="map-area" aria-label="Map workspace">
          <div ref={container} id="map-view" tabIndex={-1} className="map" aria-label={`Map layers: ${visibleLayers.map(layer => layer.label).join(', ') || 'basemap only'}`} />
        </section>
        <aside id="workspace-panel" className={`workspace-panel${activePanel ? '' : ' panel-closed'}`}
          aria-label={activePanel ? panelTitle : 'Details'} tabIndex={-1}
          onKeyDown={event => {
            if (event.key === 'Escape' && activePanel && !event.defaultPrevented) { event.preventDefault(); closePanel(); }
          }}>
          {activePanel && <>
            <div className="panel-heading">
              <div><p className="eyebrow">{activePanel === 'object' ? 'Selected object' : inspecting ? 'Source inspection' : 'Workspace'}</p>
                <h2 ref={panelHeading} tabIndex={-1}>{panelTitle}</h2></div>
              <button type="button" className="close-button" aria-label="Close panel" onClick={closePanel}>×</button>
            </div>
            <div className="panel-scroll">
              {activePanel === 'object' && objectFeature && objectLayer && <ObjectDetails key={`${selection?.layerId}-${selection?.featureIndex}`}
                feature={objectFeature} layerLabel={objectLayer.label} />}
              {sourceOpen && <div id="source-panel" className="source-panel">
                <p className="panel-intro">Choose what appears on the map. Each source keeps its own stylesheet.</p>
                <h3>Sources <span className="count">{layers.filter(layer => layer.visible).length} visible</span></h3>
                {layers.map(layer => <div key={layer.id} className="source-row" data-selected={selectedId === layer.id}>
                  <label className="source-visibility" title={layer.filename}>
                    <input type="checkbox" checked={layer.visible} onChange={event => setVisible(layer, event.target.checked)} />
                    <span>{layer.label}</span>
                  </label>
                  <label className="source-style">Stylesheet
                    <select value={layer.styleId} onChange={event => assignStyle(layer, event.target.value)}>
                      {styles.filter(style => !getStyleMismatch(layer.filename, style.style)).map(style => <option key={style.id} value={style.id}>{style.label}</option>)}
                    </select>
                  </label>
                  <button type="button" className="inspect-source" aria-label={`Inspect ${layer.label}`}
                    onClick={() => { setSelectedId(layer.id); openPanel(layer.dataset || layer.partialHeader ? 'header' : 'style'); }}>
                    Inspect{selectedId === layer.id ? ' · current' : ''}<span aria-hidden="true"> ↗</span>
                  </button>
                  <LayerStatus layer={layer} displayError={displayErrors[layer.id]} onRetry={() => void loadSource(layer)} />
                </div>)}
                <LayerOrderControl layers={layers} displayErrors={displayErrors} onMove={moveLayer} />
              </div>}
              {uploadOpen && <form id="upload-panel" className="upload-panel" onSubmit={uploadFiles} aria-label="Upload map files">
                <p className="panel-intro">Open one or more FlatGeobuf files with a MapLibre JSON stylesheet.</p>
                <p className="local-note">Files stay in your browser.</p>
                <fieldset disabled={uploading}>
                  <label>FlatGeobuf files
                    <input type="file" accept=".fgb" multiple required onChange={event => { setFgbFiles(Array.from(event.target.files ?? [])); setUploadError(null); }} />
                  </label>
                  <label>MapLibre stylesheet
                    <input type="file" accept=".json,application/json" required onChange={event => { setStyleFile(event.target.files?.[0] ?? null); setUploadError(null); }} />
                  </label>
                  <p className="field-help">Each FGB filename without its extension must match a source in the stylesheet. All files in this batch use that stylesheet.</p>
                  <button type="submit" className="primary-button" disabled={!fgbFiles.length || !styleFile}>{uploading ? 'Reading files…' : 'Add layers'}</button>
                </fieldset>
                {uploading && <p role="status">Reading files…</p>}
                {uploadError && <p role="alert">{uploadError}</p>}
              </form>}
              {inspecting && <div id="inspector-panel" className="inspector-panel">
                <label className="inspection-source">Inspecting source
                  <select value={selectedId} onChange={event => setSelectedId(event.target.value)}>
                    {layers.map(layer => <option key={layer.id} value={layer.id}>{layer.label}</option>)}
                  </select>
                </label>
                <p className="panel-intro">{inspecting.title}{inspecting.key === 'style' ? ` · ${selectedStyle.label}` : ''}</p>
                {inspecting.key === 'header' ? <>
                  {views.header ? <HeaderTree value={views.header} initialOpen /> : <p>No header available. Enable this source in Layers to load it.</p>}
                  {selected.dataset && <FeatureInspection key={selected.id} features={selected.dataset.data.features} onSelect={index => {
                    setSelection({ layerId: selected.id, featureIndex: index });
                    openPanel('object', document.getElementById('header-trigger'));
                  }} />}
                </> : views[inspecting.key] ? <pre tabIndex={0} className="inspector-json" aria-label={inspecting.title}>{JSON.stringify(views[inspecting.key], null, 2)}</pre>
                  : <p>No data available. Enable this source in Layers to load it.</p>}
                <LayerStatus layer={selected} displayError={displayErrors[selected.id]} onRetry={() => void loadSource(selected)} />
              </div>}
            </div>
          </>}
        </aside>
      </div>
    </main>
  );
}
