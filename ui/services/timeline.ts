import { TimelineResponse, TimelineFilters, TimelineBucket } from '../types';

export const fetchPhotos = async (
  cursorOffsetStr: string | null,
  filters?: TimelineFilters,
  signal?: AbortSignal
): Promise<TimelineResponse> => {
  const url = new URL('/api/timeline', window.location.origin);

  console.log('[timeline] Fetching photos with cursor:', cursorOffsetStr, 'filters:', filters);

  if (cursorOffsetStr) {
    url.searchParams.set('cursor', cursorOffsetStr);
  }

  // Apply filters to query params
  if (filters) {
    if (filters.term) url.searchParams.set('term', filters.term);
    if (filters.rating && filters.rating > 1) url.searchParams.set('rating', filters.rating.toString());
    if (filters.tag) url.searchParams.set('tag', filters.tag);
    if (filters.type) url.searchParams.set('type', filters.type);
    if (filters.orderby) url.searchParams.set('orderby', filters.orderby);
    if (filters.direction) url.searchParams.set('direction', filters.direction);
    if (filters.camera) url.searchParams.set('camera', filters.camera);
    if (filters.lens) url.searchParams.set('lens', filters.lens);
    if (filters.folder) url.searchParams.set('folder', filters.folder);
    if (filters.subject) url.searchParams.set('subject', filters.subject);
    if (filters.software) url.searchParams.set('software', filters.software);
    if (filters.focallength35) url.searchParams.set('focallength35', filters.focallength35.toString());
  }

  const res = await fetch(url.toString(), { signal });

  if (!res.ok) {
    throw new Error(`API Error: ${res.status} ${res.statusText}`);
  }

  const data: TimelineResponse = await res.json();

  return data;
};

export const fetchTimelineBuckets = async (filters?: TimelineFilters): Promise<TimelineBucket[]> => {
  const url = new URL('/api/timeline/buckets', window.location.origin);
  if (filters) {
    Object.entries(filters).forEach(([key, value]) => {
      if (value !== undefined && value !== null) url.searchParams.set(key, value.toString());
    });
  }
  const res = await fetch(url.toString());
  if (!res.ok) throw new Error(`API Error: ${res.status} ${res.statusText}`);
  return res.json();
};

export const fetchTimelineBucket = async (
  month: string,
  filters?: TimelineFilters,
  signal?: AbortSignal,
  cursor?: string
): Promise<TimelineResponse> => {
  const url = new URL('/api/timeline/bucket', window.location.origin);
  url.searchParams.set('month', month);
  if (cursor) url.searchParams.set('cursor', cursor);
  if (filters) {
    Object.entries(filters).forEach(([key, value]) => {
      if (value !== undefined && value !== null) url.searchParams.set(key, value.toString());
    });
  }
  const res = await fetch(url.toString(), { signal });
  if (!res.ok) throw new Error(`API Error: ${res.status} ${res.statusText}`);
  return res.json();
};
