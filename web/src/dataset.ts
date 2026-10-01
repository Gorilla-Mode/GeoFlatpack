import { validateStyleMin } from '@maplibre/maplibre-gl-style-spec';
import type { StyleSpecification } from 'maplibre-gl';
import { deserialize } from 'flatgeobuf/lib/mjs/geojson.js';
import type { HeaderMeta } from 'flatgeobuf/lib/mjs/header-meta.js';
import { Header } from 'flatgeobuf/lib/mjs/flat-geobuf/header.js';
import { ByteBuffer } from 'flatbuffers';
import type { FeatureCollection, Geometry } from 'geojson';

export type DatasetHeader = HeaderMeta & { name: string | null; hasZ: boolean };

export function getSourceName(filename: string): string {
  return filename.slice(filename.lastIndexOf('/') + 1).replace(/\.[^.]+$/, '');
}

export type Dataset = {
  filename: string;
  data: FeatureCollection;
  header: DatasetHeader | null;
};

export function getBounds(data: FeatureCollection): [[number, number], [number, number]] | null {
  let west = Infinity, south = Infinity, east = -Infinity, north = -Infinity;
  function visit(coordinates: unknown): void {
    if (!Array.isArray(coordinates)) return;
    const [x, y] = coordinates;
    if (typeof x === 'number' && typeof y === 'number') {
      if (Number.isFinite(x) && Number.isFinite(y)) {
        west = Math.min(west, x);
        south = Math.min(south, y);
        east = Math.max(east, x);
        north = Math.max(north, y);
      }
    } else coordinates.forEach(visit);
  }
  function visitGeometry(geometry: Geometry | null): void {
    if (geometry?.type === 'GeometryCollection') geometry.geometries.forEach(visitGeometry);
    else if (geometry) visit(geometry.coordinates);
  }
  data.features.forEach(feature => visitGeometry(feature.geometry));
  return Number.isFinite(west) ? [[west, south], [east, north]] : null;
}

export async function readStylesheet(styleFile: File): Promise<StyleSpecification> {
  if (!/\.json$/i.test(styleFile.name)) throw new Error('Choose a MapLibre .json stylesheet.');

  let style: StyleSpecification;
  try {
    style = JSON.parse(await styleFile.text());
  } catch {
    throw new Error('The stylesheet is not valid JSON.');
  }
  if (!style || typeof style !== 'object') throw new Error('Choose a MapLibre style object.');
  const errors = validateStyleMin(style);
  if (errors.length) throw new Error(`Invalid MapLibre style: ${errors[0].message}`);
  return style;
}

export function getStyleMismatch(filename: string, style: StyleSpecification): string | null {
  const source = getSourceName(filename);
  if (!Object.hasOwn(style.sources, source) || style.sources[source].type !== 'geojson') {
    return `The stylesheet must define a GeoJSON source named "${source}" to match the FGB filename.`;
  }
  if (!style.layers.some(layer => 'source' in layer && layer.source === source)) {
    return `The stylesheet must contain at least one layer using source "${source}".`;
  }
  return null;
}

export async function readDataset(fgbFile: File, style: StyleSpecification): Promise<Dataset> {
  try {
    if (!/\.fgb$/i.test(fgbFile.name)) throw new Error('Choose a .fgb data file.');
    const mismatch = getStyleMismatch(fgbFile.name, style);
    if (mismatch) throw new Error(mismatch);
    return { filename: fgbFile.name, ...await decodeFgb(await fgbFile.arrayBuffer()) };
  } catch (cause) {
    throw new Error(`${fgbFile.name}: ${cause instanceof Error ? cause.message : cause}`);
  }
}

export async function decodeFgb(buffer: ArrayBuffer, signal?: AbortSignal, onHeader?: (header: DatasetHeader) => void) {
  const data: FeatureCollection = { type: 'FeatureCollection', features: [] };
  let header: DatasetHeader | null = null;
  let expectedFeatures = 0;
  try {
    signal?.throwIfAborted();
    const bytes = new Uint8Array(buffer);
    // Reject incomplete headers before the decoder reads FlatBuffer offsets.
    if (bytes.length < 12) throw new Error('Missing header');
    const headerLength = new DataView(bytes.buffer).getUint32(8, true);
    if (headerLength < 8 || headerLength > bytes.length - 12) throw new Error('Truncated header');
    for await (const feature of deserialize(bytes, {
      headerMetaFn: metadata => {
        // The decoder cannot traverse an indexed file with a missing feature count.
        if (!Number.isSafeInteger(metadata.featuresCount) || metadata.featuresCount < 0 ||
          (metadata.indexNodeSize > 0 && metadata.featuresCount === 0)) {
          throw new Error('Invalid feature count');
        }
        expectedFeatures = metadata.featuresCount;
        // HeaderMeta omits the dataset name and dimensional flags.
        const rawHeader = Header.getSizePrefixedRootAsHeader(new ByteBuffer(bytes.subarray(8, 12 + headerLength)));
        header = { ...metadata, name: rawHeader.name(), hasZ: rawHeader.hasZ() };
        onHeader?.(header);
      },
    })) {
      signal?.throwIfAborted();
      feature.id ??= data.features.length;
      data.features.push(feature);
    }
    if (expectedFeatures > 0 && data.features.length !== expectedFeatures) throw new Error('Truncated features');
  } catch {
    signal?.throwIfAborted();
    throw new Error('Unable to read the FlatGeobuf file. Choose a valid .fgb file.');
  }
  return { data, header };
}
