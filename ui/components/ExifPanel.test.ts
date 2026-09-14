import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { displayMediaDate } from '../lib/date.ts';

describe('Optional media dates', () => {
  it('omits missing and invalid dates without throwing during navigation', () => {
    for (const value of [undefined, '', 'not-a-date', '2026-99-99']) {
      assert.equal(displayMediaDate(value), undefined);
      assert.equal(displayMediaDate(value, 120), undefined);
    }
    assert.equal(displayMediaDate('2026-09-10T12:00:00Z', Number.MAX_VALUE), undefined);
  });

  it('retains the timestamp and stored offset formatting for valid metadata', () => {
    assert.equal(displayMediaDate('2026-09-10T12:00:00Z'), 'Thu, 10 September 2026 12:00:00.000');
    assert.equal(displayMediaDate('2026-09-10T12:00:00Z', NaN), 'Thu, 10 September 2026 12:00:00.000');
    assert.equal(displayMediaDate('2026-09-10T12:00:00Z', 120), 'Thu, 10 September 2026 14:00:00.000');
  });
});
