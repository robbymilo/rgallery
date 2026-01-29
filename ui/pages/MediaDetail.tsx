import React, { useEffect, useState, useCallback, useMemo } from 'react';
import { ApiResponse, MediaItem, ViewMode } from '../types';
import { getMedia } from '../services/media';
import ImageViewer from '../components/ImageViewer';
import ThumbnailStrip from '../components/ThumbnailStrip';
import ExifPanel from '../components/ExifPanel';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { useFullscreen } from '../context/FullscreenContext';

const MediaDetail: React.FC = () => {
  const { mediaID } = useParams<{ mediaID?: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const { setIsFullscreen } = useFullscreen();
  const [data, setData] = useState<ApiResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [viewMode, setViewMode] = useState<ViewMode>(ViewMode.NORMAL);

  const filters = useMemo(() => {
    const params = new URLSearchParams(location.search);
    const filterObj: Record<string, string> = {};
    params.forEach((value, key) => {
      filterObj[key] = value;
    });
    return filterObj;
  }, [location.search]);

  const loadData = useCallback(
    async (id: number) => {
      setIsLoading(true);
      try {
        const result = await getMedia(id, filters);
        setData(result);
      } catch (e) {
        setData(null);
      }
      setIsLoading(false);
    },
    [filters]
  );

  // Load data when mediaID changes
  useEffect(() => {
    if (mediaID && !isNaN(Number(mediaID))) {
      loadData(Number(mediaID));
    }
  }, [mediaID, loadData]);

  // Update document title when media changes
  useEffect(() => {
    const previousTitle = document.title;
    if (data && data.media) {
      document.title = `${data.media.path} | rgallery`;
    } else {
      document.title = previousTitle;
    }
    return () => {
      document.title = previousTitle;
    };
  }, [data && data.media && data.media.path]);

  // Pause all video elements when media changes
  useEffect(() => {
    const videos = document.querySelectorAll('video');
    videos.forEach((video) => {
      video.pause();
      video.currentTime = 0;
    });
  }, [data?.media?.hash]);

  const handleNext = useCallback(() => {
    if (data && data.next.length > 0) {
      const params = new URLSearchParams(filters).toString();
      navigate(`/media/${data.next[0].hash}${params ? `?${params}` : ''}`);
    }
  }, [data, filters, navigate]);

  const handlePrev = useCallback(() => {
    if (data && data.previous.length > 0) {
      const params = new URLSearchParams(filters).toString();
      navigate(`/media/${data.previous[data.previous.length - 1].hash}${params ? `?${params}` : ''}`);
    }
  }, [data, filters, navigate]);

  const toggleFullscreen = useCallback(() => {
    setViewMode((prev) => (prev === ViewMode.NORMAL ? ViewMode.FULLSCREEN : ViewMode.NORMAL));
  }, []);

  // Sync fullscreen context with viewMode and clean up on unmount
  useEffect(() => {
    setIsFullscreen(viewMode === ViewMode.FULLSCREEN);
    return () => {
      setIsFullscreen(false);
    };
  }, [viewMode, setIsFullscreen]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (['INPUT', 'TEXTAREA'].includes((e.target as HTMLElement).tagName)) return;

      switch (e.key) {
        case 'ArrowRight':
          handleNext();
          break;
        case 'ArrowLeft':
          handlePrev();
          break;
        case 'f':
        case 'F':
          toggleFullscreen();
          break;
        case 'Escape':
          if (viewMode === ViewMode.FULLSCREEN) setViewMode(ViewMode.NORMAL);
          break;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [handleNext, handlePrev, toggleFullscreen, viewMode]);

  if (!data && isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full"></div>
      </div>
    );
  }

  if (!data) return null;

  return (
    <div>
      <div
        className={`flex h-screen flex-col bg-zinc-700 text-zinc-900 dark:bg-zinc-700 dark:text-zinc-100 ${viewMode === ViewMode.FULLSCREEN ? 'overflow-hidden' : ''}`}
      >
        {/* keep padding on large screens */}
        <div
          className={`transition-all duration-200 ${viewMode === ViewMode.FULLSCREEN ? 'overflow-hidden py-0' : 'py-4'}`}
        >
          <main className="flex w-full grow flex-col items-center">
            <ImageViewer
              media={data.media}
              previous={data.previous}
              next={data.next}
              viewMode={viewMode}
              onNext={handleNext}
              onPrev={handlePrev}
              onToggleFullscreen={toggleFullscreen}
            />
          </main>
        </div>
        <div className="flex w-full grow flex-col items-center bg-zinc-700">
          {viewMode === ViewMode.NORMAL && (
            <div className="mt-2">
              <ThumbnailStrip
                previous={data.previous}
                current={data.media}
                next={data.next}
                onNext={handleNext}
                onPrev={handlePrev}
              />
            </div>
          )}

          {viewMode === ViewMode.NORMAL && (
            <div className="flex w-full flex-grow flex-col items-center bg-white pb-8 dark:bg-zinc-900">
              <ExifPanel media={data.media} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default MediaDetail;
