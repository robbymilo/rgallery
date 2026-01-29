import React from 'react';
import { MediaItem } from '../types';
import { Link } from 'react-router';

interface ThumbnailStripProps {
  previous: MediaItem[];
  current: MediaItem;
  next: MediaItem[];
  onPrev: () => void;
  onNext: () => void;
}

const Thumbnail: React.FC<{ item: MediaItem; isActive?: boolean; onClick: () => void }> = ({
  item,
  isActive,
  onClick,
}) => {
  return (
    <Link
      to={'/media/' + item.hash + window.location.search}
      onClick={(e) => {
        e.stopPropagation();
        onClick();
      }}
      className={`relative h-8 w-8 flex-shrink-0 overflow-hidden rounded-md border border-zinc-300 transition-all duration-300 md:h-16 md:w-16 dark:border-zinc-700 ${isActive ? 'opacity-100' : 'opacity-40 hover:opacity-80'}`}
      style={{
        backgroundColor: item.color,
      }}
    >
      <img
        srcSet={item.srcset}
        alt={`${item.path}`}
        className="h-full w-full object-cover opacity-0 transition-opacity duration-500"
        loading="lazy"
        sizes="64px"
        width="64"
        height="64"
        onLoad={(e) => (e.currentTarget.style.opacity = '1')}
      />
      {item.type === 'video' && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/30">
          <div className="flex h-3 w-3 items-center justify-center rounded-full border-2 border-white bg-black/50 md:h-6 md:w-6">
            <div className="ml-0.5 h-0 w-0 border-t-2 border-b-2 border-l-3 border-t-transparent border-b-transparent border-l-white md:ml-1 md:border-t-4 md:border-b-4 md:border-l-6"></div>
          </div>
        </div>
      )}
    </Link>
  );
};

const ThumbnailStrip: React.FC<ThumbnailStripProps> = ({ previous, current, next, onPrev, onNext }) => {
  const handleThumbnailClick = (item: MediaItem) => {
    const allItems = [...previous, current, ...next];
    const targetIndex = allItems.findIndex((mediaItem) => mediaItem.hash === item.hash);
  };

  const leftItems = previous?.slice(0, 3) || [];
  const rightItems = next?.slice(0, 3) || [];

  return (
    <div className="flex w-full items-center justify-center gap-6 pb-6 select-none">
      <div className="flex items-center gap-3 overflow-hidden px-12">
        {[...leftItems, current, ...rightItems].map((item) => (
          <Thumbnail
            key={item.hash}
            item={item}
            isActive={item.hash === current.hash}
            onClick={() => handleThumbnailClick(item)}
          />
        ))}
      </div>
    </div>
  );
};

export default ThumbnailStrip;
