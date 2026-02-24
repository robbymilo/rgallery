import React, { useRef, useState, useLayoutEffect, useMemo, useEffect, useCallback } from 'react';
import VideoThumb from './VideoThumb';
import { Link } from 'react-router-dom';
import ArrowUp from '../svg/arrow-up.svg?react';
import { Photo, NodeType, PhotoRowNode, DateHeaderNode, VirtualItem } from '../types';
import justifiedLayout from 'justified-layout';

interface VirtualGridProps {
  photos: Photo[];
  onEndReached: () => void;
  onStartReached: () => void;
  onVisibleDateChange?: (date: Date) => void;
  onScrollToTop: () => void;
  isLoading: boolean;
  scrollToTarget?: { date: Date; timestamp: number } | null;
  hasMoreItems: boolean;
  filters: Record<string, string>;
}

interface LayoutState {
  items: VirtualItem[];
  totalHeight: number;
}

const DATE_HEADER_HEIGHT = 40;
const ROW_GAP = 4;

const groupByDate = (photos: Photo[]): Record<string, Photo[]> => {
  const groups: Record<string, Photo[]> = {};
  for (const photo of photos) {
    const key = photo.date.toISOString().split('T')[0];
    if (!groups[key]) {
      groups[key] = [];
    }
    groups[key].push(photo);
  }
  return groups;
};

const computeLayout = (photos: Photo[], containerWidth: number, targetRowHeight: number): LayoutState => {
  if (containerWidth <= 0) return { items: [], totalHeight: 0 };

  const groups = groupByDate(photos);
  // Sort dates descending
  const sortedDates = Object.keys(groups).sort((a, b) => new Date(b).getTime() - new Date(a).getTime());

  const items: VirtualItem[] = [];
  let currentTop = 0;

  for (const dateKey of sortedDates) {
    const groupPhotos = groups[dateKey];

    // add date header
    items.push({
      type: NodeType.DATE_HEADER,
      id: `header-${dateKey}`,
      date: dateKey,
      dateObj: new Date(dateKey),
      height: DATE_HEADER_HEIGHT,
      top: currentTop,
    });
    currentTop += DATE_HEADER_HEIGHT;

    // compute layout for this group
    const aspectRatios = groupPhotos.map((p) => p.aspectRatio);

    const geometry = justifiedLayout(aspectRatios, {
      containerWidth: containerWidth,
      targetRowHeight: targetRowHeight,
      boxSpacing: ROW_GAP,
      containerPadding: 0,
      showWidows: true,
    });

    // convert boxes to rows
    let currentRowTop = -1;
    // Updated type for currentRowBoxes
    let currentRowBoxes: { width: number; height: number; top: number; left: number }[] = [];
    let currentRowPhotos: Photo[] = [];

    const finalizeRow = () => {
      if (currentRowBoxes.length === 0) return;

      const rowHeight = currentRowBoxes[0].height;
      const layoutWidths = currentRowBoxes.map((b) => b.width);

      items.push({
        type: NodeType.PHOTO_ROW,
        id: `row-${dateKey}-${currentRowPhotos[0].id}`,
        photos: [...currentRowPhotos],
        rowHeight: rowHeight,
        layoutWidths: layoutWidths,
        height: rowHeight + ROW_GAP,
        top: currentTop,
      });
      currentTop += rowHeight + ROW_GAP;
    };

    // Updated type for geometry.boxes
    geometry.boxes.forEach((box: { width: number; height: number; top: number; left: number }, index: number) => {
      // detect new row by checking if top position changes
      if (currentRowTop === -1 || Math.abs(box.top - currentRowTop) > 5) {
        finalizeRow();

        currentRowTop = box.top;
        currentRowBoxes = [];
        currentRowPhotos = [];
      }

      currentRowBoxes.push(box);
      currentRowPhotos.push(groupPhotos[index]);
    });

    // finalize last row
    finalizeRow();
  }

  return { items, totalHeight: currentTop };
};

export const VirtualGrid: React.FC<VirtualGridProps> = ({
  photos,
  onEndReached,
  onStartReached,
  onVisibleDateChange,
  onScrollToTop,
  isLoading,
  scrollToTarget,
  hasMoreItems,
  filters,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [containerWidth, setContainerWidth] = useState(0);
  const [scrollTop, setScrollTop] = useState(0);
  const lastReportedDateRef = useRef<string | null>(null);

  // Refs for scroll anchoring and processing
  const prevFirstPhotoIdRef = useRef<string | null>(null);
  const lastProcessedTimestampRef = useRef<number | null>(null);
  const pendingScrollRetryRef = useRef<boolean>(false);
  const loadStartCountRef = useRef<number | null>(null);
  const attemptedLoadRef = useRef<boolean>(false);
  const noMoreTimelineRef = useRef<boolean>(false);

  // measure container width
  useLayoutEffect(() => {
    console.log('[VirtualGrid] Measuring container width');
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setContainerWidth(entry.contentRect.width);
      }
    });
    if (containerRef.current) {
      observer.observe(containerRef.current);
    }
    return () => observer.disconnect();
  }, []);

  // grid width
  const gridWidth = containerWidth < 768 ? Math.floor(containerWidth * 0.9) : Math.floor(containerWidth * 0.8);

  const targetRowHeight = containerWidth < 768 ? 120 : 150;

  // compute layout
  const { items, totalHeight: computedTotalHeight } = useMemo(() => {
    return computeLayout(photos, gridWidth, targetRowHeight);
  }, [photos, gridWidth, targetRowHeight]);

  // final padding
  const BOTTOM_PADDING = 120;
  const totalHeight = computedTotalHeight + BOTTOM_PADDING;

  // scroll
  useLayoutEffect(() => {
    if (!containerRef.current || items.length === 0) return;
    console.log('[VirtualGrid] Handling scroll adjustments');

    let didProgrammaticScroll = false;

    // jump to date
    const isNewRequest = scrollToTarget && scrollToTarget.timestamp !== lastProcessedTimestampRef.current;
    const isRetry =
      scrollToTarget && pendingScrollRetryRef.current && scrollToTarget.timestamp === lastProcessedTimestampRef.current;

    if (isNewRequest || isRetry) {
      const dateStr = scrollToTarget!.date.toISOString().split('T')[0];
      console.log(
        '[VirtualGrid] Scroll request for date:',
        dateStr,
        'isNewRequest:',
        isNewRequest,
        'isRetry:',
        isRetry
      );

      // find the header for this exact date
      let targetNode = items.find((i) => i.type === NodeType.DATE_HEADER && (i as DateHeaderNode).date === dateStr);

      console.log('[VirtualGrid] Found exact date header:', !!targetNode);

      // if header not found, find the closest date
      if (!targetNode) {
        const targetTime = scrollToTarget!.date.getTime();
        let olderHeader: DateHeaderNode | undefined;
        let newerHeader: DateHeaderNode | undefined;

        for (const item of items) {
          if (item.type === NodeType.DATE_HEADER) {
            const header = item as DateHeaderNode;
            if (header.dateObj.getTime() <= targetTime) {
              olderHeader = header;
              break;
            }
            newerHeader = header;
          }
        }

        if (olderHeader && newerHeader) {
          const olderDiff = Math.abs(olderHeader.dateObj.getTime() - targetTime);
          const newerDiff = Math.abs(newerHeader.dateObj.getTime() - targetTime);
          targetNode = newerDiff < olderDiff ? newerHeader : olderHeader;
          console.log(
            `[VirtualGrid] Closest match: ${targetNode === newerHeader ? 'Newer' : 'Older'} (${
              (targetNode as DateHeaderNode).date
            })`
          );
        } else if (olderHeader) {
          targetNode = olderHeader;
          console.log('[VirtualGrid] Only older match found:', olderHeader.date);
        } else if (newerHeader) {
          targetNode = newerHeader;
          console.log('[VirtualGrid] Only newer match found:', newerHeader.date);
        } else {
          // Fallback to first item if nothing found (shouldn't happen if list not empty)
          targetNode = items[0];
        }
      }

      if (targetNode) {
        console.log('[VirtualGrid] Target node type:', targetNode.type, 'top:', targetNode.top);

        const offset = 0;
        const targetTop = Math.max(0, targetNode.top - offset);
        containerRef.current.scrollTop = targetTop;
        console.log('[VirtualGrid] Set scrollTop to:', targetTop, 'actual scrollTop:', containerRef.current.scrollTop);

        if (Math.abs(containerRef.current.scrollTop - targetTop) > 2) {
          pendingScrollRetryRef.current = true;
          console.log('[VirtualGrid] Scroll clamped, marking for retry');
        } else {
          pendingScrollRetryRef.current = false;
        }

        if (isNewRequest) {
          lastProcessedTimestampRef.current = scrollToTarget!.timestamp;
          // Reset anchor logic
          const firstPhotoItem = items.find(
            (i) => i.type === NodeType.PHOTO_ROW && (i as PhotoRowNode).photos.length > 0
          ) as PhotoRowNode | undefined;
          prevFirstPhotoIdRef.current = firstPhotoItem?.photos[0]?.id || null;
        }

        didProgrammaticScroll = true;
      }
    }

    console.log('[VirtualGrid] Did programmatic scroll:', didProgrammaticScroll);

    if (didProgrammaticScroll) return;

    // maintain position during prepend
    const firstPhotoItem = items.find((i) => i.type === NodeType.PHOTO_ROW && (i as PhotoRowNode).photos.length > 0) as
      | PhotoRowNode
      | undefined;
    const currentFirstId = firstPhotoItem?.photos[0]?.id || null;

    if (prevFirstPhotoIdRef.current && currentFirstId !== prevFirstPhotoIdRef.current) {
      const anchorItem = items.find(
        (i) =>
          i.type === NodeType.PHOTO_ROW && (i as PhotoRowNode).photos.some((p) => p.id === prevFirstPhotoIdRef.current)
      );

      if (anchorItem) {
        // prevent jump
        containerRef.current.scrollTop += anchorItem.top;
      }
    }

    console.log('[VirtualGrid] Updated scroll anchoring from', prevFirstPhotoIdRef.current, 'to', currentFirstId);

    prevFirstPhotoIdRef.current = currentFirstId;
  }, [items, scrollToTarget]);

  // handle scroll
  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    if (containerWidth === 0) return;

    const { scrollTop, scrollHeight, clientHeight } = e.currentTarget;
    setScrollTop(scrollTop);

    const threshold = clientHeight * 1.5;

    if (scrollHeight - scrollTop - clientHeight < threshold && !isLoading && !noMoreTimelineRef.current) {
      onEndReached();
      attemptedLoadRef.current = true;
      loadStartCountRef.current = photos.length;
    }

    if (scrollTop < threshold && scrollTop > 0 && !isLoading) {
      onStartReached();
    }
  };

  // visibility calculation
  const findStartIndex = (offset: number) => {
    let low = 0;
    let high = items.length - 1;
    let result = 0;

    while (low <= high) {
      const mid = Math.floor((low + high) / 2);
      const item = items[mid];

      if (item.top + item.height > offset) {
        result = mid;
        high = mid - 1;
      } else {
        low = mid + 1;
      }
    }
    return result;
  };

  const visibleStartIndex = findStartIndex(scrollTop);
  let visibleEndIndex = visibleStartIndex;
  const viewportHeight = containerRef.current?.clientHeight || 1000;
  const renderHorizon = scrollTop + viewportHeight;

  for (let i = visibleStartIndex; i < items.length; i++) {
    if (items[i].top > renderHorizon) {
      break;
    }
    visibleEndIndex = i;
  }

  const renderStart = Math.max(0, visibleStartIndex - 3);
  const renderEnd = Math.min(items.length, visibleEndIndex + 4);
  const visibleItems = items.slice(renderStart, renderEnd);

  // get current date
  const currentHeaderDate = useMemo(() => {
    let idx = visibleStartIndex;
    while (idx >= 0 && idx < items.length) {
      if (items[idx].type === NodeType.DATE_HEADER) {
        return (items[idx] as DateHeaderNode).dateObj;
      }
      idx--;
    }
    return null;
  }, [items, visibleStartIndex]);

  // persist current date
  useEffect(() => {
    if (!onVisibleDateChange || !currentHeaderDate) return;

    console.log('[VirtualGrid] Visible date changed to', currentHeaderDate);

    const dateStr = currentHeaderDate.toISOString().split('T')[0];
    if (dateStr !== lastReportedDateRef.current) {
      lastReportedDateRef.current = dateStr;
      onVisibleDateChange(currentHeaderDate);
    }
  }, [currentHeaderDate, onVisibleDateChange]);

  const setItemRef = useCallback((id: string, el: HTMLDivElement | HTMLAnchorElement | null) => {}, []);

  // retry loading
  useEffect(() => {
    // if a load just finished, detect whether it added any new items
    if (!isLoading && attemptedLoadRef.current) {
      attemptedLoadRef.current = false;
      const prevCount = loadStartCountRef.current ?? 0;
      const newCount = photos.length;
      if (newCount === prevCount) {
        // final cursor
        noMoreTimelineRef.current = true;
        console.log('[VirtualGrid] final cursor call');
        return;
      }
      // reset for more items
      loadStartCountRef.current = null;
    }

    if (!isLoading && containerRef.current && hasMoreItems && !noMoreTimelineRef.current) {
      const { scrollTop, scrollHeight, clientHeight } = containerRef.current;
      const threshold = clientHeight * 1.5;
      if (scrollHeight - scrollTop - clientHeight < threshold) {
        console.log('[VirtualGrid] isLoading finished, triggering onEndReached check');
        onEndReached();
        attemptedLoadRef.current = true;
        loadStartCountRef.current = photos.length;
      }
    }
  }, [isLoading, onEndReached, hasMoreItems]);

  return (
    <>
      <div
        ref={containerRef}
        className="no-scrollbar dark:bg-charcoal-900 relative w-full overflow-y-auto bg-gray-50"
        onScroll={handleScroll}
        style={{
          height: 'calc(100vh - 80px)',
          contain: 'strict',
          overflowAnchor: 'none',
        }}
      >
        <div style={{ height: totalHeight, position: 'relative' }}>
          <div style={{ width: gridWidth, margin: '0 auto', position: 'relative', height: '100%' }}>
            {visibleItems.map((item) => {
              if (item.type === NodeType.DATE_HEADER) {
                const headerNode = item as DateHeaderNode;
                return (
                  <div
                    key={item.id}
                    className="absolute top-0 flex w-full items-center"
                    style={{
                      transform: `translateY(${item.top}px)`,
                      height: item.height,
                      zIndex: 20,
                    }}
                  >
                    <div className="flex items-baseline gap-3">
                      <h2
                        className="pt-2 tracking-tight text-gray-900 opacity-80 dark:text-white dark:opacity-90"
                        title={headerNode.dateObj.toISOString()}
                      >
                        {headerNode.dateObj.toUTCString().slice(0, 4)} {headerNode.dateObj.getFullYear()}{' '}
                        {headerNode.dateObj.toLocaleDateString('en-US', { month: 'long', day: 'numeric' })}
                      </h2>
                    </div>
                  </div>
                );
              } else {
                const row = item as PhotoRowNode;
                return (
                  <div
                    key={item.id}
                    className="absolute top-0 right-0 left-0 flex w-full gap-1 px-0"
                    style={{
                      transform: `translateY(${item.top}px)`,
                      height: row.rowHeight,
                    }}
                  >
                    {row.photos.map((photo, index) => (
                      <Link
                        to={`/media/${photo.id}?${new URLSearchParams(filters).toString()}`}
                        key={photo.id}
                        ref={(el) => setItemRef(photo.id, el)}
                        className="group relative overflow-hidden"
                        style={{
                          width: row.layoutWidths[index],
                          height: '100%',
                          backgroundColor: photo.color,
                        }}
                      >
                        {photo.type === 'image' ? (
                          <img
                            srcSet={photo.url}
                            alt={photo.id}
                            loading="lazy"
                            className="block h-full w-full opacity-0 transition-opacity duration-500"
                            onLoad={(e) => (e.currentTarget.style.opacity = '1')}
                          />
                        ) : photo.type === 'video' ? (
                          <VideoThumb
                            hlsUrl={`/api/transcode/${photo.id}/index.m3u8`}
                            poster={`/api/img/${photo.id}/800`}
                            alt={photo.id}
                          />
                        ) : null}
                        <div
                          className={`pointer-events-none absolute inset-0 bg-black/0 transition-colors ${
                            photo.type === 'video' ? '' : 'group-hover:bg-black/20'
                          }`}
                        />
                      </Link>
                    ))}
                  </div>
                );
              }
            })}
            <div
              aria-hidden
              className="absolute right-0 left-0"
              style={{
                height: BOTTOM_PADDING,
                top: computedTotalHeight,
              }}
            />
          </div>
        </div>
      </div>

      {/* Scroll to top */}
      <button
        onClick={onScrollToTop}
        className="fixed bottom-8 left-8 z-50 flex cursor-pointer items-center justify-center rounded-full bg-white p-4 text-black shadow-lg"
        aria-label="Scroll to top"
        title="Scroll to top"
      >
        <ArrowUp className="h-6 w-6" />
      </button>
      {isLoading && (
        <div className="pointer-events-none fixed right-0 bottom-8 left-0 z-50 flex justify-center py-4">
          <div className="border-charcoal-400 border-t-charcoal-200 h-8 w-8 animate-spin rounded-full border-4"></div>
        </div>
      )}
    </>
  );
};
