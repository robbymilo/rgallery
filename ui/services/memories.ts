import { Memory } from '../types';

export async function getMemories(): Promise<Memory[]> {
  const res = await fetch('/api/memories');
  if (!res.ok) {
    throw new Error(`API Error: ${res.status}`);
  }
  const data: Memory[] = await res.json();
  return Array.isArray(data) ? data : [];
}
