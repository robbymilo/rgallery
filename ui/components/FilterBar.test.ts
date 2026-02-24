import { describe, it } from 'node:test';
import assert from 'node:assert';
import { parseSearchTokens } from '../lib/parseSearchTokens.ts';

describe('FilterBar token parsing', () => {
  it('should parse single-word camera value', () => {
    const result = parseSearchTokens('camera:NIKON');
    assert.strictEqual(result.camera, 'NIKON');
    assert.strictEqual(result.searchQuery, '');
  });

  it('should parse multi-word camera value', () => {
    const result = parseSearchTokens('camera:NIKON D750');
    assert.strictEqual(result.camera, 'NIKON D750');
    assert.strictEqual(result.searchQuery, '');
  });

  it('should parse multi-word camera value with three words', () => {
    const result = parseSearchTokens('camera:NIKON Z 9');
    assert.strictEqual(result.camera, 'NIKON Z 9');
    assert.strictEqual(result.searchQuery, '');
  });

  it('should parse camera and preserve remaining search query', () => {
    const result = parseSearchTokens('sunset beach camera:NIKON D750');
    assert.strictEqual(result.camera, 'NIKON D750');
    assert.strictEqual(result.searchQuery, 'sunset beach');
  });

  it('should parse tag with search query', () => {
    const result = parseSearchTokens('mountains winter tag:landscape');
    assert.strictEqual(result.tag, 'landscape');
    assert.strictEqual(result.searchQuery, 'mountains winter');
  });

  it('should parse folder token', () => {
    const result = parseSearchTokens('folder:2022/vacation');
    assert.strictEqual(result.folder, '2022/vacation');
    assert.strictEqual(result.searchQuery, '');
  });

  it('should parse software token', () => {
    const result = parseSearchTokens('software:Adobe Lightroom Classic');
    assert.strictEqual(result.software, 'Adobe Lightroom Classic');
    assert.strictEqual(result.searchQuery, '');
  });

  it('should parse focallength35 as number', () => {
    const result = parseSearchTokens('focallength35:50');
    assert.strictEqual(result.focallength35, 50);
    assert.strictEqual(result.searchQuery, '');
  });

  it('should handle empty string', () => {
    const result = parseSearchTokens('');
    assert.strictEqual(result.searchQuery, '');
    assert.strictEqual(result.camera, undefined);
  });

  it('should handle plain search without tokens', () => {
    const result = parseSearchTokens('sunset beach');
    assert.strictEqual(result.searchQuery, 'sunset beach');
    assert.strictEqual(result.camera, undefined);
  });
});
