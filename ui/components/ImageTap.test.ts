import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { resolveImageTap } from '../lib/imageTap.ts';

const click = {
  time: 1000,
  point: { x: 200, y: 150 },
  pointerType: 'mouse',
  isExpanded: true,
  isImage: true,
};

describe('Image viewer tap actions', () => {
  it('opens an image immediately, but does not open from the background', () => {
    assert.equal(resolveImageTap({ ...click, isExpanded: false }, null).action, 'open');
    assert.equal(resolveImageTap({ ...click, isExpanded: false, isImage: false }, null).action, 'none');
  });

  it('ignores the second click of an opening double click, even if layout changes its target', () => {
    const opened = resolveImageTap({ ...click, isExpanded: false }, null);
    for (const isImage of [true, false]) {
      const second = resolveImageTap({ ...click, time: 1100, isImage }, opened.tap);
      assert.equal(second.action, 'none');
      assert.equal(second.tap, null);
    }
    assert.equal(resolveImageTap({ ...click, time: 1400 }, opened.tap).action, 'zoom');
  });

  it('toggles zoom immediately on image clicks, without dismissing the viewer', () => {
    const zoomed = resolveImageTap(click, null);
    assert.equal(zoomed.action, 'zoom');
    assert.equal(resolveImageTap({ ...click, time: 1500 }, zoomed.tap).action, 'zoom');
  });

  it('does not toggle zoom twice for a desktop double click', () => {
    const first = resolveImageTap(click, null);
    assert.equal(resolveImageTap({ ...click, time: 1100 }, first.tap).action, 'none');
  });

  it('closes immediately from the expanded background', () => {
    const result = resolveImageTap({ ...click, isImage: false }, null);
    assert.equal(result.action, 'close');
    assert.equal(result.tap, null);
  });

  it('opens on touch and ignores the second tap of the opening gesture', () => {
    const touch = { ...click, pointerType: 'touch' };
    const first = resolveImageTap({ ...touch, isExpanded: false }, null);
    assert.equal(first.action, 'open');
    assert.equal(resolveImageTap({ ...touch, time: 1100 }, first.tap).action, 'none');
  });

  it('leaves a single touch tap alone and zooms on a subsequent double tap', () => {
    const touch = { ...click, pointerType: 'touch' };
    const first = resolveImageTap(touch, null);
    assert.equal(first.action, 'none');
    const second = resolveImageTap({ ...touch, time: 1100 }, first.tap);
    assert.equal(second.action, 'zoom');
    assert.equal(second.tap, null);
  });

  it('requires touch double taps to be close in time and position', () => {
    const touch = { ...click, pointerType: 'touch' };
    const first = resolveImageTap(touch, null);
    assert.equal(resolveImageTap({ ...touch, time: 1400 }, first.tap).action, 'none');
    assert.equal(resolveImageTap({ ...touch, time: 1100, point: { x: 400, y: 150 } }, first.tap).action, 'none');
  });

  it('does not combine different pointer types into a double tap', () => {
    const first = resolveImageTap(click, null);
    assert.equal(resolveImageTap({ ...click, time: 1100, pointerType: 'touch' }, first.tap).action, 'none');
  });
});
