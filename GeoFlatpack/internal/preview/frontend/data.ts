import type { FeatureCollection, Geometry } from 'geojson';
import type { StyleSpecification } from 'maplibre-gl';

export function getBounds(data: FeatureCollection): [[number, number], [number, number]] | null {
  let west = Infinity, south = Infinity, east = -Infinity, north = -Infinity;
  function visit(coordinates: unknown): void {
    if (!Array.isArray(coordinates)) return;
    const [x, y] = coordinates;
    if (typeof x === 'number' && typeof y === 'number') {
      if (Number.isFinite(x) && Number.isFinite(y)) {
        west = Math.min(west, x); south = Math.min(south, y);
        east = Math.max(east, x); north = Math.max(north, y);
      }
    } else coordinates.forEach(visit);
  }
  function geometry(value: Geometry | null): void {
    if (value?.type === 'GeometryCollection') value.geometries.forEach(geometry);
    else if (value) visit(value.coordinates);
  }
  data.features.forEach(feature => geometry(feature.geometry));
  return Number.isFinite(west) ? [[west, south], [east, north]] : null;
}

// Exported styles carry original SVGs in metadata rather than an SDF sprite.
// Decode those exact icons so the preview matches the saved stylesheet.
export async function loadEmbeddedIcons(style: StyleSpecification, signal: AbortSignal): Promise<Map<string, HTMLImageElement>> {
  const images = new Map<string, HTMLImageElement>();
  const metadata = style.metadata as Record<string, unknown> | undefined;
  const icons = metadata?.['geoflatpack:icons'];
  if (icons === undefined) return images;
  if (!icons || typeof icons !== 'object' || Array.isArray(icons)) throw new Error('Invalid embedded SVG icons');
  await Promise.all(Object.entries(icons).map(async ([name, svg]) => {
    signal.throwIfAborted();
    if (typeof svg !== 'string' || !svg.trim()) throw new Error(`Invalid SVG: ${name}`);
    const image = new Image();
    const url = URL.createObjectURL(new Blob([svg], { type: 'image/svg+xml' }));
    let abort = () => {};
    try {
      await new Promise<void>((resolve, reject) => {
        abort = () => { image.removeAttribute('src'); reject(signal.reason); };
        image.onload = () => {
          if (image.naturalWidth && image.naturalHeight) resolve();
          else reject(new Error(`SVG has no dimensions: ${name}`));
        };
        image.onerror = () => reject(new Error(`Unable to decode SVG: ${name}`));
        signal.addEventListener('abort', abort, { once: true });
        image.src = url;
      });
      signal.throwIfAborted();
      images.set(name, image);
    } finally {
      image.onload = null; image.onerror = null;
      signal.removeEventListener('abort', abort);
      URL.revokeObjectURL(url);
    }
  }));
  return images;
}
