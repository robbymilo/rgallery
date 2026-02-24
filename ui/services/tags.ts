export type Tag = {
  key: string;
  value: string;
  count?: number;
};

export async function fetchTags(): Promise<Tag[]> {
  const res = await fetch('/api/tags');
  if (!res.ok) {
    throw new Error(`API error: ${res.status}`);
  }
  const data = await res.json();
  return Array.isArray(data.tags) ? data.tags : [];
}
