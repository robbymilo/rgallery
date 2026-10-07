import { useCallback, useLayoutEffect, useRef, useState } from 'react';
import type { PointerEvent as ReactPointerEvent } from 'react';
import { clampTransform, fitTransform, imagePoint, nativeScale, placeImagePoint, zoomAt } from '../lib/imageZoom';
import type { Point, ZoomTransform } from '../lib/imageZoom';
import { resolveImageTap } from '../lib/imageTap.ts';
import { useZoomTransition } from './useZoomTransition';

const SLIDE_DURATION = 300;
const distance = (a: Point, b: Point) => Math.hypot(a.x - b.x, a.y - b.y);
const midpoint = (a: Point, b: Point): Point => ({ x: (a.x + b.x) / 2, y: (a.y + b.y) / 2 });

interface Options {
  mediaKey: number;
  nativeWidth?: number;
  isVideo: boolean;
  hasPrevious: boolean;
  hasNext: boolean;
  onPrev: () => void;
  onNext: () => void;
}

export function useImageGestures(options: Options) {
  const containerRef = useRef<HTMLDivElement>(null);
  const imageRef = useRef<HTMLImageElement>(null);
  const { runZoomTransition, cancelZoomTransition } = useZoomTransition(options.mediaKey, imageRef, containerRef);
  const viewportRef = useRef<HTMLDivElement>(null);
  const [isExpanded, setIsExpanded] = useState(false);
  const pendingExpansion = useRef<{ anchor: Point; point: Point; width: number } | null>(null);
  const [transform, setTransform] = useState(fitTransform);
  const current = useRef(transform);
  const liveFrame = useRef<number | null>(null);
  const [dragOffset, setDragOffset] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [transitionEnabled, setTransitionEnabled] = useState(false);
  const [animateZoom, setAnimateZoom] = useState(false);
  const pointers = useRef(new Map<number, Point>());
  const gesture = useRef<{
    start: Point;
    transform: ZoomTransform;
    distance: number;
    moved: boolean;
    pinched: boolean;
    isImage: boolean;
  } | null>(null);
  const tap = useRef<ReturnType<typeof resolveImageTap>['tap']>(null);
  const slideTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearTap = useCallback(() => {
    tap.current = null;
  }, []);

  const bound = useCallback((value: ZoomTransform) => {
    const image = imageRef.current;
    const viewport = viewportRef.current;
    return image && viewport
      ? clampTransform(
          value,
          { width: image.clientWidth, height: image.clientHeight },
          { width: viewport.clientWidth, height: viewport.clientHeight }
        )
      : fitTransform();
  }, []);

  const writeTransform = useCallback((value: ZoomTransform) => {
    if (!imageRef.current) return;
    imageRef.current.style.transform = `translate3d(${value.pan.x}px, ${value.pan.y}px, 0) scale(${value.scale})`;
  }, []);

  const update = useCallback(
    (value: ZoomTransform) => {
      const bounded = bound(value);
      current.current = bounded;
      if (liveFrame.current !== null) {
        cancelAnimationFrame(liveFrame.current);
        liveFrame.current = null;
      }
      writeTransform(bounded);
      setTransform(bounded);
    },
    [bound, writeTransform]
  );

  // Pointer moves update the compositor directly and only schedule one write
  // per frame. React state is committed when the gesture ends.
  const updateLive = useCallback(
    (value: ZoomTransform) => {
      current.current = bound(value);
      if (liveFrame.current === null) {
        liveFrame.current = requestAnimationFrame(() => {
          liveFrame.current = null;
          writeTransform(current.current);
        });
      }
    },
    [bound, writeTransform]
  );

  const commitLive = useCallback(() => {
    if (liveFrame.current !== null) {
      cancelAnimationFrame(liveFrame.current);
      liveFrame.current = null;
    }
    writeTransform(current.current);
    setTransform(current.current);
  }, [writeTransform]);

  const cancelLive = useCallback(() => {
    if (liveFrame.current !== null) {
      cancelAnimationFrame(liveFrame.current);
      liveFrame.current = null;
    }
  }, []);

  const getNativeScale = useCallback(() => {
    const image = imageRef.current;
    return image ? nativeScale(options.nativeWidth || image.naturalWidth, image.clientWidth) : 1;
  }, [options.nativeWidth]);

  const relativePoint = useCallback((point: Point): Point => {
    const rect = viewportRef.current!.getBoundingClientRect();
    return { x: point.x - rect.left - rect.width / 2, y: point.y - rect.top - rect.height / 2 };
  }, []);

  const resetZoom = useCallback(() => {
    clearTap();
    pendingExpansion.current = null;
    pointers.current.clear();
    gesture.current = null;
    if (slideTimer.current !== null) clearTimeout(slideTimer.current);
    slideTimer.current = null;
    setIsDragging(false);
    setDragOffset(0);
    setTransitionEnabled(false);
    setAnimateZoom(false);
    update(fitTransform());
    setIsExpanded(false);
  }, [clearTap, update]);

  const closeZoom = useCallback(
    (animate = false) => {
      if (animate) runZoomTransition(resetZoom);
      else {
        cancelZoomTransition();
        resetZoom();
      }
    },
    [runZoomTransition, cancelZoomTransition, resetZoom]
  );

  const expandZoom = useCallback(
    (point: Point, value: ZoomTransform) => {
      const image = imageRef.current;
      if (!image?.clientWidth || !image.clientHeight) return;
      pendingExpansion.current = {
        anchor: imagePoint(value, relativePoint(point), { width: image.clientWidth, height: image.clientHeight }),
        point,
        width: image.clientWidth * value.scale,
      };
      setIsExpanded(true);
    },
    [relativePoint]
  );

  const openViewer = useCallback(
    (animate = false) => {
      clearTap();
      if (isExpanded || options.isVideo || pointers.current.size || slideTimer.current !== null) return;
      const open = () => {
        pendingExpansion.current = null;
        setAnimateZoom(false);
        update(fitTransform());
        setIsExpanded(true);
      };
      if (animate) runZoomTransition(open);
      else open();
    },
    [clearTap, isExpanded, options.isVideo, update, runZoomTransition]
  );

  const toggleZoom = useCallback(
    (point?: Point, animate = false) => {
      clearTap();
      if (!viewportRef.current || pointers.current.size || slideTimer.current !== null) return;
      if (!isExpanded) {
        openViewer(animate);
        return;
      }
      if (options.isVideo) return;
      const rect = viewportRef.current.getBoundingClientRect();
      const target = point || { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
      const value =
        current.current.scale > 1 ? fitTransform() : zoomAt(current.current, getNativeScale(), relativePoint(target));
      const commitZoom = () => {
        setAnimateZoom(!animate);
        update(value);
      };
      if (animate) runZoomTransition(commitZoom);
      else commitZoom();
    },
    [clearTap, isExpanded, openViewer, options.isVideo, getNativeScale, relativePoint, update, runZoomTransition]
  );

  const rebaseGesture = useCallback((pinched: boolean) => {
    const [first, second] = [...pointers.current.values()];
    gesture.current = first
      ? {
          start: second ? midpoint(first, second) : first,
          distance: second ? Math.max(1, distance(first, second)) : 0,
          transform: current.current,
          moved: pinched,
          pinched,
          isImage: gesture.current?.isImage ?? false,
        }
      : null;
  }, []);

  useLayoutEffect(() => {
    const pending = pendingExpansion.current;
    const image = imageRef.current;
    if (!isExpanded || !pending || !image) return;
    pendingExpansion.current = null;
    setAnimateZoom(false);
    update(
      placeImagePoint(pending.anchor, relativePoint(pending.point), pending.width, {
        width: image.clientWidth,
        height: image.clientHeight,
      })
    );
    // Pointer capture stays on the same element as it expands. Continue an
    // in-progress pinch from its new fit and pan, without losing either finger.
    if (pointers.current.size) rebaseGesture(true);
  }, [isExpanded, relativePoint, update, rebaseGesture]);

  useLayoutEffect(() => {
    cancelLive();
    clearTap();
    if (slideTimer.current !== null) clearTimeout(slideTimer.current);
    slideTimer.current = null;
    pointers.current.clear();
    gesture.current = null;
    setAnimateZoom(false);
    update(fitTransform());
    setDragOffset(0);
    setIsDragging(false);
    setTransitionEnabled(false);
    return () => {
      cancelLive();
      clearTap();
      if (slideTimer.current !== null) clearTimeout(slideTimer.current);
    };
  }, [options.mediaKey, cancelLive, clearTap, update]);

  useLayoutEffect(() => {
    const viewport = viewportRef.current;
    if (!viewport) return;
    const observer = new ResizeObserver(() => {
      setAnimateZoom(false);
      update({ ...current.current, scale: Math.min(current.current.scale, Math.max(4, getNativeScale())) });
    });
    observer.observe(viewport);
    if (imageRef.current) observer.observe(imageRef.current);
    return () => observer.disconnect();
  }, [options.mediaKey, getNativeScale, update]);

  const onPointerDown = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (
      event.button !== 0 ||
      slideTimer.current !== null ||
      (event.target as HTMLElement).closest('button, a, video, input, select, textarea, [data-zoom-toolbar]')
    )
      return;
    event.currentTarget.setPointerCapture(event.pointerId);
    pointers.current.set(event.pointerId, { x: event.clientX, y: event.clientY });
    const pinched = pointers.current.size > 1;
    if (pinched) {
      cancelZoomTransition();
      clearTap();
    }
    rebaseGesture(pinched || !!gesture.current?.pinched);
    if (!pinched && gesture.current) gesture.current.isImage = event.target === imageRef.current;
    setAnimateZoom(false);
    setIsDragging(true);
    setTransitionEnabled(false);
    if (pinched) setDragOffset(0);
  };

  const onPointerMove = (event: ReactPointerEvent<HTMLDivElement>) => {
    const active = gesture.current;
    if (!active || !pointers.current.has(event.pointerId)) return;
    pointers.current.set(event.pointerId, { x: event.clientX, y: event.clientY });
    const [first, second] = [...pointers.current.values()];
    if (second) {
      if (options.isVideo || !viewportRef.current) return;
      const scale = Math.max(
        1,
        Math.min(Math.max(4, getNativeScale()), (active.transform.scale * distance(first, second)) / active.distance)
      );
      const point = midpoint(first, second);
      const value = zoomAt(active.transform, scale, relativePoint(active.start), relativePoint(point));
      if (!isExpanded && scale > 1.02) {
        expandZoom(point, value);
      } else if (isExpanded) {
        updateLive(value);
      }
      return;
    }
    const dx = first.x - active.start.x;
    const dy = first.y - active.start.y;
    if (distance(first, active.start) > 5) {
      cancelZoomTransition();
      active.moved = true;
      clearTap();
    }
    if (active.transform.scale > 1) {
      updateLive({
        scale: active.transform.scale,
        pan: { x: active.transform.pan.x + dx, y: active.transform.pan.y + dy },
      });
    } else if (!active.pinched && Math.abs(dx) > Math.abs(dy)) {
      setDragOffset(dx);
    } else {
      setDragOffset(0);
    }
  };

  const endPointer = (event: ReactPointerEvent<HTMLDivElement>, cancelled = false) => {
    const active = gesture.current;
    if (!active || !pointers.current.has(event.pointerId)) return;
    pointers.current.delete(event.pointerId);
    if (event.currentTarget.hasPointerCapture(event.pointerId))
      event.currentTarget.releasePointerCapture(event.pointerId);
    if (cancelled) {
      pointers.current.clear();
      clearTap();
    } else if (pointers.current.size) {
      rebaseGesture(true);
      return;
    }
    gesture.current = null;
    commitLive();
    setIsDragging(false);
    setTransitionEnabled(true);
    setDragOffset(0);
    if (cancelled || active.pinched) return;

    const point = { x: event.clientX, y: event.clientY };
    const dx = point.x - active.start.x;
    const dy = point.y - active.start.y;
    if (active.moved || distance(point, active.start) > 5) {
      clearTap();
      const width = containerRef.current?.clientWidth || 0;
      if (active.transform.scale > 1 || Math.abs(dx) <= Math.abs(dy) || Math.abs(dx) <= width * 0.25) return;
      const navigate = dx > 0 ? (options.hasPrevious ? options.onPrev : null) : options.hasNext ? options.onNext : null;
      if (navigate) {
        setDragOffset(dx > 0 ? width : -width);
        slideTimer.current = setTimeout(() => {
          // Keep the destination slide visible and gestures locked while the
          // next media request loads. The media-key effect resets the track.
          navigate();
        }, SLIDE_DURATION);
      }
      return;
    }

    // Pointer capture retargets pointerup to the container, so use the original
    // pointerdown target to distinguish image clicks from background clicks.
    const result = resolveImageTap(
      { time: Date.now(), point, pointerType: event.pointerType, isExpanded, isImage: active.isImage },
      tap.current
    );
    if (result.action === 'open') openViewer(true);
    else if (result.action === 'close') closeZoom(true);
    else if (result.action === 'zoom') toggleZoom(point, event.pointerType !== 'touch');
    tap.current = result.tap;
  };

  return {
    isExpanded,
    openViewer,
    closeZoom,
    containerRef,
    imageRef,
    viewportRef,
    transform,
    dragOffset,
    isDragging,
    transitionEnabled,
    animateZoom,
    toggleZoom,
    pointerHandlers: {
      onPointerDown,
      onPointerMove,
      onPointerUp: (event: ReactPointerEvent<HTMLDivElement>) => endPointer(event),
      onPointerCancel: (event: ReactPointerEvent<HTMLDivElement>) => endPointer(event, true),
      onLostPointerCapture: (event: ReactPointerEvent<HTMLDivElement>) => endPointer(event, true),
    },
  };
}
