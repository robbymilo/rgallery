import React, { useRef, useState } from 'react';
import Hls from 'hls.js';
interface VideoThumbProps {
  hlsUrl: string;
  poster?: string;
  alt?: string;
}

const VideoThumb: React.FC<VideoThumbProps> = ({ hlsUrl, poster, alt }) => {
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const hlsInstanceRef = useRef<any>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isHovered, setIsHovered] = useState(false);
  const [canShowVideo, setCanShowVideo] = useState(false);

  const handlePlay = () => setIsPlaying(true);
  const handlePause = () => setIsPlaying(false);
  const handleCanPlay = () => setCanShowVideo(true);
  const handleLoadedData = () => setCanShowVideo(true);

  const handleMouseEnter = () => {
    setIsHovered(true);
    if (!videoRef.current) return;
    const Hls = (window as any).Hls;
    if (Hls && Hls.isSupported()) {
      if (!hlsInstanceRef.current) {
        const hls = new Hls({ maxBufferLength: 3, debug: false });
        hlsInstanceRef.current = hls;

        hls.on(Hls.Events.MANIFEST_PARSED, () => {
          videoRef.current?.play?.();
        });

        hls.on(Hls.Events.ERROR, (e, data) => {
          console.error('HLS error', data);
        });

        hls.loadSource(hlsUrl);
        hls.attachMedia(videoRef.current);
      }
    }
  };

  const handleMouseLeave = () => {
    setIsHovered(false);
    setIsPlaying(false);
    setCanShowVideo(false);
    if (videoRef.current) {
      videoRef.current.pause();
      videoRef.current.currentTime = 0;
    }
    if (hlsInstanceRef.current) {
      hlsInstanceRef.current.destroy();
      hlsInstanceRef.current = null;
    }
    if (videoRef.current) {
      try {
        if (videoRef.current.src) videoRef.current.removeAttribute('src');
      } catch (e) {}
    }
  };

  const showPoster = !(canShowVideo && isPlaying);

  return (
    <div className="relative h-full w-full" onMouseEnter={handleMouseEnter} onMouseLeave={handleMouseLeave}>
      {poster && (
        <img
          src={poster}
          alt={alt}
          className={`pointer-events-none absolute inset-0 z-10 h-full w-full object-cover opacity-0 transition-opacity duration-500`}
          draggable={false}
          onLoad={(e) => (e.currentTarget.style.opacity = '1')}
        />
      )}
      <video
        ref={videoRef}
        className={`absolute inset-0 z-20 h-full w-full object-cover ${canShowVideo && isPlaying ? 'opacity-100' : 'opacity-0'}`}
        style={{ transition: 'none' }}
        muted
        loop
        playsInline
        autoPlay
        preload="metadata"
        onPlay={handlePlay}
        onPause={handlePause}
        onCanPlay={handleCanPlay}
        onLoadedData={handleLoadedData}
      />
      {!isPlaying && (
        <div className="pointer-events-none absolute inset-0 z-30 flex items-center justify-center">
          <div className="flex h-8 w-8 items-center justify-center rounded-full border-2 border-white bg-black/50">
            <div className="ml-1 h-0 w-0 border-t-8 border-b-8 border-l-12 border-t-transparent border-b-transparent border-l-white"></div>
          </div>
        </div>
      )}
    </div>
  );
};

export default VideoThumb;
