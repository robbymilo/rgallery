import React, { useState } from 'react';
import { FilterState, SortOption } from '../types';
import StarRating from './StarRating';
import Close from '../svg/close.svg?react';
import Filter from '../svg/filter.svg?react';
import Magnifier from '../svg/magnifier.svg?react';

interface FilterBarProps {
  filters: FilterState;
  onFilterChange: (newFilters: FilterState) => void;
  totalItems: number;
}

const FilterBar: React.FC<FilterBarProps> = ({ filters, onFilterChange, totalItems }) => {
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value;

    // Look for tokens like `tag:value`, `camera:value`, `lens:value`, `software:value`, `focallength35:123`
    const parts = raw.split(/\s+/).filter(Boolean);
    let remainingParts = [...parts];
    let foundTag: string | undefined = undefined;
    let foundCamera: string | undefined = undefined;
    let foundLens: string | undefined = undefined;
    let foundSoftware: string | undefined = undefined;
    let foundFolder: string | undefined = undefined;
    let foundFocal: number | undefined = undefined;

    for (let i = 0; i < parts.length; i++) {
      const p = parts[i];
      let m;
      m = p.match(/^tag:(.+)$/i);
      if (m) {
        foundTag = m[1];
        remainingParts.splice(remainingParts.indexOf(p), 1);
        continue;
      }
      m = p.match(/^camera:(.+)$/i);
      if (m) {
        foundCamera = m[1];
        remainingParts.splice(remainingParts.indexOf(p), 1);
        continue;
      }
      m = p.match(/^lens:(.+)$/i);
      if (m) {
        foundLens = m[1];
        remainingParts.splice(remainingParts.indexOf(p), 1);
        continue;
      }
      m = p.match(/^software:(.+)$/i);
      if (m) {
        foundSoftware = m[1];
        remainingParts.splice(remainingParts.indexOf(p), 1);
        continue;
      }
      m = p.match(/^folder:(.+)$/i);
      if (m) {
        foundFolder = m[1];
        remainingParts.splice(remainingParts.indexOf(p), 1);
        continue;
      }
      m = p.match(/^focallength35:(\d+)$/i);
      if (m) {
        foundFocal = parseInt(m[1], 10);
        remainingParts.splice(remainingParts.indexOf(p), 1);
        continue;
      }
    }

    const newSearch = remainingParts.join(' ');
    onFilterChange({
      ...filters,
      searchQuery: newSearch,
      tag: foundTag,
      camera: foundCamera,
      lens: foundLens,
      software: foundSoftware,
      folder: foundFolder,
      focallength35: foundFocal,
    });
  };

  const handleSortChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    onFilterChange({ ...filters, sortBy: e.target.value as SortOption });
  };

  const updateFilter = <K extends keyof FilterState>(key: K, value: FilterState[K]) => {
    onFilterChange({ ...filters, [key]: value });
  };

  const clearFilters = () => {
    // create a new object to force state update
    onFilterChange({
      searchQuery: '',
      minRating: 1,
      tag: undefined,
      folder: undefined,
      camera: undefined,
      lens: undefined,
      software: undefined,
      focallength35: undefined,
      mediaType: 'all',
      sortBy: 'date-desc',
      __forceUpdate: Date.now(),
    } as FilterState);
    setIsMobileMenuOpen(false);
  };

  const hasActiveFilters =
    filters.searchQuery !== '' ||
    filters.minRating > 1 ||
    Boolean(filters.tag) ||
    Boolean(filters.folder) ||
    Boolean(filters.camera) ||
    Boolean(filters.lens) ||
    Boolean(filters.software) ||
    Boolean(filters.focallength35) ||
    filters.mediaType !== 'all' ||
    filters.sortBy !== 'date-desc';

  const FilterControls = ({ mobile = false }: { mobile?: boolean }) => (
    <>
      {!mobile &&
        (hasActiveFilters ? (
          <button
            onClick={clearFilters}
            className="ml-1 flex h-9 shrink-0 cursor-pointer items-center gap-1.5 rounded-lg border border-red-500 bg-red-100 px-3 text-xs font-medium text-red-700 transition-all hover:border-red-600 hover:bg-red-200 hover:text-red-900 hover:underline dark:border-red-900/30 dark:bg-red-950/20 dark:text-red-400 dark:hover:border-red-900/50 dark:hover:bg-red-900/40 dark:hover:text-red-300"
          >
            <Close className="h-3 w-3" />
            <span className="whitespace-nowrap">Clear</span>
          </button>
        ) : (
          <div className="ml-1 h-9 px-3" style={{ visibility: 'hidden', minWidth: '74px' }} />
        ))}
      {/* type toggle */}
      <div
        className={`flex shrink-0 rounded-lg border border-neutral-200 bg-white p-1 dark:border-neutral-800 dark:bg-neutral-900/80 ${mobile ? 'h-9 w-full justify-between' : 'h-9'}`}
      >
        {(
          [
            { value: 'all', label: 'All' },
            { value: 'image', label: 'Images' },
            { value: 'video', label: 'Videos' },
          ] as const
        ).map((type) => (
          <button
            key={type.value}
            onClick={() => updateFilter('mediaType', type.value as FilterState['mediaType'])}
            className={`flex h-full cursor-pointer items-center justify-center rounded-md px-3 text-[11px] font-medium whitespace-nowrap transition-all hover:underline ${mobile ? 'flex-1' : ''} ${
              filters.mediaType === type.value
                ? 'bg-neutral-100 text-neutral-900 underline shadow-sm ring-1 ring-black/30 dark:bg-neutral-800 dark:text-white'
                : 'text-neutral-700 hover:text-neutral-900 dark:text-neutral-500 dark:hover:text-neutral-300'
            }`}
          >
            {type.label}
          </button>
        ))}
      </div>

      {/* stars */}
      <div
        className={`flex shrink-0 items-center gap-2 rounded-lg border border-neutral-200 px-3 dark:border-neutral-800 dark:bg-neutral-900/50 ${mobile ? 'h-9 w-full justify-center' : 'h-9'}`}
      >
        <StarRating rating={filters.minRating} interactive onRate={(r) => updateFilter('minRating', r)} size="md" />
      </div>

      {/* sort */}
      {/*<div className={`relative shrink-0 ${mobile ? 'w-full' : ''}`}>
        <select
          value={filters.sortBy}
          onChange={handleSortChange}
          className={`focus:ring-primary-500 h-9 cursor-pointer appearance-none rounded-lg border border-neutral-200 bg-white text-xs font-medium text-neutral-900 transition-colors hover:border-neutral-400 focus:ring-1 focus:outline-none dark:border-neutral-800 dark:bg-neutral-900/50 dark:text-neutral-300 dark:hover:border-neutral-600 ${mobile ? 'w-full px-3' : 'pr-8 pl-3'}`}
        >
          <option value="date-desc">Date (newest)</option>
          <option value="date-asc">Date (oldest)</option>
          <option value="modified-desc">Modified (newest)</option>
          <option value="modified-asc">Modified (oldest)</option>
        </select>
        <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center px-2 text-neutral-400 dark:text-neutral-500">
          <ChevronDown className="h-3 w-3" />
        </div>
      </div>*/}
    </>
  );

  return (
    <div className="dark:bg-charcoal-900 sticky top-0 z-40 w-full border-b border-neutral-200 bg-white/95 backdrop-blur transition-all dark:border-white/5">
      <div className="mx-auto w-[90vw] py-3 md:w-[80vw]">
        <div className="flex items-center justify-between gap-4">
          <div className="flex min-w-fit items-center">
            <h1 className="h1-small">
              Timeline{' '}
              <span
                className="ml-1 inline-block min-w-[75px] text-left font-mono text-sm font-normal text-neutral-500 dark:text-neutral-500"
                title="Total items"
                aria-label="Total items"
              >
                {totalItems.toLocaleString('en-GB')}
              </span>
            </h1>
          </div>

          <div className="flex w-full max-w-[700px] flex-1 items-center justify-end gap-2 lg:gap-3">
            {/* desktop) */}
            <div className="hidden items-center gap-2 lg:flex">
              <FilterControls mobile={false} />
            </div>

            {/* mobile clear */}
            {hasActiveFilters && (
              <button
                className="flex h-9 w-9 items-center justify-center rounded-lg border border-red-500 bg-red-100 text-red-700 transition-colors hover:border-red-600 hover:bg-red-200 hover:text-red-900 lg:hidden dark:border-red-900/30 dark:bg-red-950/20 dark:text-red-400 dark:hover:border-red-900/50 dark:hover:bg-red-900/40 dark:hover:text-red-300"
                onClick={clearFilters}
                aria-label="Clear filters"
              >
                <Close className="h-4 w-4" />
              </button>
            )}

            {/* filter */}
            <button
              className={`flex h-9 w-9 items-center justify-center rounded-lg border transition-colors lg:hidden ${
                isMobileMenuOpen || (hasActiveFilters && filters.searchQuery === '')
                  ? 'border-primary-500 bg-primary-100 text-primary-600 dark:border-primary-600 dark:bg-primary-500/10 dark:text-primary-600'
                  : 'border-neutral-200 bg-white text-neutral-700 hover:text-neutral-900 dark:border-neutral-800 dark:bg-neutral-900/50 dark:text-neutral-500 dark:hover:text-neutral-200'
              }`}
              onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
              aria-label="Show filter controls"
            >
              <Filter className="h-5 w-5" />
            </button>

            {/* search */}
            <div className="group relative transition-all sm:max-w-xs md:w-[200px]">
              <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                <Magnifier className="group-focus-within:text-primary-600 dark:group-focus-within:text-primary-400 h-4 w-4 text-neutral-400 transition-colors dark:text-neutral-500" />
              </div>
              {(() => {
                const parts: string[] = [];
                if (filters.searchQuery) parts.push(filters.searchQuery);
                if (filters.tag) parts.push(`tag:${filters.tag}`);
                if (filters.folder) parts.push(`folder:${filters.folder}`);
                if (filters.camera) parts.push(`camera:${filters.camera}`);
                if (filters.lens) parts.push(`lens:${filters.lens}`);
                if (filters.software) parts.push(`software:${filters.software}`);
                if (filters.focallength35) parts.push(`focallength35:${filters.focallength35}`);
                const inputValue = parts.join(' ').trim();

                return (
                  <input
                    type="text"
                    placeholder="Search..."
                    className="focus:border-primary-500 focus:ring-primary-500 block h-9 w-full rounded-lg border border-neutral-200 bg-white pr-3 pl-9 text-sm text-[10px] leading-5 text-neutral-900 placeholder-neutral-400 transition-all focus:ring-0 focus:outline-none dark:border-neutral-800 dark:bg-neutral-900/50 dark:text-neutral-200 dark:placeholder-neutral-500"
                    value={inputValue}
                    onChange={handleSearchChange}
                  />
                );
              })()}
            </div>
          </div>
        </div>

        {/* mobile dropdown */}
        {isMobileMenuOpen && (
          <div className="animate-in fade-in slide-in-from-top-1 mt-3 border-t border-neutral-200 pt-3 duration-200 lg:hidden dark:border-white/5">
            <div className="flex flex-col gap-3">
              <FilterControls mobile={true} />
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default FilterBar;
