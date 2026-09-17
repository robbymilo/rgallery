import type { Point } from './imageZoom';

interface Tap {
  time: number;
  point: Point;
  pointerType: string;
}

interface TapInput {
  time: number;
  point: Point;
  pointerType: string;
  isExpanded: boolean;
  isImage: boolean;
}

export function resolveImageTap(
  input: TapInput,
  previous: Tap | null
): { action: 'open' | 'close' | 'zoom' | 'none'; tap: Tap | null } {
  const repeated =
    previous !== null &&
    input.time - previous.time < 300 &&
    input.pointerType === previous.pointerType &&
    Math.hypot(input.point.x - previous.point.x, input.point.y - previous.point.y) < 30;

  // Double clicks and double taps perform only their first action, so opening
  // cannot immediately zoom or dismiss, and zoom cannot immediately reverse.
  if (repeated) {
    return { action: 'none', tap: null };
  }
  if (!input.isImage) {
    return { action: input.isExpanded ? 'close' : 'none', tap: null };
  }

  const tap: Tap = {
    time: input.time,
    point: input.point,
    pointerType: input.pointerType,
  };
  if (!input.isExpanded) return { action: 'open', tap };
  return { action: 'zoom', tap };
}
