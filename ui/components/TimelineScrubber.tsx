import React, { useMemo, useRef, useState, useEffect, useCallback } from 'react';
import { ApiTimelineItem } from '../types';
import Scrubber from '../svg/scrubber.svg?react';

interface TimelineScrubberProps {
  timeline: ApiTimelineItem[];
  currentDate: Date;
  onDateSelect: (date: Date) => void;
  onScrub: (date: Date) => void;
  className?: string;
}

interface MonthNode {
  monthIndex: number;
  dateStr: string;
  year: number;
  total: number;
  isSpacer?: boolean;
}

interface YearGroup {
  year: number;
  months: MonthNode[];
  count: number;
}

const TICK_HEIGHT = 8; // h-3 = 12px
const PADDING_TOP = 40; // py-10 = 40px
const MIN_YEAR_TICKS = 3;
const DRAG_THRESHOLD = 4;

export const TimelineScrubber: React.FC<TimelineScrubberProps> = ({
  timeline,
  currentDate,
  onDateSelect,
  onScrub,
  className,
}) => {
  const outerRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  const [isDragging, setIsDragging] = useState(false);
  const [thumbTop, setThumbTop] = useState<number | null>(null);

  const lastScrubbedDateRef = useRef<Date | null>(null);
  const scrollSpeedRef = useRef(0);
  const animationFrameRef = useRef<number | null>(null);
  const lastPointerPosRef = useRef<{ x: number; y: number } | null>(null);
  const dragStartRef = useRef<{ x: number; y: number } | null>(null);
  const didDragRef = useRef(false);
  const grabOffsetRef = useRef(0);

  // Track if we have done the initial scroll positioning
  const hasInitialScrolledRef = useRef(false);

  // Flatten timeline for calendar
  const { groups, flatNodes, totalItems } = useMemo(() => {
    const grps: YearGroup[] = [];
    const nodes: MonthNode[] = [];

    // get total of items per month
    let monthTotals = new Map<string, number>();
    let totalItems = 0 as number;
    timeline.forEach((m) => {
      const d = new Date(m.date);
      // month is zero based key
      const key = d.getFullYear() + '-' + d.getMonth();
      monthTotals.set(key, (monthTotals.get(key) || 0) + m.count);
      totalItems += m.count;
    });

    // Each month has one tick; its target must be the newest day in that month.
    [...timeline].sort((a, b) => b.date.localeCompare(a.date)).forEach((item) => {
      const d = new Date(item.date);
      const year = d.getFullYear();
      const month = d.getMonth();

      let yearGroup = grps.find((g) => g.year === year);
      if (!yearGroup) {
        yearGroup = { year, months: [], count: 0 };
        grps.push(yearGroup);
      }

      if (!yearGroup.months.find((m) => m.monthIndex === month)) {
        const node = {
          monthIndex: month,
          dateStr: item.date,
          year: year,
          total: monthTotals.get(`${year}-${month}`) || 0,
        };
        yearGroup.months.push(node);
      }
      yearGroup.count += item.count;
    });

    // Sort Descending (Newest Top)
    grps.sort((a, b) => b.year - a.year);
    grps.forEach((g) => g.months.sort((a, b) => b.monthIndex - a.monthIndex));

    grps.forEach((g) => {
      g.months.forEach((m) => nodes.push(m));

      // Keep sparse years from collapsing into a single indistinguishable
      // tick. These spacers are part of the navigation scale, so the pointer
      // mapping stays in sync with the rendered height.
      for (let i = g.months.length; i < MIN_YEAR_TICKS; i += 1) {
        const lastMonth = g.months[g.months.length - 1];
        nodes.push({
          monthIndex: lastMonth.monthIndex,
          dateStr: lastMonth.dateStr,
          year: g.year,
          total: 0,
          isSpacer: true,
        });
      }
    });

    return { groups: grps, flatNodes: nodes, totalItems };
  }, [timeline]);

  const startAutoScroll = () => {
    if (animationFrameRef.current) return;

    const loop = () => {
      if (scrollSpeedRef.current !== 0 && scrollContainerRef.current) {
        scrollContainerRef.current.scrollTop += scrollSpeedRef.current;
        if (lastPointerPosRef.current) {
          processPointer(lastPointerPosRef.current.x, lastPointerPosRef.current.y - grabOffsetRef.current);
        }
        animationFrameRef.current = requestAnimationFrame(loop);
      } else {
        animationFrameRef.current = null;
      }
    };
    animationFrameRef.current = requestAnimationFrame(loop);
  };

  const stopAutoScroll = () => {
    scrollSpeedRef.current = 0;
    if (animationFrameRef.current) {
      cancelAnimationFrame(animationFrameRef.current);
      animationFrameRef.current = null;
    }
  };

  const handlePointerDown = (e: React.PointerEvent) => {
    if (e.pointerType === 'touch') return;
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(true);
    e.currentTarget.setPointerCapture(e.pointerId);
    lastScrubbedDateRef.current = null;
    dragStartRef.current = { x: e.clientX, y: e.clientY };
    didDragRef.current = false;
    grabOffsetRef.current = e.clientY - (outerRef.current?.getBoundingClientRect().top ?? 0) - (thumbTop ?? 0);
  };

  const handlePointerMove = (e: React.PointerEvent) => {
    if (!isDragging || e.pointerType === 'touch') return;
    e.preventDefault();
    if (!startDrag(e.clientX, e.clientY)) return;
    handleInteraction(e.clientX, e.clientY);
  };

  const handlePointerUp = (e: React.PointerEvent) => {
    if (e.pointerType === 'touch') return;
    endInteraction(e.currentTarget, e.pointerId);
  };

  const handleTouchStart = (e: React.TouchEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(true);
    lastScrubbedDateRef.current = null;
    if (e.touches.length > 0) {
      dragStartRef.current = { x: e.touches[0].clientX, y: e.touches[0].clientY };
      didDragRef.current = false;
      grabOffsetRef.current = e.touches[0].clientY - (outerRef.current?.getBoundingClientRect().top ?? 0) - (thumbTop ?? 0);
    }
  };

  const handleTouchMove = (e: React.TouchEvent) => {
    if (!isDragging) return;
    if (e.cancelable) e.preventDefault();
    if (e.touches.length > 0) {
      if (!startDrag(e.touches[0].clientX, e.touches[0].clientY)) return;
      handleInteraction(e.touches[0].clientX, e.touches[0].clientY);
    }
  };

  const handleTouchEnd = (e: React.TouchEvent) => {
    setIsDragging(false);
    stopAutoScroll();
    dragStartRef.current = null;
    if (didDragRef.current && lastScrubbedDateRef.current) {
      onDateSelect(lastScrubbedDateRef.current);
    }
    lastScrubbedDateRef.current = null;
    didDragRef.current = false;
    grabOffsetRef.current = 0;
  };

  const startDrag = (x: number, y: number) => {
    if (didDragRef.current) return true;
    const start = dragStartRef.current;
    if (!start || Math.hypot(x - start.x, y - start.y) < DRAG_THRESHOLD) return false;
    didDragRef.current = true;
    return true;
  };

  const handleInteraction = (clientX: number, clientY: number) => {
    lastPointerPosRef.current = { x: clientX, y: clientY };
    const thumbCenterY = clientY - grabOffsetRef.current;
    updateThumbFromPointer(thumbCenterY);

    // Edge scrolling
    if (outerRef.current) {
      const rect = outerRef.current.getBoundingClientRect();
      const relY = clientY - rect.top;
      const EDGE_ZONE = 60;
      const MAX_SPEED = window.innerWidth <= 768 ? 40 : 20;

      if (relY < EDGE_ZONE) {
        const intensity = 1 - Math.max(0, relY) / EDGE_ZONE;
        scrollSpeedRef.current = -1 * intensity * MAX_SPEED;
        startAutoScroll();
      } else if (relY > rect.height - EDGE_ZONE) {
        const intensity = 1 - Math.max(0, rect.height - relY) / EDGE_ZONE;
        scrollSpeedRef.current = intensity * MAX_SPEED;
        startAutoScroll();
      } else {
        stopAutoScroll();
      }
    }

    processPointer(clientX, thumbCenterY);
  };

  const endInteraction = (target: Element, pointerId: number) => {
    setIsDragging(false);
    stopAutoScroll();
    dragStartRef.current = null;
    target.releasePointerCapture(pointerId);
    if (didDragRef.current && lastScrubbedDateRef.current) {
      onDateSelect(lastScrubbedDateRef.current);
    }
    lastScrubbedDateRef.current = null;
    didDragRef.current = false;
    grabOffsetRef.current = 0;
  };

  const updateThumbFromPointer = (clientY: number) => {
    if (!outerRef.current) return;
    const containerRect = outerRef.current.getBoundingClientRect();
    const relativeY = clientY - containerRect.top;

    // Clamp to valid tick area
    const minTop = PADDING_TOP + TICK_HEIGHT / 2;
    const maxTop = containerRect.height - 10;
    const clampedY = Math.max(minTop, Math.min(maxTop, relativeY));

    setThumbTop(clampedY);
  };

  const processPointer = (x: number, y: number) => {
    if (!scrollContainerRef.current || !outerRef.current) return;

    const scrollRect = scrollContainerRef.current.getBoundingClientRect();
    const scrollTop = scrollContainerRef.current.scrollTop;
    const relativeY = y - scrollRect.top + scrollTop - PADDING_TOP;

    const index = Math.floor(relativeY / TICK_HEIGHT);
    const clampedIndex = Math.max(0, Math.min(flatNodes.length - 1, index));
    const node = flatNodes[clampedIndex];

    if (node) {
      const date = new Date(node.dateStr);
      if (!lastScrubbedDateRef.current || lastScrubbedDateRef.current.getTime() !== date.getTime()) {
        lastScrubbedDateRef.current = date;
        onScrub(date);
      }
    }
  };

  // update thumb position when auto-centering during scroll
  const handleScroll = () => {
    if (isDragging || !currentDate || !outerRef.current || !scrollContainerRef.current) return;

    // Re-sync thumb to the current active date during scroll animation
    const year = currentDate.getFullYear();
    const month = currentDate.getMonth();
    const index = flatNodes.findIndex((n) => n.year === year && n.monthIndex === month);

    if (index !== -1) {
      const elTop = PADDING_TOP + index * TICK_HEIGHT;
      const elCenter = elTop + TICK_HEIGHT / 2;
      const currentScrollTop = scrollContainerRef.current.scrollTop;
      setThumbTop(elCenter - currentScrollTop);
    }
  };

  // sync thumb position and scroll position with current date
  useEffect(() => {
    if (isDragging || !outerRef.current || !scrollContainerRef.current || flatNodes.length === 0) return;

    const year = currentDate.getFullYear();
    const month = currentDate.getMonth();
    const index = flatNodes.findIndex((n) => n.year === year && n.monthIndex === month);

    if (index !== -1) {
      const elTop = PADDING_TOP + index * TICK_HEIGHT;
      const elCenter = elTop + TICK_HEIGHT / 2;

      const containerRect = outerRef.current.getBoundingClientRect();
      const viewportHeight = containerRect.height;
      const currentScrollTop = scrollContainerRef.current.scrollTop;
      const currentVisualY = elCenter - currentScrollTop;

      const safeZoneTop = viewportHeight * 0.3;
      const safeZoneBottom = viewportHeight * 0.7;

      if (currentVisualY < safeZoneTop || currentVisualY > safeZoneBottom) {
        // Calculate target scroll to center
        let targetScrollTop = elCenter - viewportHeight / 2;
        const maxScroll = scrollContainerRef.current.scrollHeight - scrollContainerRef.current.clientHeight;
        targetScrollTop = Math.max(0, Math.min(maxScroll, targetScrollTop));

        // Use instant scroll on first load to ensure position is correct immediately
        const isFirstLoad = !hasInitialScrolledRef.current;

        scrollContainerRef.current.scrollTo({
          top: targetScrollTop,
          behavior: isFirstLoad ? 'auto' : 'smooth',
        });

        if (isFirstLoad) {
          hasInitialScrolledRef.current = true;
        }

        setThumbTop(elCenter - targetScrollTop);
      } else {
        setThumbTop(currentVisualY);
        hasInitialScrolledRef.current = true;
      }
    }
  }, [currentDate, flatNodes, isDragging]);

  if (timeline.length === 0) return null;

  return (
    <div ref={outerRef} className={`relative h-full select-none ${className}`}>
      <div
        ref={scrollContainerRef}
        className="pointer-events-none h-full w-full overflow-hidden pb-32"
        onScroll={handleScroll}
        style={{ scrollBehavior: isDragging ? 'auto' : 'smooth' }}
      >
        <div className="flex w-full flex-col items-end py-10 pr-4">
          {groups.map((group) => (
            <div key={group.year} className="relative flex w-full flex-col items-end">
              <div
                className="absolute top-0 right-8 pr-4 text-[10px] leading-2 font-bold text-zinc-600 tabular-nums dark:text-zinc-400"
              >
                {group.year}
              </div>
              {group.months.map((m) => {
                const ratio = Math.round((m.total / totalItems) * 1000);

                return (
                  <div key={m.monthIndex} className="z-10 flex h-2 w-12 items-center justify-end">
                    <div className="h-1 bg-zinc-600 transition-all dark:bg-zinc-400" style={{ width: `${ratio}px` }} />
                  </div>
                );
              })}
              {Array.from({ length: Math.max(0, MIN_YEAR_TICKS - group.months.length) }).map((_, index) => (
                <div key={`spacer-${index}`} className="h-2 w-12" aria-hidden="true" />
              ))}
            </div>
          ))}
          <div className="h-64" />
        </div>
      </div>

      {thumbTop !== null && (
        <button
          className="pointer-events-auto absolute right-0 z-20 w-24 cursor-grab transition-none focus:outline-none active:cursor-grabbing"
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={handlePointerUp}
          onTouchStart={handleTouchStart}
          onTouchMove={handleTouchMove}
          onTouchEnd={handleTouchEnd}
          style={{
            top: thumbTop,
            transform: 'translateY(-50%)',
            transition: isDragging ? 'none' : 'top 0.1s linear',
            touchAction: 'none',
          }}
          role="slider"
          aria-label="Timeline scrubber"
          aria-valuetext={currentDate.toLocaleString('default', { month: 'long', year: 'numeric' })}
        >
          <div className="pointer-events-none flex w-24 min-w-max items-center justify-between gap-1.5 rounded-l-full bg-zinc-600 px-3 py-1 text-black shadow-lg dark:bg-white">
            <span className="text-[10px] font-bold whitespace-nowrap text-white tabular-nums dark:text-zinc-600">
              {currentDate.getFullYear()}{' '}
              <span className="text-white dark:text-zinc-600">
                {currentDate.toLocaleString('default', { month: 'short' })}
              </span>
            </span>
            <Scrubber className="h-3 w-3 text-white dark:text-zinc-600" />
          </div>
        </button>
      )}
    </div>
  );
};
