import React, { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { fetchPhotos } from '../services/timeline';
import { Photo, TimelineResponse, ApiTimelineItem, FilterState, TimelineFilters } from '../types';
import { VirtualGrid } from '../components/VirtualGrid';
import { TimelineScrubber } from '../components/TimelineScrubber';
import FilterBar from '../components/FilterBar';
import Error from '../components/Error';

const PAGE_SIZE = 1000;
const STORAGE_KEY = 'rgallery_scroll_date';

const App: React.FC = () => {
  const [photos, setPhotos] = useState<Photo[]>([]);
  const [timeline, setTimeline] = useState<ApiTimelineItem[]>([]);
  const [isLoading, setIsLoading] = useState(true); // Start true to block UI until init
  const [visibleDate, setVisibleDate] = useState<Date>(new Date());
  const [totalCount, setTotalCount] = useState<number>(0);
  const [error, setError] = useState<Error | null>(null);

  const [minOffset, setMinOffset] = useState<number>(0);
  const [maxOffset, setMaxOffset] = useState<number>(0);

  // Initialize filters from URL params
  const getInitialFilters = (): FilterState => {
    const params = new URLSearchParams(window.location.search);
    return {
      searchQuery: params.get('term') || '',
      minRating: params.get('rating') ? parseInt(params.get('rating') || '0') : 1,
      tag: params.get('tag') || undefined,
      folder: params.get('folder') || undefined,
      camera: params.get('camera') || undefined,
      lens: params.get('lens') || undefined,
      software: params.get('software') || undefined,
      focallength35: params.get('focallength35') ? parseInt(params.get('focallength35') || '0') : undefined,
      mediaType: (params.get('type') as 'image' | 'video') || 'all',
      sortBy: `${params.get('orderby') || 'date'}-${params.get('direction') || 'desc'}` as
        | 'date-asc'
        | 'date-desc'
        | 'modified-asc'
        | 'modified-desc',
    };
  };

  const [filters, setFilters] = useState<FilterState>(getInitialFilters());

  const [scrollToRequest, setScrollToRequest] = useState<{ date: Date; timestamp: number } | null>(null);

  const initCalledRef = useRef(false);
  const isLoadingRef = useRef(false);
  const storageTimeoutRef = useRef<number>(0);

  const latestScrubDateRef = useRef<Date | null>(null);
  const isScrubbingLoopActiveRef = useRef(false);

  const nextPrefetchRef = useRef<Promise<Photo[]> | null>(null);
  const prevPrefetchRef = useRef<Promise<Photo[]> | null>(null);

  // Map API photos to Photo objects
  const mapApiPhotos = useCallback((apiPhotos: TimelineResponse['photos']): Photo[] => {
    return apiPhotos.map((p) => ({
      id: p.id.toString(),
      url:
        p.t === 'video'
          ? `/api/transcode/${p.id}/index.m3u8`
          : (() => {
              // Build a srcset where the largest candidate is not larger than the photo width
              const candidates = [200, 400, 800];
              const valid = candidates.filter((s) => s <= p.w);
              return valid.map((s) => `/api/img/${p.id}/${s} ${s}w`).join(', ');
            })(),
      width: p.w,
      height: p.h,
      aspectRatio: p.w / p.h,
      date: new Date(p.d),
      type: p.t === 'video' ? 'video' : 'image',
      color: p.c,
      path: p.path,
    }));
  }, []);

  // Convert FilterState to API params
  const getApiFilters = useCallback((): TimelineFilters => {
    console.log('[Timeline] Converting filters to API params', filters);
    const [orderby, direction] = filters.sortBy.split('-') as ['date' | 'modified', 'asc' | 'desc'];
    return {
      term: filters.searchQuery || undefined,
      // Only include rating in API filters when it's 2 or greater
      rating: filters.minRating && filters.minRating > 1 ? filters.minRating : undefined,
      tag: filters.tag || undefined,
      camera: filters.camera || undefined,
      lens: filters.lens || undefined,
      software: filters.software || undefined,
      folder: filters.folder || undefined,
      focallength35: filters.focallength35 || undefined,
      type: filters.mediaType !== 'all' ? filters.mediaType : undefined,
      orderby,
      direction,
    };
  }, [filters]);

  // Sync filters to URL
  useEffect(() => {
    console.log('[Timeline] Updating URL with filters', filters);
    const params = new URLSearchParams(window.location.search);
    const apiFilters = getApiFilters();

    // Remove old filter params
    params.delete('term');
    params.delete('rating');
    params.delete('tag');
    params.delete('folder');
    params.delete('type');
    params.delete('orderby');
    params.delete('direction');

    // Add current filter params (only if not default)
    if (apiFilters.term) params.set('term', apiFilters.term);
    if (apiFilters.rating && apiFilters.rating > 1) params.set('rating', apiFilters.rating.toString());
    if (apiFilters.tag) params.set('tag', apiFilters.tag);
    if (apiFilters.folder) params.set('folder', apiFilters.folder);
    if (apiFilters.type) params.set('type', apiFilters.type);
    // Only add orderby/direction if not default values
    if (apiFilters.orderby && apiFilters.orderby !== 'date') params.set('orderby', apiFilters.orderby);
    if (apiFilters.direction && apiFilters.direction !== 'desc') params.set('direction', apiFilters.direction);

    const newUrl = params.toString() ? `?${params.toString()}` : window.location.pathname;
    window.history.replaceState({}, '', newUrl);
  }, [filters, getApiFilters]);

  // Listen for memory-scroll event from Layout (after all declarations)
  useEffect(() => {
    const handleMemoryScroll = (e: Event) => {
      console.log('[Timeline] Received memory-scroll event');
      const customEvent = e as CustomEvent;
      const dateStr = customEvent.detail?.date;
      if (dateStr) {
        const dateObj = new Date(dateStr);
        if (!isNaN(dateObj.getTime())) {
          jumpToDate(dateObj, false);
        }
      }
    };
    window.addEventListener('memory-scroll', handleMemoryScroll);
    return () => {
      window.removeEventListener('memory-scroll', handleMemoryScroll);
    };
  }, [timeline, mapApiPhotos]);

  // Initial load
  const loadChunk = useCallback(
    async (offset: number, mode: 'append' | 'prepend') => {
      if (isLoadingRef.current) return;
      if (offset < 0) return;

      console.log(`[Timeline] Loading chunk at offset ${offset} (${mode})`);

      isLoadingRef.current = true;
      setIsLoading(true);

      try {
        const apiFilters = getApiFilters();
        if (error) setError(null);
        const response = await fetchPhotos(offset.toString(), apiFilters);
        const newPhotos = mapApiPhotos(response.photos);

        if (newPhotos.length === 0) return;

        setPhotos((prev) => {
          const map = new Map();
          if (mode === 'append') {
            prev.forEach((p) => map.set(p.id, p));
            newPhotos.forEach((p) => map.set(p.id, p));
          } else {
            newPhotos.forEach((p) => map.set(p.id, p));
            prev.forEach((p) => map.set(p.id, p));
          }
          return Array.from(map.values());
        });

        if (mode === 'append') {
          setMaxOffset((prev) => prev + newPhotos.length);
        } else {
          setMinOffset((prev) => Math.max(0, prev - PAGE_SIZE));
        }
      } catch (e) {
        console.error(e);
        setError(e);
      } finally {
        isLoadingRef.current = false;
        setIsLoading(false);
      }
    },
    [mapApiPhotos, getApiFilters]
  );

  useEffect(() => {
    if (initCalledRef.current) return;
    initCalledRef.current = true;
    isLoadingRef.current = true;

    console.log('[Timeline] Performing initial load');

    const runInit = async () => {
      try {
        const apiFilters = getApiFilters();
        if (error) setError(null);
        const initResponse = await fetchPhotos('0', apiFilters);
        const globalTimeline = initResponse.timeline || [];
        setTimeline(globalTimeline);
        setTotalCount(initResponse.meta.total);

        const params = new URLSearchParams(window.location.search);
        let dateParam = params.get('date');
        if (!dateParam) {
          dateParam = localStorage.getItem(STORAGE_KEY);
        }

        console.log('[Timeline] dateParam from URL or localStorage:', dateParam);

        let targetIndex = 0;
        let targetDateObj: Date | null = null;

        if (dateParam && !isNaN(new Date(dateParam).getTime())) {
          targetDateObj = new Date(dateParam);
          const dateStr = dateParam.split('T')[0];
          console.log('[Timeline] Calculating index for date:', dateStr);
          for (const item of globalTimeline) {
            if (item.date > dateStr) {
              targetIndex += item.count;
            } else {
              break;
            }
          }
          console.log(
            '[Timeline] Target index:',
            targetIndex,
            'cursor:',
            Math.floor(targetIndex / PAGE_SIZE) * PAGE_SIZE
          );
        }

        const cursor = Math.floor(targetIndex / PAGE_SIZE) * PAGE_SIZE;
        let initialPhotos: Photo[] = [];

        if (cursor === 0) {
          console.log('[Timeline] Using initial response photos (cursor === 0)');
          initialPhotos = mapApiPhotos(initResponse.photos);
        } else {
          console.log('[Timeline] Fetching photos from cursor:', cursor);
          if (error) setError(null);
          const response = await fetchPhotos(cursor.toString(), apiFilters);
          initialPhotos = mapApiPhotos(response.photos);
        }

        setPhotos(initialPhotos);
        setMinOffset(cursor);
        setMaxOffset(cursor + initialPhotos.length);

        console.log('[Timeline] Initial photos loaded:', initialPhotos.length, 'first date:', initialPhotos[0]?.date);

        if (targetDateObj && initialPhotos.length > 0) {
          console.log('[Timeline] Setting scroll request for date:', targetDateObj);
          setScrollToRequest({ date: targetDateObj, timestamp: Date.now() });
          setVisibleDate(targetDateObj);
        } else if (initialPhotos.length > 0) {
          console.log('[Timeline] No target date, using first photo date');
          setVisibleDate(initialPhotos[0].date);
        }
      } catch (e) {
        setError(e);
        console.error('Timeline failed:', e);
      } finally {
        isLoadingRef.current = false;
        setIsLoading(false);
      }
    };

    runInit();
  }, [mapApiPhotos, getApiFilters]);

  // Reset pagination when filters change
  const prevFiltersRef = useRef(filters);
  useEffect(() => {
    // Don't run before initial load completes
    if (!initCalledRef.current) {
      prevFiltersRef.current = filters;
      return;
    }

    // Skip if filters haven't changed since last run
    if (JSON.stringify(filters) === JSON.stringify(prevFiltersRef.current)) {
      return;
    }

    console.log('[Timeline] Filters changed, resetting pagination', filters);

    const resetWithFilters = async () => {
      isLoadingRef.current = true;
      setIsLoading(true);

      try {
        const apiFilters = getApiFilters();
        if (error) setError(null);
        const response = await fetchPhotos('0', apiFilters);

        setTimeline(response.timeline || []);
        setTotalCount(response.meta.total);
        const newPhotos = mapApiPhotos(response.photos || []);
        setPhotos(newPhotos);
        setMinOffset(0);
        setMaxOffset(newPhotos.length);

        // Reset to top
        if (newPhotos.length > 0) {
          setScrollToRequest({ date: newPhotos[0].date, timestamp: Date.now() });
          setVisibleDate(newPhotos[0].date);
        }
      } catch (e) {
        console.error('Filter reset failed:', e);
      } finally {
        isLoadingRef.current = false;
        setIsLoading(false);
      }
    };

    // Run the reset and then update the previous-filters snapshot
    resetWithFilters().finally(() => {
      prevFiltersRef.current = filters;
    });
  }, [filters, mapApiPhotos, getApiFilters]);

  const handleEndReached = async () => {
    if (isLoadingRef.current) return;
    if (nextPrefetchRef.current) {
      console.log('[Timeline] Using prefetched next photos');

      isLoadingRef.current = true;
      setIsLoading(true);
      try {
        const newPhotos = await nextPrefetchRef.current;
        nextPrefetchRef.current = null;
        if (newPhotos.length > 0) {
          setPhotos((prev) => {
            const map = new Map();
            prev.forEach((p) => map.set(p.id, p));
            newPhotos.forEach((p) => map.set(p.id, p));
            return Array.from(map.values());
          });
          setMaxOffset((prev) => prev + newPhotos.length);
        }
      } catch (e) {
        console.error(e);
      } finally {
        isLoadingRef.current = false;
        setIsLoading(false);
      }
    } else {
      loadChunk(maxOffset, 'append');
    }
  };

  const handleStartReached = async () => {
    if (isLoadingRef.current) return;
    if (minOffset <= 0) return;

    if (prevPrefetchRef.current) {
      console.log('[Timeline] Using prefetched previous photos');

      isLoadingRef.current = true;
      setIsLoading(true);
      try {
        const newPhotos = await prevPrefetchRef.current;
        prevPrefetchRef.current = null;
        if (newPhotos.length > 0) {
          setPhotos((prev) => {
            const map = new Map();
            newPhotos.forEach((p) => map.set(p.id, p));
            prev.forEach((p) => map.set(p.id, p));
            return Array.from(map.values());
          });
          setMinOffset((prev) => Math.max(0, prev - PAGE_SIZE));
        }
      } catch (e) {
        console.error(e);
      } finally {
        isLoadingRef.current = false;
        setIsLoading(false);
      }
    } else {
      const newOffset = Math.max(0, minOffset - PAGE_SIZE);
      loadChunk(newOffset, 'prepend');
    }
  };

  const performJump = async (date: Date, updateUrl: boolean) => {
    const dateStr = date.toISOString().split('T')[0];
    let index = 0;

    // Calculate global index of the target date
    for (const item of timeline) {
      if (item.date > dateStr) {
        index += item.count;
      } else {
        break;
      }
    }

    try {
      console.log(
        `[Timeline] Performing jump to date ${dateStr} at index ${index} (offset ${Math.floor(index / PAGE_SIZE) * PAGE_SIZE})`
      );
      const cursor = Math.floor(index / PAGE_SIZE) * PAGE_SIZE;
      const apiFilters = getApiFilters();
      const response = await fetchPhotos(cursor.toString(), apiFilters);
      const newPhotos = mapApiPhotos(response.photos);

      setPhotos(newPhotos);
      setMinOffset(cursor);
      setMaxOffset(cursor + newPhotos.length);

      nextPrefetchRef.current = null;
      prevPrefetchRef.current = null;

      setScrollToRequest({
        date: date,
        timestamp: Date.now(),
      });

      if (updateUrl) {
        const params = new URLSearchParams(window.location.search);
        params.set('date', dateStr);
        const newUrl = `?${params.toString()}`;
        window.history.pushState({}, '', newUrl);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const jumpToDate = async (date: Date, updateUrl: boolean = true) => {
    setVisibleDate(date);
    latestScrubDateRef.current = date;

    if (isScrubbingLoopActiveRef.current) return;

    isScrubbingLoopActiveRef.current = true;
    isLoadingRef.current = true;
    setIsLoading(true);

    try {
      while (latestScrubDateRef.current) {
        const target = latestScrubDateRef.current;
        latestScrubDateRef.current = null;
        await performJump(target, updateUrl);
      }
    } finally {
      isScrubbingLoopActiveRef.current = false;
      isLoadingRef.current = false;
      setIsLoading(false);
    }
  };

  const handleScrub = useCallback(
    (date: Date) => {
      jumpToDate(date, false);
    },
    [timeline, mapApiPhotos, getApiFilters]
  );

  const handleDateSelect = (date: Date) => {
    jumpToDate(date, false);
  };

  const handleScrollToTop = useCallback(() => {
    if (timeline.length === 0) return;
    const firstDate = new Date(timeline[0].date);
    if (minOffset === 0) {
      setScrollToRequest({ date: firstDate, timestamp: Date.now() });
      setVisibleDate(firstDate);
    } else {
      handleDateSelect(firstDate);
    }
  }, [timeline, minOffset, handleDateSelect]);

  useEffect(() => {
    const onScrollTop = () => handleScrollToTop();
    window.addEventListener('scroll-to-top', onScrollTop);
    return () => window.removeEventListener('scroll-to-top', onScrollTop);
  }, [handleScrollToTop]);

  const handleVisibleDateChange = (date: Date) => {
    if (!isScrubbingLoopActiveRef.current) {
      setVisibleDate(date);
    }

    if (storageTimeoutRef.current) {
      clearTimeout(storageTimeoutRef.current);
    }
    storageTimeoutRef.current = window.setTimeout(() => {
      const dateStr = date.toISOString().split('T')[0];
      localStorage.setItem(STORAGE_KEY, dateStr);
    }, 50);
  };

  const [hasMoreItems, setHasMoreItems] = useState(true); // Track if more items are available

  // Update hasMoreItems whenever photos or totalCount changes
  useEffect(() => {
    setHasMoreItems(photos.length < totalCount);
  }, [photos.length, totalCount]);

  // convert TimelineFilters to Record
  const convertFiltersToQueryParams = (filters: TimelineFilters): Record<string, string> => {
    return Object.fromEntries(
      Object.entries(filters)
        .filter(([key, value]) => {
          // remove default values
          if (key === 'orderby' && value === 'date') return false;
          if (key === 'direction' && value === 'desc') return false;
          return value !== undefined && value !== null;
        })
        .map(([key, value]) => [key, value.toString()])
    );
  };

  return (
    <div className="dark:bg-charcoal-900 relative flex h-full flex-col bg-gray-50 font-sans transition-colors duration-300">
      <FilterBar filters={filters} onFilterChange={setFilters} totalItems={totalCount} />

      {error && <Error error={error.message} />}

      <main className="relative h-full flex-1 pt-14">
        <div className="absolute inset-0 z-0">
          <VirtualGrid
            photos={photos}
            onEndReached={handleEndReached}
            onStartReached={handleStartReached}
            onVisibleDateChange={handleVisibleDateChange}
            onScrollToTop={handleScrollToTop}
            isLoading={isLoading}
            scrollToTarget={scrollToRequest}
            hasMoreItems={hasMoreItems}
            filters={convertFiltersToQueryParams(getApiFilters())}
          />
        </div>

        <div className="pointer-events-none absolute top-0 right-0 bottom-0 z-40 flex w-24 flex-col justify-center overflow-hidden">
          <TimelineScrubber
            timeline={timeline}
            onDateSelect={handleDateSelect}
            onScrub={handleScrub}
            currentDate={visibleDate}
            className="h-full"
          />
        </div>
      </main>
    </div>
  );
};

export default App;
