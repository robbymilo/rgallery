import type { ApiResponse } from '../types';

// Keep the current slide visible while its metadata loads.
export function adjacentMedia(data: ApiResponse | null, hash: number): ApiResponse | null {
  if (!data || data.media.hash === hash) return data;
  const items = [...data.previous, data.media, ...data.next];
  const index = items.findIndex((item) => item.hash === hash);
  if (index < 0) return null;
  return { ...data, media: items[index], previous: items.slice(0, index), next: items.slice(index + 1) };
}
