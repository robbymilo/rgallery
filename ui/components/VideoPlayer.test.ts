import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { resumePosition, resolutionLabel, displayTime } from '../lib/video.ts';

describe('Video playback preferences', () => {
  it('ignores invalid, finished, and out-of-range resume positions', () => {
    for (const input of [null, 'NaN', 'Infinity', '-2', '1', '59', '1000']) assert.equal(resumePosition(input, 60), 0);
    assert.equal(resumePosition('12.5', 60), 12.5);
  });
  it('shows actual dimensions for portrait and smaller sources', () => {
    assert.equal(resolutionLabel(3840, 2160, 1280), '720p');
    assert.equal(resolutionLabel(2160, 3840, 1280), '720p');
    assert.equal(resolutionLabel(640, 480, 1280), '480p');
  });
  it('formats playback times without non-finite values', () => {
    assert.equal(displayTime(125.9), '2:05');
    assert.equal(displayTime(Infinity), '0:00');
  });
});
