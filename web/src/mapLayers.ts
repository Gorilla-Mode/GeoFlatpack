import { Popup } from 'maplibre-gl';
import type { Map, MapMouseEvent, StyleSpecification } from 'maplibre-gl';
import { getErrorMessage, getSourceName } from './dataset';
import type { Dataset, SourceLayer } from './dataset';

type LoadedLayer = SourceLayer & { dataset: Dataset };

function removeOverlays(map: Map, layerIds: string[], sourceIds: string[]) {
  for (const id of [...layerIds].reverse()) if (map.getLayer(id)) map.removeLayer(id);
  for (const id of sourceIds) if (map.getSource(id)) map.removeSource(id);
}

export function renderMapLayers(map: Map, layers: { layer: LoadedLayer; style: StyleSpecification }[]) {
  const layerIds: string[] = [];
  const sourceIds: string[] = [];
  const owners: Record<string, { layer: LoadedLayer; featureId: (index: number) => unknown }> = {};
  const errors: Record<string, string> = {};
  const renderedLayerIds = new Set<string>();
  const popup = new Popup({ className: 'obstacle-popup', closeOnClick: false, maxWidth: 'min(360px, calc(100vw - 48px))' });
  const canvas = map.getCanvas();

  function findFeature({ point: { x, y } }: MapMouseEvent) {
    if (!layerIds.length) return;
    const hit = map.queryRenderedFeatures([[x - 4, y - 4], [x + 4, y + 4]], { layers: layerIds })[0];
    if (!hit) return;
    const owner = owners[hit.source];
    // Resolve the original feature so popups retain nested properties and value types.
    const feature = owner?.layer.dataset.data.features.find((_, index) => owner.featureId(index) === hit.id);
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
    popup.setLngLat(event.lngLat).setDOMContent(content).addTo(map);
  }

  function updateCursor(event: MapMouseEvent) { canvas.style.cursor = findFeature(event) ? 'pointer' : ''; }
  function resetCursor() { canvas.style.cursor = ''; }

  for (const { layer, style } of layers) {
    const sourceId = `geoflatpack-${layer.id}`;
    const addedLayers: string[] = [];
    let sourceAdded = false;
    try {
      const originalSource = getSourceName(layer.filename);
      const source = style.sources[originalSource];
      if (source.type !== 'geojson') throw new Error('The assigned stylesheet must use a GeoJSON source.');
      const data = layer.dataset.data;
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
      renderedLayerIds.add(layer.id);
    } catch (cause) {
      removeOverlays(map, addedLayers, sourceAdded ? [sourceId] : []);
      errors[layer.id] = `Unable to display ${layer.label}: ${getErrorMessage(cause)}`;
    }
  }

  map.on('click', showFeature);
  map.on('mousemove', updateCursor);
  canvas.addEventListener('mouseleave', resetCursor);

  // App may have removed the map before React runs the overlay effect's cleanup.
  function cleanup(mapActive = true) {
    popup.remove();
    map.off('click', showFeature);
    map.off('mousemove', updateCursor);
    canvas.removeEventListener('mouseleave', resetCursor);
    resetCursor();
    if (mapActive) removeOverlays(map, layerIds, sourceIds);
  }

  return { cleanup, errors, renderedLayerIds };
}
