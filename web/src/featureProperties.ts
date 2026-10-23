import type { Feature } from 'geojson';

const nameFields = ['name', 'navn', 'skytefeltNavn'];
const identityFields = new Set([...nameFields, 'fid', 'id', 'gml_id']);

export function featureTitle(feature: Pick<Feature, 'properties' | 'id'>, index: number): string {
  for (const key of nameFields) {
    const value = feature.properties?.[key];
    if (typeof value === 'string' && value.trim()) return value;
  }
  return `Object ${feature.id ?? index + 1}`;
}

export function propertyGroups(properties: Feature['properties']) {
  const entries = Object.entries(properties ?? {});
  return [
    { title: 'Identification', entries: entries.filter(([key]) => identityFields.has(key)) },
    { title: 'Other properties', entries: entries.filter(([key]) => !identityFields.has(key)) },
  ].filter(group => group.entries.length);
}

export function scalarValue(value: unknown): { type: string; text: string } {
  if (value === null) return { type: 'Null', text: 'null' };
  if (typeof value === 'string') return { type: 'String', text: value.trim() ? value : JSON.stringify(value) };
  if (typeof value === 'boolean') return { type: 'Boolean', text: String(value) };
  if (typeof value === 'number') return { type: 'Number', text: Object.is(value, -0) ? '-0' : String(value) };
  return { type: typeof value, text: String(value) };
}
