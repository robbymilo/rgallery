export interface Point {
  x: number;
  y: number;
}

export interface ZoomTransform {
  scale: number;
  pan: Point;
}

export const fitTransform = (): ZoomTransform => ({ scale: 1, pan: { x: 0, y: 0 } });

export function nativeScale(nativeWidth: number, fittedWidth: number): number {
  return fittedWidth > 0 ? Math.max(1, nativeWidth / fittedWidth) : 1;
}

// Points are relative to the center of the fitted image. Preserve the image
// coordinate under the pointer (or moving pinch midpoint) as the scale changes.
export function zoomAt(transform: ZoomTransform, scale: number, from: Point, to: Point = from): ZoomTransform {
  const ratio = scale / transform.scale;
  return {
    scale,
    pan: {
      x: to.x - (from.x - transform.pan.x) * ratio,
      y: to.y - (from.y - transform.pan.y) * ratio,
    },
  };
}

// Carry an image coordinate across a change from the page-sized viewport to
// the overlay. Fractions are relative to the image center, independent of fit.
export function imagePoint(transform: ZoomTransform, point: Point, image: { width: number; height: number }): Point {
  return {
    x: (point.x - transform.pan.x) / (image.width * transform.scale),
    y: (point.y - transform.pan.y) / (image.height * transform.scale),
  };
}

export function placeImagePoint(
  anchor: Point,
  point: Point,
  displayedWidth: number,
  image: { width: number; height: number }
): ZoomTransform {
  const scale = nativeScale(displayedWidth, image.width);
  return {
    scale,
    pan: { x: point.x - anchor.x * image.width * scale, y: point.y - anchor.y * image.height * scale },
  };
}

export function clampTransform(
  transform: ZoomTransform,
  image: { width: number; height: number },
  viewport: { width: number; height: number }
): ZoomTransform {
  const limitX = Math.max(0, (image.width * transform.scale - viewport.width) / 2);
  const limitY = Math.max(0, (image.height * transform.scale - viewport.height) / 2);
  return {
    scale: transform.scale,
    pan: {
      x: limitX ? Math.max(-limitX, Math.min(limitX, transform.pan.x)) : 0,
      y: limitY ? Math.max(-limitY, Math.min(limitY, transform.pan.y)) : 0,
    },
  };
}
