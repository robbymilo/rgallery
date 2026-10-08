import React, { useState, useEffect, useCallback, useRef } from 'react';
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

  const requestControllerRef = useRef(new AbortController());

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
    params.delete('camera');
    params.delete('lens');
    params.delete('software');
    params.delete('focallength35');
    params.delete('type');
    params.delete('orderby');
    params.delete('direction');

    // Add current filter params (only if not default)
    if (apiFilters.term) params.set('term', apiFilters.term);
    if (apiFilters.rating && apiFilters.rating > 1) params.set('rating', apiFilters.rating.toString());
    if (apiFilters.tag) params.set('tag', apiFilters.tag);
    if (apiFilters.folder) params.set('folder', apiFilters.folder);
    if (apiFilters.camera) params.set('camera', apiFilters.camera);
    if (apiFilters.lens) params.set('lens', apiFilters.lens);
    if (apiFilters.software) params.set('software', apiFilters.software);
    if (apiFilters.focallength35) params.set('focallength35', apiFilters.focallength35.toString());
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

      const controller = requestControllerRef.current;
      console.log(`[Timeline] Loading chunk at offset ${offset} (${mode})`);

      isLoadingRef.current = true;
      setIsLoading(true);

      try {
        const apiFilters = getApiFilters();
        if (error) setError(null);
        const response = await fetchPhotos(offset.toString(), apiFilters, controller.signal);
        if (controller.signal.aborted) return;
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
        if (controller.signal.aborted) return;
        console.error(e);
        setError(e);
      } finally {
        if (!controller.signal.aborted) {
          isLoadingRef.current = false;
          setIsLoading(false);
        }
      }
    },
    [mapApiPhotos, getApiFilters]
  );

  // Cancel every request from the previous filters, including pagination and jumps.
  useEffect(() => {
    requestControllerRef.current.abort();
    const controller = new AbortController();
    requestControllerRef.current = controller;
    const restorePosition = !initCalledRef.current;
    latestScrubDateRef.current = null;
    isScrubbingLoopActiveRef.current = false;
    isLoadingRef.current = true;
    setIsLoading(true);
    setError(null);
    setPhotos([]);
    setTimeline([]);
    setTotalCount(0);
    setMinOffset(0);
    setMaxOffset(0);

    const load = async () => {
      try {
        const apiFilters = getApiFilters();
        const response = await fetchPhotos('0', apiFilters, controller.signal);
        if (controller.signal.aborted) return;
        const histogram = response.timeline || [];
        let cursor = 0;
        let targetDate: Date | null = null;
        if (restorePosition) {
          const date = new URLSearchParams(window.location.search).get('date') || localStorage.getItem(STORAGE_KEY);
          if (date && !isNaN(new Date(date).getTime())) {
            targetDate = new Date(date);
            const index = histogram
              .filter((item) => item.date > date.split('T')[0])
              .reduce((sum, item) => sum + item.count, 0);
            if (index < response.meta.total) cursor = Math.floor(index / PAGE_SIZE) * PAGE_SIZE;
          }
        }
        const page = cursor === 0 ? response : await fetchPhotos(cursor.toString(), apiFilters, controller.signal);
        if (controller.signal.aborted) return;
        const loaded = mapApiPhotos(page.photos);
        setTimeline(histogram);
        setTotalCount(response.meta.total);
        setPhotos(loaded);
        setMinOffset(cursor);
        setMaxOffset(cursor + loaded.length);
        if (loaded.length) {
          const date = targetDate || loaded[0].date;
          setScrollToRequest({ date, timestamp: Date.now() });
          setVisibleDate(date);
        }
        initCalledRef.current = true;
      } catch (e) {
        if (!controller.signal.aborted) setError(e);
      } finally {
        if (!controller.signal.aborted) {
          isLoadingRef.current = false;
          setIsLoading(false);
        }
      }
    };
    void load();
    return () => controller.abort();
  }, [mapApiPhotos, getApiFilters]);

  const handleEndReached = () => {
    void loadChunk(maxOffset, 'append');
  };
  const handleStartReached = () => {
    if (minOffset > 0) void loadChunk(Math.max(0, minOffset - PAGE_SIZE), 'prepend');
  };

  const performJump = async (date: Date, updateUrl: boolean) => {
    const controller = requestControllerRef.current;
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
      const response = await fetchPhotos(cursor.toString(), apiFilters, controller.signal);
      if (controller.signal.aborted) return;
      const newPhotos = mapApiPhotos(response.photos);

      setPhotos(newPhotos);
      setMinOffset(cursor);
      setMaxOffset(cursor + newPhotos.length);

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
      if (controller.signal.aborted) return;
      setError(e);
      console.error(e);
    }
  };

  const jumpToDate = async (date: Date, updateUrl: boolean = true) => {
    const controller = requestControllerRef.current;
    setVisibleDate(date);
    latestScrubDateRef.current = date;

    if (isScrubbingLoopActiveRef.current) return;

    isScrubbingLoopActiveRef.current = true;
    isLoadingRef.current = true;
    setIsLoading(true);

    try {
      while (!controller.signal.aborted && latestScrubDateRef.current) {
        const target = latestScrubDateRef.current;
        latestScrubDateRef.current = null;
        await performJump(target, updateUrl);
      }
    } finally {
      if (!controller.signal.aborted) {
        isScrubbingLoopActiveRef.current = false;
        isLoadingRef.current = false;
        setIsLoading(false);
      }
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

  const handleRefresh = useCallback(async () => {
    const controller = requestControllerRef.current;
    if (isLoadingRef.current) return;
    isLoadingRef.current = true;
    setIsLoading(true);
    try {
      const apiFilters = getApiFilters();
      if (error) setError(null);
      const response = await fetchPhotos('0', apiFilters, controller.signal);
      if (controller.signal.aborted) return;
      const newPhotos = mapApiPhotos(response.photos || []);
      setTimeline(response.timeline || []);
      setTotalCount(response.meta.total);
      setPhotos(newPhotos);
      setMinOffset(0);
      setMaxOffset(newPhotos.length);
      if (newPhotos.length > 0) {
        setScrollToRequest({ date: newPhotos[0].date, timestamp: Date.now() });
        setVisibleDate(newPhotos[0].date);
      }
    } catch (e) {
      if (controller.signal.aborted) return;
      setError(e);
      console.error('Refresh failed:', e);
    } finally {
      if (!controller.signal.aborted) {
        isLoadingRef.current = false;
        setIsLoading(false);
      }
    }
  }, [getApiFilters, mapApiPhotos, error]);

  const handleScrollToTop = useCallback(() => {
    if (timeline.length === 0) return;
    const firstDate = new Date(timeline[0].date);
    if (minOffset === 0) {
      void handleRefresh();
    } else {
      handleDateSelect(firstDate);
    }
  }, [timeline, minOffset, handleDateSelect, handleRefresh]);

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
