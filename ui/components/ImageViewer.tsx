import React, { useState, useRef, useEffect } from 'react';
import { useImageGestures } from '../hooks/useImageGestures';
import { MediaItem } from '../types';
import { useViewerOverlay } from '../hooks/useViewerOverlay';
import Fullscreen from '../svg/fullscreen.svg?react';
import Close from '../svg/close.svg?react';
import Left from '../svg/left.svg?react';
import Right from '../svg/right.svg?react';
import Download from '../svg/download.svg?react';

interface ImageViewerProps {
  media: MediaItem;
  previous: MediaItem[];
  next: MediaItem[];
  onNext: () => void;
  onPrev: () => void;
}

interface MediaSlideProps {
  item: MediaItem | null;
  isActive: boolean;
  zoomLevel: number;
  pan: { x: number; y: number };
  suppressTransition?: boolean;
  isDragging?: boolean;
  imageRef?: React.Ref<HTMLImageElement>;
  viewportRef?: React.Ref<HTMLDivElement>;
}

const MediaSlide: React.FC<MediaSlideProps> = ({
  item,
  isActive,
  zoomLevel,
  pan,
  suppressTransition,
  isDragging,
  imageRef,
  viewportRef,
}) => {
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    // Reset loading whenever the item changes
    setLoading(true);
  }, [item?.hash]);

  const isVideo = item?.type === 'video';
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
  }, [isVideo, item?.hash]);

  if (!item) return null;

  const style: React.CSSProperties = isZoomed
    ? {
        transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoomLevel})`,
        cursor: isDragging ? 'grabbing' : 'zoom-out',
      }
    : {
        cursor: isActive && !isVideo ? 'zoom-in' : 'default',
        transform: 'scale(1)',
      };

  const className =
    suppressTransition || (isZoomed && isDragging)
      ? `w-auto h-auto max-w-full max-h-full object-contain select-none transition-transform duration-0`
      : `w-auto h-auto max-w-full max-h-full object-contain select-none transition-transform duration-200 ease-out motion-reduce:transition-none`;

  return (
    <div ref={viewportRef} inert={!isActive} className="relative flex h-full w-full items-center justify-center">
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
            ref={imageRef}
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

const ImageViewer: React.FC<ImageViewerProps> = ({ media, previous, next, onNext, onPrev }) => {
  const prevItem = previous.length > 0 ? previous[previous.length - 1] : null;
  const nextItem = next.length > 0 ? next[0] : null;
  const {
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
    pointerHandlers,
  } = useImageGestures({
    mediaKey: media.hash,
    nativeWidth: media.width,
    isVideo: media.type === 'video',
    hasPrevious: !!prevItem,
    hasNext: !!nextItem,
    onPrev,
    onNext,
  });
  const { scale: zoomLevel, pan } = transform;
  useViewerOverlay(isExpanded, containerRef);

  const containerStyle: React.CSSProperties = isExpanded
    ? {
        width: '100vw',
        height: '100dvh',
        position: 'fixed',
        top: 0,
        left: 0,
        zIndex: 100,
        touchAction: 'none',
      }
    : {
        width: '100%',
        height: '75vh',
        position: 'relative',
        touchAction: 'none',
      };

  // Clipping must not create a scroll container: focusing controls after a
  // resize can otherwise scroll the entire slide track away from its center.
  const containerClass = `group overflow-clip select-none outline-none transition-colors duration-300 ease-in-out ${
    isExpanded ? 'bg-black' : 'mx-auto bg-black/0'
  }`;

  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isExpanded) {
        e.preventDefault();
        closeZoom(true);
        return;
      }
      if ((e.target as HTMLElement).closest('input, textarea, select, [contenteditable="true"]')) return;
      if (e.key.toLowerCase() === 'z' && !e.ctrlKey && !e.metaKey && !e.altKey && !e.repeat) {
        e.preventDefault();
        toggleZoom(undefined, true);
      }
    };
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, [isExpanded, closeZoom, toggleZoom]);

  const slideClass = isExpanded
    ? `relative flex h-full w-1/3 items-center justify-center overflow-clip`
    : `relative flex h-full w-1/3 items-center justify-center overflow-clip lg:px-4`;

  return (
    <div className="h-[75vh] w-full">
      <div
        ref={containerRef}
        className={containerClass}
        style={containerStyle}
        role={isExpanded ? 'dialog' : undefined}
        aria-modal={isExpanded ? true : undefined}
        aria-label={isExpanded ? 'Media viewer' : undefined}
        tabIndex={-1}
        {...pointerHandlers}
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
              suppressTransition={!animateZoom}
              imageRef={imageRef}
              viewportRef={viewportRef}
              isDragging={isDragging}
            />
          </div>

          {/* Next slide */}
          <div key={nextItem ? nextItem.hash : 'next-placeholder'} className={slideClass}>
            <MediaSlide item={nextItem} isActive={false} zoomLevel={1} pan={{ x: 0, y: 0 }} />
          </div>
        </div>

        {/* Navigation arrows */}
        {!isDragging && (
          <>
            <button
              aria-label="Previous image"
              disabled={!prevItem}
              onClick={(e) => {
                e.stopPropagation();
                onPrev();
              }}
              className="absolute top-1/2 left-4 z-20 -translate-y-1/2 cursor-pointer rounded-full p-4 text-white/70 transition-all outline-none hover:text-white focus-visible:ring-2 focus-visible:ring-white disabled:hidden"
            >
              <Left className="h-8 w-8" />
            </button>
            <button
              aria-label="Next image"
              disabled={!nextItem}
              onClick={(e) => {
                e.stopPropagation();
                onNext();
              }}
              className="absolute top-1/2 right-4 z-20 -translate-y-1/2 cursor-pointer rounded-full p-4 text-white/70 transition-all outline-none hover:text-white focus-visible:ring-2 focus-visible:ring-white disabled:hidden"
            >
              <Right className="h-8 w-8" />
            </button>
          </>
        )}

        {/* Toolbar */}
        <div
          data-zoom-toolbar
          className={`absolute top-4 right-4 z-30 flex items-center gap-3 opacity-100 transition-opacity duration-300 ${
            isExpanded
              ? ''
              : 'md:opacity-0 md:group-hover:opacity-100 md:focus-within:opacity-100 [@media(hover:none)]:opacity-100'
          }`}
        >
          <a
            href={`/api/media-originals/${media.path}`}
            download
            target="_blank"
            rel="noopener noreferrer"
            onClick={(e) => e.stopPropagation()}
            className="flex items-center justify-center rounded-lg border border-zinc-300 bg-zinc-200 p-2.5 text-black shadow-lg backdrop-blur-md transition-colors hover:bg-zinc-300 dark:border-white/10 dark:bg-black/50 dark:text-white dark:hover:bg-white/10"
            aria-label="Download original"
            title="Download Original"
          >
            <Download />
          </a>
          <button
            onClick={(e) => {
              e.stopPropagation();
              if (isExpanded) closeZoom(true);
              else openViewer(true);
            }}
            className="rounded-lg border border-zinc-300 bg-zinc-200 p-2.5 text-black shadow-lg backdrop-blur-md transition-colors hover:bg-zinc-300 dark:border-white/10 dark:bg-black/50 dark:text-white dark:hover:bg-white/10"
            disabled={media.type === 'video' && !isExpanded}
            data-zoom-control
            aria-label={isExpanded ? 'Close viewer' : 'Open fullscreen'}
            aria-expanded={isExpanded}
            title={isExpanded ? 'Close viewer (Esc)' : 'Open fullscreen (Z)'}
          >
            {isExpanded ? <Close className="h-6 w-6" /> : <Fullscreen className="h-6 w-6" />}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ImageViewer;
