import React, { useEffect, useRef, useState } from 'react';

interface VideoThumbProps {
  hlsUrl: string;
  poster?: string;
  alt?: string;
}

// Plays a short, silent preview while a gallery item is hovered.
const VideoThumb: React.FC<VideoThumbProps> = ({ hlsUrl, poster, alt }) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const hoveredRef = useRef(false);
  const requestRef = useRef(0);
  const [playing, setPlaying] = useState(false);
  const [loading, setLoading] = useState(false);

  // Stops the preview and releases its media source.
  const stop = () => {
    hoveredRef.current = false;
    requestRef.current++;
    const video = videoRef.current;
    if (video) {
      video.pause();
      video.removeAttribute('src');
      video.load();
    }
  };

  useEffect(() => () => stop(), [hlsUrl]);

  // Starts a hover preview unless reduced motion is enabled.
  const hover = () => {
    stop();
    setPlaying(false);
    setLoading(false);
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    const video = videoRef.current;
    if (!video) return;
    hoveredRef.current = true;
    const request = requestRef.current;
    setLoading(true);
    video.src = hlsUrl.replace(/\/index\.m3u8(?:\?.*)?$/, '/preview.mp4');
    void video.play().catch(() => {
      if (request !== requestRef.current) return;
      setPlaying(false);
      setLoading(false);
    });
  };

  return (
    <div
      className="relative h-full w-full"
      onMouseEnter={hover}
      onMouseLeave={() => {
        stop();
        setPlaying(false);
        setLoading(false);
      }}
    >
      {poster && (
        <img
          src={poster}
          alt={alt}
          className="pointer-events-none absolute inset-0 h-full w-full object-cover"
          draggable={false}
        />
      )}
      <video
        ref={videoRef}
        muted
        loop
        playsInline
        preload="none"
        aria-label={alt || 'Video preview'}
        className={`pointer-events-none absolute inset-0 h-full w-full object-cover ${playing ? 'opacity-100' : 'opacity-0'}`}
        onPlaying={() => {
          if (!hoveredRef.current) return;
          setPlaying(true);
          setLoading(false);
        }}
        onWaiting={() => {
          if (hoveredRef.current) setLoading(true);
        }}
        onError={() => {
          setPlaying(false);
          setLoading(false);
        }}
      />
      {(!playing || loading) && (
        <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
          <div
            role={loading ? 'status' : undefined}
            aria-label={loading ? 'Loading video preview' : undefined}
            aria-hidden={loading ? undefined : true}
            className={`flex h-8 w-8 items-center justify-center rounded-full border-2 border-white bg-black/50 ${loading ? 'animate-spin border-t-transparent' : ''}`}
          >
            {!loading && (
              <div className="ml-1 h-0 w-0 border-t-8 border-b-8 border-l-12 border-t-transparent border-b-transparent border-l-white" />
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default VideoThumb;
