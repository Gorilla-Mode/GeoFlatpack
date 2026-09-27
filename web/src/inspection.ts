import { GeometryType } from 'flatgeobuf/lib/mjs/flat-geobuf/geometry-type.js';
import { ColumnType } from 'flatgeobuf/lib/mjs/flat-geobuf/column-type.js';
import type { FeatureCollection, Geometry } from 'geojson';
import type { DatasetHeader } from './dataset';

function coordinatesTree(value: unknown): unknown {
  if (!Array.isArray(value)) return value;
  if (value.length > 0 && value.every(coordinate => typeof coordinate === 'number')) {
    return Object.fromEntries(value.map((coordinate, index) => [
      ['X', 'Y', 'Z'][index] ?? `Coordinate ${index + 1}`, coordinate,
    ]));
  }
  return value.map(coordinatesTree);
}

function geometryTree(geometry: Geometry | null): Record<string, unknown> {
  if (!geometry) return { 'Geometry type': null };
  return {
    'Geometry type': geometry.type,
    ...(geometry.type === 'GeometryCollection'
      ? { Geometries: geometry.geometries.map(geometryTree) }
      : { Coordinates: coordinatesTree(geometry.coordinates) }),
  };
}

export function createInspection(header: DatasetHeader | null, data?: FeatureCollection) {
  if (!header) return null;
  const { name, hasZ, geometryType, featuresCount, indexNodeSize, envelope, crs, columns, ...details } = header;
  const bounds = envelope && Array.from(envelope);
  const crsCode = crs?.code_string || (crs?.code ? String(crs.code) : null);
  const fields = columns?.map(({ name, type, ...details }) => ({
    Name: name,
    Type: ColumnType[type] ?? `Unrecognized (${type})`,
    Details: details,
  })) ?? [];
  return {
    'FGB Header': {
      Name: name,
      'Geometry type': GeometryType[geometryType] ?? `Unrecognized (${geometryType})`,
      'Feature count': featuresCount,
      'Has Z': hasZ,
      'Index node size': indexNodeSize,
      Bounds: bounds?.length === 4
        ? { Min: { X: bounds[0], Y: bounds[1] }, Max: { X: bounds[2], Y: bounds[3] } }
        : bounds,
      CRS: crs?.org && crsCode ? `${crs.org}:${crsCode}` : crs?.name ?? crsCode ?? null,
      Fields: fields,
      Details: { ...details, CRS: crs },
    },
    // A parsed header remains inspectable even if feature decoding fails.
    ...(data ? { Features: Object.fromEntries(data.features.map((feature, index) => [
      `Feature ${index + 1}`,
      { ...geometryTree(feature.geometry), Properties: feature.properties },
    ])) } : {}),
  };
}
