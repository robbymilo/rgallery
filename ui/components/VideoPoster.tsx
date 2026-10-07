import React, { useState } from 'react';
import './VideoFrame.css';

interface VideoPosterProps {
  src?: string;
  title?: string;
  aspectRatio?: number;
  onAspectRatio?: (ratio: number) => void;
}

// Keeps the poster fitted to its image dimensions.
export default function VideoPoster({ src, title, aspectRatio = 16 / 9, onAspectRatio }: VideoPosterProps) {
  const [decoded, setDecoded] = useState<{ src?: string; ratio: number } | null>(null);
  const ready = decoded?.src === src && !!decoded?.ratio;
  const ratio = ready ? decoded.ratio : Number.isFinite(aspectRatio) && aspectRatio > 0 ? aspectRatio : 16 / 9;
  return (
    <div className="gallery-video-stage" data-video-poster>
      <div className="gallery-video-player" style={{ '--video-aspect-ratio': ratio } as React.CSSProperties}>
        <img
          className="gallery-video-poster"
          src={src}
          alt={title || 'Video'}
          draggable={false}
          style={{ opacity: ready ? 1 : 0 }}
          onLoad={(event) => {
            const image = event.currentTarget;
            const ratio = image.naturalWidth / image.naturalHeight;
            setDecoded({ src, ratio });
            onAspectRatio?.(ratio);
          }}
        />
      </div>
    </div>
  );
}
