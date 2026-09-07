import { ApiResponse } from '../types';

export const getMedia = async (
  mediaId: number,
  filters: Record<string, string> = {},
  signal?: AbortSignal
): Promise<ApiResponse> => {
  const queryParams = new URLSearchParams(filters).toString();
  const res = await fetch(`/api/media/${mediaId}${queryParams ? `?${queryParams}` : ''}`, { signal });
  if (!res.ok) {
    throw new Error(`API Error: ${res.status}`);
  }
  return res.json();
};
