import { validateStyleMin } from '@maplibre/maplibre-gl-style-spec';
import type { StyleSpecification } from 'maplibre-gl';
import { deserialize } from 'flatgeobuf/lib/mjs/geojson.js';
import type { HeaderMeta } from 'flatgeobuf/lib/mjs/header-meta.js';
import type { FeatureCollection } from 'geojson';

export function getSourceName(filename: string): string {
  return filename.slice(filename.lastIndexOf('/') + 1).replace(/\.[^.]+$/, '');
}

export type Dataset = {
  filename: string;
  style: StyleSpecification;
  data: FeatureCollection;
  header: HeaderMeta | null;
};

export async function readDataset(fgbFile: File, styleFile: File): Promise<Dataset> {
  if (!/\.fgb$/i.test(fgbFile.name)) throw new Error('Choose a .fgb data file.');
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

  const source = getSourceName(fgbFile.name);
  if (!Object.hasOwn(style.sources, source) || style.sources[source].type !== 'geojson') {
    throw new Error(`The stylesheet must define a GeoJSON source named "${source}" to match the FGB filename.`);
  }
  if (!style.layers.some(layer => 'source' in layer && layer.source === source)) {
    throw new Error(`The stylesheet must contain at least one layer using source "${source}".`);
  }

  const data: FeatureCollection = { type: 'FeatureCollection', features: [] };
  let header: HeaderMeta | null = null;
  let expectedFeatures = 0;
  try {
    const bytes = new Uint8Array(await fgbFile.arrayBuffer());
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
        header = metadata;
      },
    })) {
      feature.id ??= data.features.length;
      data.features.push(feature);
    }
    if (expectedFeatures > 0 && data.features.length !== expectedFeatures) throw new Error('Truncated features');
  } catch {
    throw new Error('Unable to read the FlatGeobuf file. Choose a valid .fgb file.');
  }
  return { filename: fgbFile.name, style, data, header };
}
