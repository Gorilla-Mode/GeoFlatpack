import { Popup } from 'maplibre-gl';
import type { Map, MapMouseEvent, StyleSpecification } from 'maplibre-gl';
import type { Feature } from 'geojson';
import { getErrorMessage, getPolygonVertices, getSourceName } from './dataset';
import type { Dataset, SourceLayer } from './dataset';
import { loadEmbeddedIcons } from './styleIcons';

type LoadedLayer = SourceLayer & { dataset: Dataset };

function removeOverlays(map: Map, layerIds: string[], sourceIds: string[], imageIds: string[]) {
  for (const id of [...layerIds].reverse()) if (map.getLayer(id)) map.removeLayer(id);
  for (const id of sourceIds) if (map.getSource(id)) map.removeSource(id);
  for (const id of imageIds) if (map.hasImage(id)) map.removeImage(id);
}

export async function renderMapLayers(map: Map, layers: { layer: LoadedLayer; style: StyleSpecification }[], signal: AbortSignal) {
  const prepared = await Promise.all(layers.map(async entry => ({
    ...entry, icons: await loadEmbeddedIcons(entry.style, signal),
  })));
  signal.throwIfAborted();
  const layerIds: string[] = [];
  const sourceIds: string[] = [];
  const imageIds: string[] = [];
  const owners: Record<string, { layer: LoadedLayer; findFeature: (id: unknown) => Feature | undefined }> = {};
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
    const feature = owner?.findFeature(hit.id);
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

  for (const { layer, style, icons } of prepared) {
    const sourceId = `geoflatpack-${layer.id}`;
    const vertexSourceId = `${sourceId}-vertices`;
    const addedLayers: string[] = [];
    const addedSources: string[] = [];
    const addedImages: string[] = [];
    const runtimeIcons: Record<string, string> = Object.create(null);
    const iconErrors = [...icons.errors];
    try {
      const originalSource = getSourceName(layer.filename);
      const source = style.sources[originalSource];
      if (source.type !== 'geojson') throw new Error('The assigned stylesheet must use a GeoJSON source.');
      const data = layer.dataset.data;
      map.addSource(sourceId, { ...source, data });
      addedSources.push(sourceId);
      for (const [name, image] of icons.images) {
        if (!image) continue;
        const imageId = `${sourceId}-icon-${encodeURIComponent(name)}`;
        try {
          map.addImage(imageId, image, { pixelRatio: 1, sdf: false });
          if (!map.hasImage(imageId)) throw new Error('Unable to register the SVG image.');
          addedImages.push(imageId);
          runtimeIcons[name] = imageId;
        } catch (cause) {
          if (map.hasImage(imageId)) map.removeImage(imageId);
          iconErrors.push(`Icon "${name}": ${getErrorMessage(cause)}`);
        }
      }
      for (const styleLayer of style.layers) {
        if (!('source' in styleLayer) || styleLayer.source !== originalSource) continue;
        const id = `${sourceId}-${styleLayer.id}`;
        let runtimeLayer = { ...styleLayer, id, source: sourceId };
        if (runtimeLayer.type === 'symbol') {
          const metadata = runtimeLayer.metadata;
          if (metadata && typeof metadata === 'object' && 'geoflatpack:placement' in metadata &&
            metadata['geoflatpack:placement'] === 'vertices') {
            if (!addedSources.includes(vertexSourceId)) {
              map.addSource(vertexSourceId, { type: 'geojson', data: getPolygonVertices(data) });
              addedSources.push(vertexSourceId);
              owners[vertexSourceId] = {
                layer,
                findFeature: id => typeof id === 'number' ? data.features[id] : undefined,
              };
            }
            runtimeLayer = { ...runtimeLayer, source: vertexSourceId };
          }
          const icon = runtimeLayer.layout?.['icon-image'];
          if (typeof icon === 'string' && icons.images.has(icon)) {
            const layout = { ...runtimeLayer.layout };
            if (runtimeIcons[icon]) layout['icon-image'] = runtimeIcons[icon];
            else delete layout['icon-image'];
            runtimeLayer = { ...runtimeLayer, layout };
          }
        }
        map.addLayer(runtimeLayer);
        if (!map.getLayer(id)) throw new Error(`Unable to add style layer "${styleLayer.id}".`);
        addedLayers.push(id);
      }
      owners[sourceId] = {
        layer,
        findFeature: id => data.features.find((feature, index) => {
          const featureId = typeof source.promoteId === 'string'
            ? feature.properties?.[source.promoteId]
            : source.generateId ? index : feature.id;
          return featureId === id;
        }),
      };
      sourceIds.push(...addedSources);
      layerIds.push(...addedLayers);
      imageIds.push(...addedImages);
      renderedLayerIds.add(layer.id);
      if (iconErrors.length) errors[layer.id] = `Unable to display some icons for ${layer.label}: ${iconErrors.join(' ')}`;
    } catch (cause) {
      removeOverlays(map, addedLayers, addedSources, addedImages);
      addedSources.forEach(id => { delete owners[id]; });
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
    if (mapActive) removeOverlays(map, layerIds, sourceIds, imageIds);
  }

  return { cleanup, errors, renderedLayerIds };
}
