import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { adjacentMedia } from '../lib/media.ts';

describe('Gallery navigation while metadata loads', () => {
  const items = [1, 2, 3, 4, 5].map((hash) => ({ hash, type: 'video' }));
  const data = { media: items[2], previous: items.slice(0, 2), next: items.slice(3), collection: '', slug: '' };

  it('keeps the same media objects and ordered neighbors when moving in either direction', () => {
    for (const [hash, before, after] of [
      [2, [1], [3, 4, 5]],
      [4, [1, 2, 3], [5]],
    ] as const) {
      const next = adjacentMedia(data, hash)!;
      assert.equal(next.media, items[hash - 1]);
      assert.deepEqual(
        next.previous.map((item) => item.hash),
        before
      );
      assert.deepEqual(
        next.next.map((item) => item.hash),
        after
      );
      assert.equal(adjacentMedia(next, 3)!.media, data.media);
    }
  });

  it('does not substitute the old media for an unknown URL', () => {
    assert.equal(adjacentMedia(data, 99), null);
    assert.equal(adjacentMedia(null, 3), null);
    assert.equal(adjacentMedia(data, 3), data);
  });
});
