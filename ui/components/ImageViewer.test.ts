import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { clampTransform, fitTransform, imagePoint, nativeScale, placeImagePoint, zoomAt } from '../lib/imageZoom.ts';

describe('Image viewer zoom geometry', () => {
  it('zooms large images to native size and does not shrink small images', () => {
    assert.equal(nativeScale(6000, 1200), 5);
    assert.equal(nativeScale(600, 600), 1);
    assert.equal(nativeScale(600, 1200), 1);
    assert.equal(nativeScale(6000, 0), 1);
  });

  it('keeps the clicked image coordinate under the pointer at 100%', () => {
    const point = { x: 180, y: -90 };
    const result = zoomAt(fitTransform(), nativeScale(6000, 1200), point);
    assert.equal(result.scale, 5);
    assert.equal(point.x * result.scale + result.pan.x, point.x);
    assert.equal(point.y * result.scale + result.pan.y, point.y);
  });

  it('follows a moving pinch midpoint when already zoomed and panned', () => {
    const start = { scale: 2, pan: { x: -100, y: 80 } };
    const from = { x: 120, y: -40 };
    const to = { x: 170, y: 20 };
    const result = zoomAt(start, 3, from, to);
    assert.equal(((from.x - start.pan.x) / start.scale) * result.scale + result.pan.x, to.x);
    assert.equal(((from.y - start.pan.y) / start.scale) * result.scale + result.pan.y, to.y);
  });

  it('preserves the clicked image point at native size when the viewer expands', () => {
    const fitted = { width: 900, height: 600 };
    const expanded = { width: 1200, height: 800 };
    const before = { x: 135, y: -60 };
    // The full-window viewer has a different center from the page image.
    const after = { x: 135, y: -120 };
    const anchor = imagePoint(fitTransform(), before, fitted);
    const result = placeImagePoint(anchor, after, 3600, expanded);
    assert.equal(result.scale, 3);
    assert.deepEqual(imagePoint(result, after, expanded), anchor);
  });

  it('preserves pinch size and focal point across expansion from a panned image', () => {
    const fitted = { width: 300, height: 450 };
    const expanded = { width: 390, height: 585 };
    const start = { scale: 2, pan: { x: 30, y: -100 } };
    const anchor = imagePoint(start, { x: 50, y: 80 }, fitted);
    const target = { x: 40, y: -30 };
    const result = placeImagePoint(anchor, target, fitted.width * start.scale, expanded);
    assert.equal(result.scale * expanded.width, 600);
    const restored = imagePoint(result, target, expanded);
    assert.ok(Math.abs(restored.x - anchor.x) < 1e-10);
    assert.ok(Math.abs(restored.y - anchor.y) < 1e-10);
  });

  it('bounds landscape panning and centers the axis that fits in the viewport', () => {
    assert.deepEqual(
      clampTransform(
        { scale: 2, pan: { x: 2000, y: -2000 } },
        { width: 1000, height: 300 },
        { width: 1000, height: 800 }
      ),
      { scale: 2, pan: { x: 500, y: 0 } }
    );
  });

  it('bounds portrait panning and returns to center when pinched back to fit', () => {
    const image = { width: 300, height: 800 };
    const viewport = { width: 1000, height: 800 };
    assert.deepEqual(clampTransform({ scale: 2, pan: { x: -2000, y: -2000 } }, image, viewport), {
      scale: 2,
      pan: { x: 0, y: -400 },
    });
    assert.deepEqual(clampTransform({ scale: 1, pan: { x: -2000, y: 2000 } }, image, viewport), fitTransform());
  });
});
