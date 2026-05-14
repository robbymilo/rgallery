import React, { useEffect, useState } from 'react';
import { getMemories } from '../services/memories';
import { Memory } from '../types';
import { useAuth } from '../context/AuthContext';
import { useNavigate, Link } from 'react-router-dom';
import Calendar from '../svg/calendar.svg?react';
import ArrowLeft from '../svg/arrow-left.svg?react';
import ArrowRight from '../svg/arrow-right.svg?react';
import Loading from '../components/Loading';
import Error from '../components/Error';

const MemoryCard: React.FC<{ memory: Memory }> = ({ memory }) => {
  const displayItems = memory.media.slice(0, 4);
  const remainingCount = memory.total - displayItems.length;

  let gridClass = 'grid-cols-2 grid-rows-2';
  if (displayItems.length === 1) {
    gridClass = 'grid-cols-1 grid-rows-1';
  } else if (displayItems.length === 2) {
    gridClass = 'grid-cols-1 grid-rows-2';
  } // 3 and 4: default 2x2

  return (
    <div className="group dark:border-charcoal-700 dark:bg-charcoal-800 overflow-hidden rounded-2xl border border-gray-100 bg-white shadow-sm transition-all duration-300">
      {/* Header */}
      <div className="dark:border-charcoal-700/50 border-b border-gray-100 p-5">
        <h3 className="mb-1 flex items-center text-xl font-bold text-gray-900 transition-colors dark:text-white">
          <Calendar className="mr-2 h-4 w-4" /> {memory.value}
        </h3>
      </div>

      <div
        className={`grid gap-1 ${gridClass} dark:bg-charcoal-900 w-full bg-gray-100`}
        style={{ aspectRatio: '1 / 1' }}
      >
        {displayItems.map((item, index) => {
          const srcset = item.srcset || undefined;
          const hash = item.thumbnailUrl?.match(/\/(\d+)\/400$/)?.[1] ?? '';
          const isOverlay = index === displayItems.length - 1 && remainingCount > 0;

          let cellClass = 'relative overflow-hidden';
          if (displayItems.length === 3 && index === 0) {
            cellClass += ' row-span-2';
          }
          // For 1 image, fill entire grid
          if (displayItems.length === 1) {
            cellClass += ' col-span-1 row-span-1';
          }
          // For 2 images, each takes half vertically
          if (displayItems.length === 2) {
            cellClass += ' col-span-1 row-span-1';
          }
          // Center the third image in 3-image layout
          if (displayItems.length === 3 && index === 2) {
            cellClass += ' flex items-center justify-center';
          } else {
            cellClass += ' block';
          }

          // If overlay, link to date, else link to media
          const to = isOverlay ? `/?date=${memory.value}` : `/media/${item.hash}`;
          return (
            <Link key={`${item.hash}`} to={to} className={cellClass}>
              <div className="dark:bg-charcoal-700 relative h-full w-full bg-gray-200">
                <img
                  alt={item.path}
                  srcSet={srcset}
                  className="absolute inset-0 h-full w-full object-cover transition-transform duration-700"
                  loading="lazy"
                  sizes="450px"
                  style={{ backgroundColor: item.color }}
                />
                {isOverlay && (
                  <div className="absolute inset-0 flex items-center justify-center bg-black/60">
                    <span className="text-xl font-bold text-white">+{remainingCount}</span>
                  </div>
                )}
              </div>
            </Link>
          );
        })}
      </div>

      <Link
        to={`/?date=${memory.value}`}
        className="text-primary-600 dark:text-primary-400 dark:bg-charcoal-800/50 flex items-center justify-between bg-gray-50 p-4 text-xs font-medium tracking-wide uppercase hover:underline"
      >
        View more
      </Link>
    </div>
  );
};

const Memories: React.FC = () => {
  const [memories, setMemories] = useState<Memory[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const { logout } = useAuth();
  const navigate = useNavigate();

  const getDateFromUrl = () => {
    const search = window.location.search;
    const params = new URLSearchParams(search);
    return params.get('date') ?? undefined;
  };

  const [currentDate, setCurrentDate] = useState<string | undefined>(getDateFromUrl());

  useEffect(() => {
    let mounted = true;

    const fetchMemories = async () => {
      try {
        setLoading(true);
        const data = await getMemories(currentDate);
        if (mounted) {
          setMemories(data);
          setError(null);
        }
      } catch (err: unknown) {
        if (!mounted) return;
        console.error('Fetch error:', err);
        if (
          err &&
          typeof err === 'object' &&
          'message' in err &&
          typeof (err as { message?: unknown }).message === 'string'
        ) {
          setError((err as { message: string }).message);
        } else {
          setError('Failed to load memories.');
        }
      } finally {
        if (mounted) setLoading(false);
      }
    };

    fetchMemories();

    return () => {
      mounted = false;
    };
  }, [currentDate, logout, navigate]);

  // listen for browser navigation (back/forward)
  useEffect(() => {
    const onPopState = () => {
      const params = new URLSearchParams(window.location.search);
      setCurrentDate(params.get('date') ?? undefined);
    };
    window.addEventListener('popstate', onPopState);
    return () => window.removeEventListener('popstate', onPopState);
  }, []);

  const goToDate = (date: string | undefined) => {
    const params = new URLSearchParams(window.location.search);
    if (date) {
      params.set('date', date);
    } else {
      params.delete('date');
    }
    const paramString = params.toString();
    const newUrl = paramString ? `${window.location.pathname}?${paramString}` : window.location.pathname;
    window.history.pushState({}, '', newUrl);
    setCurrentDate(date);
  };

  const getAdjacentDate = (date: string | undefined, diff: number) => {
    let baseDate: Date;
    if (date) {
      baseDate = new Date(date);
      if (isNaN(baseDate.getTime())) baseDate = new Date();
    } else {
      baseDate = new Date();
    }
    baseDate.setDate(baseDate.getDate() + diff);
    return baseDate.toISOString().slice(0, 10);
  };

  // Helper to check if a date is in the future (after today)
  const isFutureDate = (dateStr: string) => {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    const d = new Date(dateStr);
    d.setHours(0, 0, 0, 0);
    return d > today;
  };

  if (loading) {
    return <Loading />;
  }

  if (error) {
    return <Error error={error} />;
  }

  return (
    <div className="mx-auto w-[90vw] flex-1 py-8 md:w-[80vw]">
      <div className="pb-12">
        <div className="mb-8">
          <h1>On this day</h1>
          <div className="mt-4 flex gap-4">
            <button
              className="bg-primary-50 text-primary-600 hover:bg-primary-100 dark:bg-primary-500/10 dark:text-primary-400 dark:hover:bg-primary-500/20 flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors"
              onClick={() => goToDate(getAdjacentDate(currentDate, -1))}
            >
              <ArrowLeft className="mr-2 h-5 w-5" />
              {getAdjacentDate(currentDate, -1)}
            </button>
            <button
              className="bg-primary-50 text-primary-600 hover:bg-primary-100 dark:bg-primary-500/10 dark:text-primary-400 dark:hover:bg-primary-500/20 flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors disabled:cursor-not-allowed disabled:opacity-50"
              onClick={() => goToDate(getAdjacentDate(currentDate, 1))}
              disabled={isFutureDate(getAdjacentDate(currentDate, 1))}
            >
              {getAdjacentDate(currentDate, 1)}
              <ArrowRight className="ml-2 h-5 w-5" />
            </button>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-8 md:grid-cols-2 lg:grid-cols-3">
          {memories.map((memory) => (
            <MemoryCard key={memory.key} memory={memory} />
          ))}
          {memories.length === 0 && (
            <div>
              <p>No memories found today.</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default Memories;
