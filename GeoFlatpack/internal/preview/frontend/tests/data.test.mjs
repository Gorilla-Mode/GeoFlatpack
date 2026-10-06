import test from 'node:test';
import assert from 'node:assert/strict';
import { getBounds, loadEmbeddedIcons } from '../data.ts';

test('bounds cover actual multipart geometry and skip missing coordinates', () => {
  const data = { features: [
    { geometry: null },
    { geometry: { type: 'GeometryCollection', geometries: [
      { type: 'Point', coordinates: [11, 60, 100] },
      { type: 'MultiPolygon', coordinates: [[[[12, 61], [10, 58], [12, 61]]]] },
    ] } },
  ] };
  assert.deepEqual(getBounds(data), [[10, 58], [12, 61]]);
  assert.equal(getBounds({ features: [{ geometry: null }] }), null);
});

test('embedded SVGs decode with original dimensions and abort promptly', async () => {
  const previous = globalThis.Image;
  class MockImage {
    naturalWidth = 24;
    naturalHeight = 32;
    set src(value) { if (value) queueMicrotask(() => this.onload?.()); }
    removeAttribute() {}
  }
  globalThis.Image = MockImage;
  try {
    const signal = new AbortController().signal;
    const images = await loadEmbeddedIcons({ metadata: { 'geoflatpack:icons': { example: '<svg/>' } } }, signal);
    assert.equal(images.get('example').naturalWidth, 24);
    assert.equal(images.get('example').naturalHeight, 32);
    assert.equal((await loadEmbeddedIcons({}, signal)).size, 0);
    await assert.rejects(loadEmbeddedIcons({ metadata: { 'geoflatpack:icons': [] } }, signal), /Invalid/);
    const abort = new AbortController();
    abort.abort();
    await assert.rejects(loadEmbeddedIcons({ metadata: { 'geoflatpack:icons': { example: '<svg/>' } } }, abort.signal), /abort/i);
  } finally { globalThis.Image = previous; }
});
