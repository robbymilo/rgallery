import type { Point } from './imageZoom';

interface Tap {
  time: number;
  point: Point;
  pointerType: string;
  opening: boolean;
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

  // A desktop double click performs only its first action. On touch, an
  // opening double tap must not immediately zoom (or dismiss) the viewer.
  if (repeated && (input.pointerType !== 'touch' || previous.opening)) {
    return { action: 'none', tap: null };
  }
  if (!input.isImage) {
    return { action: input.isExpanded ? 'close' : 'none', tap: null };
  }

  const tap: Tap = {
    time: input.time,
    point: input.point,
    pointerType: input.pointerType,
    opening: !input.isExpanded,
  };
  if (!input.isExpanded) return { action: 'open', tap };
  if (input.pointerType === 'touch') {
    return repeated ? { action: 'zoom', tap: null } : { action: 'none', tap };
  }
  return { action: 'zoom', tap };
}
