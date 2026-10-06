import { Map, setWorkerUrl, AttributionControl } from 'maplibre-gl';
import type { StyleSpecification, ErrorEvent } from 'maplibre-gl';
import workerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url';
import 'maplibre-gl/dist/maplibre-gl.css';
import { getBounds, loadEmbeddedIcons } from './data';
import type { FeatureCollection } from 'geojson';
import './style.css';

setWorkerUrl(workerUrl);
let current: { map: Map; abort: AbortController; overlay: StyleSpecification } | undefined;

type PreviewRequest = { style: StyleSpecification; data: FeatureCollection; basemap: string };
declare global {
  interface Window {
    renderPreview: (request: PreviewRequest) => Promise<{ warning: string; error: string }>;
    clearPreviewBasemap: () => Promise<void>;
  }
}

function removeBasemap(map: Map, overlay: StyleSpecification): void {
  for (const layer of [...(map.getStyle().layers ?? [])].reverse()) {
    if (layer.type !== 'background' && !overlay.layers.some(item => item.id === layer.id)) map.removeLayer(layer.id);
  }
  for (const id of Object.keys(map.getStyle().sources)) if (!Object.hasOwn(overlay.sources, id)) map.removeSource(id);
  const background = map.getStyle().layers?.find(layer => layer.type === 'background');
  if (background) map.setPaintProperty(background.id, 'background-color', '#eeeeee');
}

// The browser also reports failed HTTP resources that MapLibre can silently
// replace, such as missing glyphs drawn using a local fallback font.
window.clearPreviewBasemap = async () => {
  if (!current) return;
  removeBasemap(current.map, current.overlay);
  await settled(current.map, current.abort.signal, 1000);
};

function settled(map: Map, signal: AbortSignal, timeout: number): Promise<void> {
  signal.throwIfAborted();
  return new Promise((resolve, reject) => {
    const finish = (error?: unknown) => {
      clearTimeout(timer);
      map.off('idle', idle);
      signal.removeEventListener('abort', aborted);
      if (error) reject(error); else resolve();
    };
    const idle = () => finish();
    const aborted = () => finish(signal.reason);
    const timer = setTimeout(() => finish(new Error('Map resources timed out')), timeout);
    map.once('idle', idle);
    signal.addEventListener('abort', aborted, { once: true });
    map.triggerRepaint();
  });
}

window.renderPreview = async request => {
  current?.abort.abort();
  current?.map.remove();
  current = undefined;
  const abort = new AbortController();
  const signal = abort.signal;
  let map: Map;
  try {
    map = new Map({
      container: 'map', interactive: false, attributionControl: false,
      style: { version: 8, sources: {}, layers: [{ id: 'background', type: 'background', paint: { 'background-color': '#eeeeee' } }] },
      bearing: 0, pitch: 0, fadeDuration: 0,
      canvasContextAttributes: { antialias: true, preserveDrawingBuffer: true },
    });
  } catch (error) {
    return { warning: '', error: `WebGL unavailable: ${String(error)}` };
  }
  current = { map, abort, overlay: request.style };
  let warning = '';
  try {
    await settled(map, signal, 1500);
    if (request.basemap) {
      let failed = false;
      const resourceError = () => { failed = true; };
      map.on('error', resourceError);
      try {
        const response = await fetch(request.basemap, { signal: AbortSignal.any([signal, AbortSignal.timeout(3000)]) });
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const basemap = await response.json() as StyleSpecification;
        map.setStyle(basemap);
        await settled(map, signal, 3500);
        if (failed) throw new Error('Basemap resources unavailable');
        map.addControl(new AttributionControl({ compact: false }), 'bottom-right');
      } catch (error) {
        signal.throwIfAborted();
        warning = 'Basemap unavailable';
        map.setStyle({ version: 8, sources: {}, layers: [{ id: 'background', type: 'background', paint: { 'background-color': '#eeeeee' } }] });
        await settled(map, signal, 1000);
      } finally {
        map.off('error', resourceError);
      }
    }
    signal.throwIfAborted();
    let overlayError = '';
    let installingOverlay = true;
    const errorListener = (event: ErrorEvent & { sourceId?: string }) => {
      // Glyph/sprite failures can arrive without a source ID after fitting.
      // Our overlay uses local GeoJSON and decoded SVGs, with no remote fonts.
      if (installingOverlay || !request.basemap || (event.sourceId && Object.hasOwn(request.style.sources, event.sourceId))) {
        overlayError = event.error.message;
      } else warning = 'Basemap unavailable';
    };
    map.on('error', errorListener);
    try {
      const icons = await loadEmbeddedIcons(request.style, signal);
      for (const [name, icon] of icons) map.addImage(name, icon, { sdf: false });
      for (const [id, source] of Object.entries(request.style.sources)) {
        if (source.type !== 'geojson') throw new Error('Preview requires a GeoJSON source');
        map.addSource(id, { ...source, data: request.data });
      }
      for (const layer of request.style.layers) if (layer.type !== 'background') map.addLayer(layer);
      if (overlayError) throw new Error(overlayError);
      installingOverlay = false;
      const bounds = getBounds(request.data);
      if (!bounds) throw new Error('No geometry to preview');
      bounds[0][1] = Math.max(-85.051129, Math.min(85.051129, bounds[0][1]));
      bounds[1][1] = Math.max(-85.051129, Math.min(85.051129, bounds[1][1]));
      const padding = Math.min(32, map.getCanvas().height / 5, map.getCanvas().width / 5);
      const points = request.data.features.every(({ geometry }) => geometry?.type === 'Point' || geometry?.type === 'MultiPoint');
      const maxZoom = points ? 16 : 22;
      map.fitBounds(bounds, { padding, maxZoom, duration: 0 });
      try { await settled(map, signal, 2000); }
      catch (error) {
        signal.throwIfAborted();
        if (!request.basemap || overlayError) throw error;
        warning = 'Basemap unavailable';
      }
    } finally {
      map.off('error', errorListener);
    }
    if (overlayError) throw new Error(overlayError);
    if (warning && request.basemap) {
      // Remove incomplete basemap tiles while keeping the styled overlay.
      removeBasemap(map, request.style);
      await settled(map, signal, 1000);
    }
    return { warning, error: '' };
  } catch (error) {
    signal.throwIfAborted();
    return { warning, error: error instanceof Error ? error.message : String(error) };
  }
};
