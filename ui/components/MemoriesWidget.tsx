import React, { useState, useEffect, useRef } from 'react';
import { Link } from 'react-router-dom';
import ChevronRight from '../svg/chevron-right.svg?react';
import Close from '../svg/close.svg?react';
import { Memory } from '../types';

interface MemoriesWidgetProps {
  memories: Memory[];
}

const MemoriesWidget: React.FC<MemoriesWidgetProps> = ({ memories }) => {
  const [isVisible, setIsVisible] = useState(false);
  const [isHovered, setIsHovered] = useState(false);
  const [isDismissed, setIsDismissed] = useState(true);

  useEffect(() => {
    const today = new Date().toISOString().split('T')[0];
    const lastDismissedDate = localStorage.getItem('rgallery_memories_dismissed_date');

    if (lastDismissedDate === today) {
      setIsDismissed(true);
      return;
    }

    // If not dismissed today, show it
    setIsDismissed(false);

    const timer = setTimeout(() => {
      setIsVisible(true);
    }, 250);
    return () => clearTimeout(timer);
  }, []);

  const handleMemoryClick = (e: React.MouseEvent, date: string) => {
    if (location.pathname === '/') {
      console.log('[MemoriesWidget] Dispatching memory-scroll event for date:', date);
      e.preventDefault();
      window.dispatchEvent(new CustomEvent('memory-scroll', { detail: { date } }));
    }
  };

  const handleDismiss = (e: React.MouseEvent) => {
    e.stopPropagation();
    const today = new Date().toISOString().split('T')[0];
    localStorage.setItem('rgallery_memories_dismissed_date', today);
    setIsVisible(false);
    setIsDismissed(true);
  };

  const getTransformClass = (): string => {
    if (!isVisible) return '-translate-x-[110%]';
    if (isHovered) return 'translate-x-0';
    return '-translate-x-[19.8rem]';
  };

  const handleSpinePointer = (e: React.PointerEvent<HTMLDivElement>) => {
    e.stopPropagation();
    setIsHovered((s) => !s);
  };

  if (isDismissed) return null;

  return (
    <div
      className={`absolute top-1/2 left-0 z-50 mt-32 flex -translate-y-1/2 flex-row-reverse items-start transition-transform duration-500 ${getTransformClass()}`}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <div
        onPointerDown={handleSpinePointer}
        role="button"
        className="bg-primary-600 pointer-events-auto relative z-50 flex h-80 w-10 touch-manipulation flex-col items-center justify-between rounded-r-2xl border-t border-r border-b border-white/20 py-8 text-white shadow-2xl transition-all hover:brightness-110 md:w-12"
      >
        <button
          onClick={handleDismiss}
          className="absolute top-2 right-1/2 z-50 translate-x-1/2 cursor-pointer rounded-full p-1.5 text-white/60 transition-colors hover:bg-white/20 hover:text-white"
          title="Dismiss memories for today"
          aria-label="Dismiss memories for today"
        >
          <Close className="h-4 w-4" />
        </button>

        <div className="mt-4 flex flex-col items-center gap-4">
          <div className="text-primary-600 flex h-6 w-6 flex-col items-center justify-center rounded-full bg-white text-xs font-bold shadow-sm">
            {memories.length}
          </div>
        </div>

        <div className="flex flex-1 items-center justify-center">
          <h2 className="rotate-180 font-bold tracking-[0.2em] whitespace-nowrap uppercase opacity-90 [writing-mode:vertical-rl]">
            On this day
          </h2>
        </div>

        <div className="mb-2 opacity-50">
          <ChevronRight className={`transition-transform duration-300 ${isHovered ? 'rotate-180' : ''}`} />
        </div>
      </div>

      <div className="-mr-1 flex h-80 w-80 flex-col overflow-hidden border-r border-zinc-200 bg-white/95 shadow-2xl dark:border-zinc-700 dark:bg-zinc-900/95">
        <div className="flex-1 space-y-3 overflow-y-auto p-4">
          {memories.map((memory, i) => (
            <Link
              key={i}
              to={`/?date=${memories[i].value}`}
              onClick={(e) => {
                handleMemoryClick(e, memories[i].value);
              }}
              className="group/card flex items-center gap-4 rounded-xl border border-transparent p-2 transition-colors hover:border-zinc-200 hover:bg-zinc-100 dark:hover:border-zinc-700 dark:hover:bg-zinc-800"
            >
              <div
                className="relative h-16 w-16 shrink-0 overflow-hidden rounded-lg shadow-sm transition-shadow group-hover/card:shadow-md"
                style={{
                  backgroundColor: memory.media[0].color,
                }}
              >
                <img
                  srcSet={memory.media[0].srcset}
                  alt={memory.media[0].path}
                  className="h-full w-full object-cover opacity-0 transition-opacity duration-500"
                  sizes="64px"
                  width="64"
                  height="64"
                  onLoad={(e) => (e.currentTarget.style.opacity = '1')}
                  loading="lazy"
                />
                <div className="absolute inset-0 bg-black/10 transition-colors group-hover/card:bg-transparent" />
              </div>

              <div className="flex-1">
                <p className="text-sm font-bold text-zinc-800 dark:text-zinc-200">
                  {memory.key} {memory.key === 1 ? 'year' : 'years'} ago
                </p>
                <p className="mt-0.5 text-xs text-zinc-500">
                  {memory.total} {memory.total === 1 ? 'photo' : 'photos'}
                </p>
              </div>
            </Link>
          ))}
        </div>

        <div className="border-t border-zinc-200 bg-zinc-50 p-4 dark:border-zinc-800 dark:bg-zinc-900">
          <Link
            to="/memories"
            className="bg-primary-600 hover:bg-primary-500 shadow-primary-500/20 flex w-full items-center justify-center gap-2 rounded-lg py-2.5 text-sm font-medium text-white shadow-lg transition-all"
          >
            View all memories
          </Link>
        </div>
      </div>
    </div>
  );
};

export default MemoriesWidget;
