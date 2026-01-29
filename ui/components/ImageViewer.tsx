import React, { useState, useRef, useEffect, useCallback, useLayoutEffect } from 'react';
import { MediaItem, ViewMode } from '../types';
import ZoomIn from '../svg/zoom-in.svg?react';
import ZoomOut from '../svg/zoom-out.svg?react';
import Left from '../svg/left.svg?react';
import Right from '../svg/right.svg?react';
import Fullscreen from '../svg/fullscreen.svg?react';
import FullscreenClose from '../svg/fullscreen-close.svg?react';
import Download from '../svg/download.svg?react';

interface ImageViewerProps {
  media: MediaItem;
  previous: MediaItem[];
  next: MediaItem[];
  viewMode: ViewMode;
  onNext: () => void;
  onPrev: () => void;
  onToggleFullscreen: () => void;
}

interface MediaSlideProps {
  item: MediaItem | null;
  isActive: boolean;
  zoomLevel: number;
  pan: { x: number; y: number };
  suppressTransition?: boolean;
  isDragging?: boolean;
}

const MediaSlide: React.FC<MediaSlideProps> = ({ item, isActive, zoomLevel, pan, suppressTransition, isDragging }) => {
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    // Reset loading whenever the item changes
    setLoading(true);
  }, [item?.hash]);

  if (!item) return <div className="text-zinc-700">Empty</div>;

  const isVideo = item.type === 'video';
  const isZoomed = isActive && zoomLevel > 1;
  const videoRef = useRef<HTMLVideoElement | null>(null);

  useEffect(() => {
    if (!isVideo || !item.hash) return;
    if (typeof window === 'undefined') return;

    const Hls = window.Hls || (typeof require !== 'undefined' ? require('hls.js') : null);
    if (Hls && Hls.isSupported && Hls.isSupported() && videoRef.current) {
      const hls = new Hls({
        debug: false,
        maxBufferLength: 3,
      });
      hls.loadSource(`/api/transcode/${item.hash}/index.m3u8`);
      hls.attachMedia(videoRef.current);
      hls.on(Hls.Events.MEDIA_ATTACHED, function () {});
      return () => {
        hls.destroy();
      };
    } else if (videoRef.current && videoRef.current.canPlayType('application/vnd.apple.mpegurl')) {
      videoRef.current.src = `/transcode/${item.hash}/index.m3u8`;
    }
  }, [isVideo, item.hash]);

  const style: React.CSSProperties = isZoomed
    ? {
        transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoomLevel})`,
        cursor: 'move',
      }
    : {
        cursor: isActive ? 'zoom-in' : 'default',
        transform: 'scale(1)',
      };

  const className =
    suppressTransition || (isZoomed && isDragging)
      ? `max-w-full max-h-full object-contain select-none transition-transform duration-0`
      : `max-w-full max-h-full object-contain select-none transition-transform duration-300 ease-in-out`;

  return (
    <div className="relative flex h-full w-full items-center justify-center">
      {isVideo ? (
        <video
          id={`video-${item.hash}`}
          ref={videoRef}
          className={className}
          style={style}
          controls
          onLoadedData={() => setLoading(false)}
          onError={() => setLoading(false)}
        />
      ) : (
        <>
          <img
            srcSet={item.srcset}
            // If zoomed, tell browser we might render at full native width (item.width).
            // If not zoomed, it's fitting in the viewport (100vw).
            sizes={isZoomed && item.width ? `${item.width}px` : '100vw'}
            alt={loading ? '' : item.path}
            className={className}
            style={style}
            draggable={false}
            width={item.width}
            height={item.height}
            onLoad={() => setLoading(false)}
            onError={() => setLoading(false)}
          />
        </>
      )}

      {loading && (
        <div
          className="pointer-events-none absolute inset-0 flex items-center justify-center"
          aria-hidden={false}
          role="status"
        >
          <div className="flex items-center justify-center p-3">
            <svg width="36" height="36" fill="#fff" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
              <path d="M12,1A11,11,0,1,0,23,12,11,11,0,0,0,12,1Zm0,19a8,8,0,1,1,8-8A8,8,0,0,1,12,20Z" opacity=".25" />
              <path
                d="M10.14,1.16a11,11,0,0,0-9,8.92A1.59,1.59,0,0,0,2.46,12,1.52,1.52,0,0,0,4.11,10.7a8,8,0,0,1,6.66-6.61A1.42,1.42,0,0,0,12,2.69h0A1.57,1.57,0,0,0,10.14,1.16Z"
                className="spinner"
              />
            </svg>
          </div>
        </div>
      )}
    </div>
  );
};

const ImageViewer: React.FC<ImageViewerProps> = ({
  media,
  previous,
  next,
  viewMode,
  onNext,
  onPrev,
  onToggleFullscreen,
}) => {
  const [zoomLevel, setZoomLevel] = useState<number>(1);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const [suppressImageTransition, setSuppressImageTransition] = useState(false);

  const [dragOffset, setDragOffset] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [transitionEnabled, setTransitionEnabled] = useState(false);

  const containerRef = useRef<HTMLDivElement>(null);
  const dragStartX = useRef<number | null>(null);
  const dragStartPan = useRef<{ x: number; y: number; y_start?: number }>({ x: 0, y: 0 });
  // We need to track the initial touch/mouse position to calculate total distance moved
  // to differentiate between a click and a drag.
  const initialClientPos = useRef({ x: 0, y: 0 });

  // Click handling
  const clickTimeoutRef = useRef<number | null>(null);
  const lastTapTimeRef = useRef<number>(0);
  const lastTapPosRef = useRef({ x: 0, y: 0 });
  const ignoreMouseUntilRef = useRef<number>(0);

  const prevItem = previous && previous.length > 0 ? previous[previous.length - 1] : null;
  const nextItem = next && next.length > 0 ? next[0] : null;

  // prevent flash of the next image at the wrong offset.
  useLayoutEffect(() => {
    setZoomLevel(1);
    setPan({ x: 0, y: 0 });
    setDragOffset(0);
    setTransitionEnabled(false);
    setIsDragging(false);
  }, [media.hash]);

  // Sync Zoom state with ViewMode: If we exit fullscreen, we must zoom out.
  useEffect(() => {
    if (viewMode === ViewMode.NORMAL) {
      setZoomLevel(1);
      setPan({ x: 0, y: 0 });
      setSuppressImageTransition(true);
      window.setTimeout(() => setSuppressImageTransition(false), 50);
    }
  }, [viewMode]);

  const toggleZoom = useCallback(() => {
    if (zoomLevel === 1) {
      setZoomLevel(2.5); // Value > 1 triggers "Zoomed" state in MediaSlide
      // Automatically enter fullscreen when zooming in
      if (viewMode !== ViewMode.FULLSCREEN) {
        onToggleFullscreen();
      }
    } else {
      setZoomLevel(1);
      setPan({ x: 0, y: 0 });
      // Exit fullscreen when zooming out
      if (viewMode === ViewMode.FULLSCREEN) {
        onToggleFullscreen();
      }
    }
  }, [zoomLevel, viewMode, onToggleFullscreen]);

  const onMouseDown = (e: React.MouseEvent) => {
    if (Date.now() < ignoreMouseUntilRef.current) return;
    if (e.button !== 0) return;

    e.preventDefault();
    dragStartX.current = e.clientX;
    initialClientPos.current = { x: e.clientX, y: e.clientY };
    dragStartPan.current.y_start = e.clientY;

    // Store initial pan state for delta calculation
    if (zoomLevel > 1) {
      dragStartPan.current.x = pan.x;
      dragStartPan.current.y = pan.y;
    }

    setIsDragging(true);
    setTransitionEnabled(false);
  };

  const onMouseMove = (e: React.MouseEvent) => {
    if (!isDragging || dragStartX.current === null) return;
    const dx = e.clientX - dragStartX.current;
    const dy = e.clientY - (dragStartPan.current.y_start || 0);

    if (zoomLevel > 1) {
      setPan({
        x: dragStartPan.current.x + dx,
        y: dragStartPan.current.y + dy,
      });
    } else {
      setDragOffset(dx);
    }
  };

  const onMouseUp = (e: React.MouseEvent) => {
    if (Date.now() < ignoreMouseUntilRef.current) return;
    if (e.button !== 0) return;
    handleEnd(e.clientX, e.clientY);
  };

  const onTouchStart = (e: React.TouchEvent) => {
    const touch = e.touches[0];
    dragStartX.current = touch.clientX;
    initialClientPos.current = { x: touch.clientX, y: touch.clientY };
    dragStartPan.current.y_start = touch.clientY;

    if (zoomLevel > 1) {
      dragStartPan.current.x = pan.x;
      dragStartPan.current.y = pan.y;
    }

    setIsDragging(true);
    setTransitionEnabled(false);
  };

  const onTouchMove = (e: React.TouchEvent) => {
    if (!isDragging || dragStartX.current === null) return;
    const touch = e.touches[0];
    const dx = touch.clientX - dragStartX.current;
    const dy = touch.clientY - (dragStartPan.current.y_start || 0);

    if (zoomLevel > 1) {
      setPan({
        x: dragStartPan.current.x + dx,
        y: dragStartPan.current.y + dy,
      });
      e.preventDefault();
    } else {
      setDragOffset(dx);
    }
  };

  const onTouchEnd = (e: React.TouchEvent) => {
    const touch = e.changedTouches[0];
    const clientX = touch.clientX;
    const clientY = touch.clientY;

    // Record a short period to ignore mouse events synthesized from this touch
    ignoreMouseUntilRef.current = Date.now() + 400;

    // Double-tap detection: time + distance
    const now = Date.now();
    const timeSinceLastTap = now - lastTapTimeRef.current;
    const dx = clientX - lastTapPosRef.current.x;
    const dy = clientY - lastTapPosRef.current.y;
    const dist = Math.sqrt(dx * dx + dy * dy);
    const isDoubleTap = timeSinceLastTap > 0 && timeSinceLastTap < 300 && dist < 30;

    if (isDoubleTap) {
      // clear pending single-tap handler
      if (clickTimeoutRef.current) {
        clearTimeout(clickTimeoutRef.current);
        clickTimeoutRef.current = null;
      }
      toggleZoom();
      lastTapTimeRef.current = 0;
      lastTapPosRef.current = { x: 0, y: 0 };
      // don't treat this as a drag end
      setIsDragging(false);
      dragStartX.current = null;
      return;
    }

    // not a double tap
    handleEnd(clientX, clientY);
  };

  const handleEnd = (clientX: number, clientY: number) => {
    if (!isDragging || !containerRef.current) return;
    setIsDragging(false);

    // Calculate total movement distance to distinguish Click vs Drag
    const dist = Math.sqrt(
      Math.pow(clientX - initialClientPos.current.x, 2) + Math.pow(clientY - initialClientPos.current.y, 2)
    );

    const isDrag = dist > 5;

    if (!isDrag) {
      // Handle Click / Tap
      const now = Date.now();
      const timeSinceLastTap = now - lastTapTimeRef.current;

      if (timeSinceLastTap < 300) {
        // double tap
        if (clickTimeoutRef.current) {
          clearTimeout(clickTimeoutRef.current);
          clickTimeoutRef.current = null;
        }
        toggleZoom();
        lastTapTimeRef.current = 0; // Reset
      } else {
        // single tap
        lastTapTimeRef.current = now;
        lastTapPosRef.current = { x: clientX, y: clientY };

        if (zoomLevel === 1) {
          if (viewMode === ViewMode.FULLSCREEN) {
            // In fullscreen, wait to distinguish from double-tap (zoom)
            // This prevents exiting fullscreen just to zoom back in
            clickTimeoutRef.current = window.setTimeout(() => {
              onToggleFullscreen();
              clickTimeoutRef.current = null;
            }, 300);
          } else {
            // In normal mode, toggle immediately (no delay)
            onToggleFullscreen();
          }
        }
      }

      setDragOffset(0);
      dragStartX.current = null;
      return;
    }

    // drag end
    if (zoomLevel > 1) {
      dragStartPan.current = { ...pan };
    } else {
      const rect = containerRef.current.getBoundingClientRect();
      const style = window.getComputedStyle(containerRef.current);
      const borderX = (parseFloat(style.borderLeftWidth) || 0) + (parseFloat(style.borderRightWidth) || 0);
      const width = rect.width - borderX;

      const threshold = width * 0.25;

      setTransitionEnabled(true);

      if (dragOffset > threshold && prevItem) {
        // Swipe Right -> Prev
        setDragOffset(width);
        setTimeout(onPrev, 300);
      } else if (dragOffset < -threshold && nextItem) {
        // Swipe Left -> Next
        setDragOffset(-width);
        setTimeout(onNext, 300);
      } else {
        // Do not slide
        setDragOffset(0);
      }
    }

    dragStartX.current = null;
  };

  const containerStyle: React.CSSProperties =
    viewMode === ViewMode.FULLSCREEN
      ? {
          width: '100vw',
          height: '100vh',
          position: 'relative',
          top: 0,
          left: 0,
          zIndex: 50,
          transition: 'all 200ms',
        }
      : {
          width: '100%',
          height: '75vh',
          position: 'relative',
          transition: 'all 200ms',
        };

  const containerClass = `group overflow-hidden select-none transition-colors duration-300 ease-in-out ${
    viewMode === ViewMode.FULLSCREEN ? 'bg-black' : 'mx-auto bg-black/0'
  }`;

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key.toLowerCase() === 'z') {
        toggleZoom();
      }
    };
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, [toggleZoom]);

  const slideClass =
    viewMode === ViewMode.FULLSCREEN
      ? `relative flex h-full w-1/3 items-center justify-center overflow-hidden`
      : `relative flex h-full w-1/3 items-center justify-center overflow-hidden lg:px-4`;

  return (
    <div
      ref={containerRef}
      className={containerClass}
      style={containerStyle}
      onMouseDown={onMouseDown}
      onMouseMove={onMouseMove}
      onMouseUp={onMouseUp}
      onMouseLeave={onMouseUp}
      onTouchStart={onTouchStart}
      onTouchMove={onTouchMove}
      onTouchEnd={onTouchEnd}
    >
      {/* Slider track */}
      <div
        className="flex h-full"
        style={{
          width: '300%',
          marginLeft: '-100%',
          transform: `translate3d(${dragOffset}px, 0, 0)`,
          transition: transitionEnabled ? 'transform 0.3s cubic-bezier(0.2, 0.8, 0.2, 1)' : 'none',
          willChange: 'transform',
        }}
      >
        {/* Previous slide */}
        <div key={prevItem ? prevItem.hash : 'prev-placeholder'} className={slideClass}>
          <MediaSlide item={prevItem} isActive={false} zoomLevel={1} pan={{ x: 0, y: 0 }} />
        </div>

        {/* Current slide */}
        <div key={media.hash} className={slideClass}>
          <MediaSlide
            item={media}
            isActive={true}
            zoomLevel={zoomLevel}
            pan={pan}
            suppressTransition={suppressImageTransition}
            isDragging={isDragging}
          />
        </div>

        {/* Next slide */}
        <div key={nextItem ? nextItem.hash : 'next-placeholder'} className={slideClass}>
          <MediaSlide item={nextItem} isActive={false} zoomLevel={1} pan={{ x: 0, y: 0 }} />
        </div>
      </div>

      {/* Navigation arrows */}
      {!isDragging && zoomLevel === 1 && (
        <>
          <button
            onMouseDown={(e) => e.stopPropagation()}
            onMouseUp={(e) => e.stopPropagation()}
            onTouchStart={(e) => e.stopPropagation()}
            onTouchEnd={(e) => e.stopPropagation()}
            onClick={(e) => {
              e.stopPropagation();
              onPrev();
            }}
            className="absolute top-1/2 left-4 z-20 -translate-y-1/2 cursor-pointer rounded-full p-4 text-white/70 transition-all outline-none hover:text-white"
          >
            <Left className="h-8 w-8" />
          </button>
          <button
            onMouseDown={(e) => e.stopPropagation()}
            onMouseUp={(e) => e.stopPropagation()}
            onTouchStart={(e) => e.stopPropagation()}
            onTouchEnd={(e) => e.stopPropagation()}
            onClick={(e) => {
              e.stopPropagation();
              onNext();
            }}
            className="absolute top-1/2 right-4 z-20 -translate-y-1/2 cursor-pointer rounded-full p-4 text-white/70 transition-all outline-none hover:text-white"
          >
            <Right className="h-8 w-8" />
          </button>
        </>
      )}

      {/* Toolbar */}
      <div className="absolute top-4 right-4 z-30 flex items-center gap-3 opacity-0 transition-opacity duration-300 group-hover:opacity-100">
        <a
          href={`/api/media-originals/${media.path}`}
          download
          target="_blank"
          rel="noopener noreferrer"
          onMouseDown={(e) => e.stopPropagation()}
          onMouseUp={(e) => e.stopPropagation()}
          onTouchStart={(e) => e.stopPropagation()}
          onTouchEnd={(e) => e.stopPropagation()}
          onClick={(e) => e.stopPropagation()}
          className="flex items-center justify-center rounded-lg border border-zinc-300 bg-zinc-200 p-2.5 text-black shadow-lg backdrop-blur-md transition-colors hover:bg-zinc-300 dark:border-white/10 dark:bg-black/50 dark:text-white dark:hover:bg-white/10"
          title="Download Original"
        >
          <Download />
        </a>
        <button
          onMouseDown={(e) => e.stopPropagation()}
          onMouseUp={(e) => e.stopPropagation()}
          onTouchStart={(e) => e.stopPropagation()}
          onTouchEnd={(e) => e.stopPropagation()}
          onClick={(e) => {
            e.stopPropagation();
            toggleZoom();
          }}
          className="rounded-lg border border-zinc-300 bg-zinc-200 p-2.5 text-black shadow-lg backdrop-blur-md transition-colors hover:bg-zinc-300 dark:border-white/10 dark:bg-black/50 dark:text-white dark:hover:bg-white/10"
          title="Zoom (Z)"
        >
          {zoomLevel > 1 ? <ZoomOut /> : <ZoomIn />}
        </button>
        <button
          onMouseDown={(e) => e.stopPropagation()}
          onMouseUp={(e) => e.stopPropagation()}
          onTouchStart={(e) => e.stopPropagation()}
          onTouchEnd={(e) => e.stopPropagation()}
          onClick={(e) => {
            e.stopPropagation();
            onToggleFullscreen();
          }}
          className="rounded-lg border border-zinc-300 bg-zinc-200 p-2.5 text-black shadow-lg backdrop-blur-md transition-colors hover:bg-zinc-300 dark:border-white/10 dark:bg-black/50 dark:text-white dark:hover:bg-white/10"
          title="Fullscreen (F)"
        >
          {viewMode === ViewMode.FULLSCREEN ? <FullscreenClose /> : <Fullscreen />}
        </button>
      </div>
    </div>
  );
};

export default ImageViewer;
