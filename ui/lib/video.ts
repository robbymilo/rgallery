export type VideoQuality = 'auto' | 'saver' | 'small' | 'high';
export const qualityKey = 'rgallery:video-quality';

// Reads the saved quality preference, defaulting to Auto.
export function readQuality(): VideoQuality {
  try {
    const value = localStorage.getItem(qualityKey);
    if (value === 'saver' || value === 'small' || value === 'high') return value;
  } catch {}
  return 'auto';
}

// Returns a saved position only when playback can resume there.
export function resumePosition(value: string | null, duration: number): number {
  const position = Number(value);
  return Number.isFinite(position) && position >= 2 && position < duration - 2 ? position : 0;
}

// Formats seconds as a playback timestamp.
export function displayTime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return '0:00';
  return `${Math.floor(seconds / 60)}:${Math.floor(seconds % 60)
    .toString()
    .padStart(2, '0')}`;
}

// Returns the displayed resolution after applying the size limit.
export function resolutionLabel(width: number, height: number, edge: number): string {
  const scale = Math.min(1, edge / Math.max(width, height));
  return `${Math.max(2, Math.floor((Math.min(width, height) * scale) / 2) * 2)}p`;
}

// Fetches video metadata and reports playback request errors.
export async function videoJSON<T>(url: string, signal: AbortSignal): Promise<T> {
  const response = await fetch(url, { signal, credentials: 'same-origin', cache: 'no-store' });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(
      body.error ||
        (response.status === 401 ? 'Sign in again to play this video.' : 'Unable to prepare this video. Please retry.')
    );
  }
  return response.json() as Promise<T>;
}
