import React from 'react';

const StarIcon = ({ className, filled }: { className?: string; filled?: boolean }) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="24"
    height="24"
    viewBox="0 0 24 24"
    fill={filled ? 'currentColor' : 'none'}
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
    className={className}
  >
    <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon>
  </svg>
);

interface StarRatingProps {
  rating?: number;
  onRate?: (rating: number) => void;
  interactive?: boolean;
  size?: 'sm' | 'md' | 'lg';
}

const StarRating: React.FC<StarRatingProps> = ({ rating = 1, onRate, interactive = false, size = 'md' }) => {
  const [hoverRating, setHoverRating] = React.useState<number | null>(null);

  const displayRating = hoverRating !== null ? hoverRating : rating;

  const sizeClass = {
    sm: 'w-3 h-3',
    md: 'w-4 h-4',
    lg: 'w-6 h-6',
  }[size];

  return (
    <div className="flex items-center gap-0.5" onMouseLeave={() => interactive && setHoverRating(null)}>
      {[1, 2, 3, 4, 5].map((starValue) => {
        const isFilled = starValue <= displayRating;
        return (
          <button
            key={starValue}
            type="button"
            disabled={!interactive}
            onClick={() => onRate && onRate(starValue)}
            onMouseEnter={() => interactive && setHoverRating(starValue)}
            className={` ${interactive ? 'cursor-pointer transition-transform hover:scale-110' : 'cursor-default'} ${isFilled ? 'text-yellow-400' : 'text-neutral-600'} `}
          >
            <StarIcon filled={isFilled} className={sizeClass} />
          </button>
        );
      })}
    </div>
  );
};

export default StarRating;
