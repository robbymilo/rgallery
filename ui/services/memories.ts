import { Memory } from '../types';

export async function getMemories(date?: string): Promise<Memory[]> {
  const url = new URL('/api/memories', window.location.origin);
  if (date) {
    url.searchParams.append('date', date);
  }
  const res = await fetch(url.toString());
  if (!res.ok) {
    throw new Error(`API Error: ${res.status}`);
  }
  const data: Memory[] = await res.json();
  return Array.isArray(data) ? data : [];
}
