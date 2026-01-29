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
}

interface YearGroup {
  year: number;
  months: MonthNode[];
  count: number;
}

const TICK_HEIGHT = 8; // h-3 = 12px
const PADDING_TOP = 40; // py-10 = 40px

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

    timeline.sort().forEach((item) => {
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
    });

    return { groups: grps, flatNodes: nodes, totalItems };
  }, [timeline]);

  const startAutoScroll = () => {
    if (animationFrameRef.current) return;

    const loop = () => {
      if (scrollSpeedRef.current !== 0 && scrollContainerRef.current) {
        scrollContainerRef.current.scrollTop += scrollSpeedRef.current;
        if (lastPointerPosRef.current) {
          processPointer(lastPointerPosRef.current.x, lastPointerPosRef.current.y);
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

    handleInteraction(e.clientX, e.clientY);
  };

  const handlePointerMove = (e: React.PointerEvent) => {
    if (!isDragging || e.pointerType === 'touch') return;
    e.preventDefault();
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
      handleInteraction(e.touches[0].clientX, e.touches[0].clientY);
    }
  };

  const handleTouchMove = (e: React.TouchEvent) => {
    if (!isDragging) return;
    if (e.cancelable) e.preventDefault();
    if (e.touches.length > 0) {
      handleInteraction(e.touches[0].clientX, e.touches[0].clientY);
    }
  };

  const handleTouchEnd = (e: React.TouchEvent) => {
    setIsDragging(false);
    stopAutoScroll();
    if (lastScrubbedDateRef.current) {
      onDateSelect(lastScrubbedDateRef.current);
      lastScrubbedDateRef.current = null;
    }
  };

  const handleInteraction = (clientX: number, clientY: number) => {
    lastPointerPosRef.current = { x: clientX, y: clientY };
    updateThumbFromPointer(clientY);

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

    processPointer(clientX, clientY);
  };

  const endInteraction = (target: Element, pointerId: number) => {
    setIsDragging(false);
    stopAutoScroll();
    target.releasePointerCapture(pointerId);
    if (lastScrubbedDateRef.current) {
      onDateSelect(lastScrubbedDateRef.current);
      lastScrubbedDateRef.current = null;
    }
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
              <div className="absolute top-0 right-16 pr-4 text-[10px] leading-none font-bold text-zinc-500">
                {group.year}
              </div>
              {group.months.map((m) => {
                const ratio = Math.round((m.total / totalItems) * 1000);

                return (
                  <div key={m.monthIndex} className="z-10 flex h-2 w-12 items-center justify-end">
                    <div className="h-1 bg-zinc-600 transition-all" style={{ width: `${ratio}px` }} />
                  </div>
                );
              })}
            </div>
          ))}
          <div className="h-64" />
        </div>
      </div>

      {thumbTop !== null && (
        <button
          className="pointer-events-auto absolute right-0 z-20 w-auto cursor-grab transition-none focus:outline-none active:cursor-grabbing"
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
          <div className="pointer-events-none flex min-w-max items-center gap-1.5 rounded-l-full bg-white px-3 py-1 text-black shadow-lg">
            <Scrubber className="h-3 w-3 text-black" />
            <span className="text-[10px] leading-none font-bold whitespace-nowrap tabular-nums">
              {currentDate.getFullYear()}{' '}
              <span className="text-black">{currentDate.toLocaleString('default', { month: 'short' })}</span>
            </span>
          </div>
        </button>
      )}
    </div>
  );
};
