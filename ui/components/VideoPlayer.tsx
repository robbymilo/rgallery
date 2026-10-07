import React, { useEffect, useRef, useState } from 'react';
import Hls from 'hls.js';
import { Container } from '@videojs/react';
import { Video, VideoPlayer as VideoJSPlayer } from '@videojs/react/video';
import VideoControls from './VideoControls';
import VideoPoster from './VideoPoster';
import { qualityKey, readQuality, resolutionLabel, resumePosition, videoJSON, type VideoQuality } from '../lib/video';

interface VideoInfo {
  duration: number;
  width: number;
  height: number;
  version: string;
  defaultQuality: string;
  profiles: { id: VideoQuality; label: string; longEdge: number; maxRate: number }[];
  direct: { url: string; type: string } | null;
}

interface VideoPlayerProps {
  hash: number | string;
  active: boolean;
  poster?: string;
  title?: string;
  aspectRatio?: number;
}

// Plays gallery videos with quality selection and saved positions.
const VideoPlayer: React.FC<VideoPlayerProps> = ({ hash, active, poster, title, aspectRatio = 16 / 9 }) => {
  const videoRef = useRef<HTMLVideoElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const hlsRef = useRef<Hls | null>(null);
  const requestRef = useRef<AbortController | null>(null);
  const metadataRef = useRef<VideoInfo | null>(null);
  const mountedRef = useRef(false);
  const resumeRef = useRef(0);
  const playingRef = useRef(false);
  const hasPlayedRef = useRef(false);
  const saveAtRef = useRef(0);
  const switchRef = useRef(false);
  const pausedLoadingRef = useRef(false);
  const gestureRef = useRef<{ x: number; y: number; dragged: boolean } | null>(null);
  const [info, setInfo] = useState<VideoInfo | null>(null);
  const [quality, setQuality] = useState<VideoQuality>(readQuality);
  const [started, setStarted] = useState(false);
  const [status, setStatus] = useState('');
  const [error, setError] = useState('');
  const [resume, setResume] = useState(0);
  const [currentQuality, setCurrentQuality] = useState('');
  const [position, setPosition] = useState(0);
  const [decodedAspectRatio, setDecodedAspectRatio] = useState(0);
  const [hasFrame, setHasFrame] = useState(false);
  const ratio = decodedAspectRatio || (info ? info.width / info.height : aspectRatio);
  const base = `/api/transcode/${hash}`;

  // Saves the current position for the next visit.
  const savePosition = () => {
    const video = videoRef.current;
    const metadata = metadataRef.current;
    if (!video || !metadata) return;
    const key = `rgallery:video-position:${hash}:${metadata.version}`;
    try {
      if (video.currentTime <= 0 || video.ended || video.currentTime >= metadata.duration - 2)
        localStorage.removeItem(key);
      else localStorage.setItem(key, String(video.currentTime));
    } catch {
      /* Playback does not depend on storage access. */
    }
  };

  // Stops playback and releases the current stream.
  const detach = () => {
    pausedLoadingRef.current = false;
    hasPlayedRef.current = false;
    hlsRef.current?.destroy();
    hlsRef.current = null;
    const video = videoRef.current;
    if (video) {
      if (mountedRef.current) setHasFrame(false);
      video.pause();
      video.removeAttribute('src');
      video.load();
    }
  };

  // Prepares and attaches the selected quality at the requested position.
  const load = async (nextQuality: VideoQuality, position: number, autoplay: boolean, forceHLS = false) => {
    requestRef.current?.abort();
    const controller = new AbortController();
    requestRef.current = controller;
    switchRef.current = true;
    setError('');
    setStatus('Preparing video…');
    try {
      const metadata = await videoJSON<VideoInfo>(`${base}/info?quality=${nextQuality}`, controller.signal);
      if (controller.signal.aborted) return;
      metadataRef.current = metadata;
      setInfo(metadata);
      const video = videoRef.current;
      if (!video) return;
      position = Math.max(0, Math.min(position, metadata.duration - 0.1));
      let url: string;
      const direct = !forceHLS && metadata.direct && video.canPlayType(metadata.direct.type);
      if (direct) url = metadata.direct!.url;
      else {
        const prepared = await videoJSON<{ url: string }>(
          `${base}/prepare?quality=${nextQuality}&start=${position}`,
          controller.signal
        );
        url = prepared.url;
      }
      if (controller.signal.aborted || !mountedRef.current) return;
      // Keep playing during preparation, then switch at the latest playback position.
      const start =
        video.src && !forceHLS ? Math.min(video.currentTime || position, metadata.duration - 0.1) : position;
      const shouldPlay = forceHLS ? autoplay : video.src ? !video.paused : autoplay;
      detach();
      resumeRef.current = start;
      playingRef.current = shouldPlay;
      let recovered = false;
      if (!direct && Hls.isSupported()) {
        const hls = new Hls({
          maxBufferLength: 6,
          maxMaxBufferLength: 8,
          backBufferLength: 8,
          maxBufferSize: 4 * 1024 * 1024,
          startPosition: start,
          startLevel: 0,
          fragLoadingTimeOut: 60000,
          manifestLoadingTimeOut: 60000,
          levelLoadingTimeOut: 60000,
          fragLoadingMaxRetry: 2,
          capLevelToPlayerSize: true,
          autoStartLoad: false,
        });
        hlsRef.current = hls;
        hls.on(Hls.Events.MANIFEST_PARSED, () => {
          hls.startLoad(start);
        });
        hls.on(Hls.Events.LEVEL_SWITCHED, (_, data) => {
          const level = hls.levels[data.level];
          if (level) setCurrentQuality(`${Math.min(level.width, level.height)}p`);
        });
        hls.on(Hls.Events.ERROR, (_, data) => {
          if (!data.fatal) return;
          if (data.type === Hls.ErrorTypes.MEDIA_ERROR && !recovered) {
            recovered = true;
            hls.recoverMediaError();
            return;
          }
          setStatus('');
          setError('Playback interrupted. Retry to continue from this position.');
          hls.stopLoad();
        });
        hls.attachMedia(video);
        hls.loadSource(url);
      } else if (direct || video.canPlayType('application/vnd.apple.mpegurl')) {
        video.src = url;
        // The server also enforces quality limits for native HLS.
        video.load();
        setCurrentQuality('');
      } else throw new Error('This browser does not support HLS video playback.');
      video.dataset.direct = direct ? 'true' : 'false';
    } catch (e) {
      if (controller.signal.aborted || !mountedRef.current) return;
      setStatus('');
      setError(e instanceof Error ? e.message : 'Unable to play this video.');
    } finally {
      if (!controller.signal.aborted) switchRef.current = false;
    }
  };

  useEffect(() => {
    mountedRef.current = true;
    if (!active)
      return () => {
        mountedRef.current = false;
      };
    setStarted(false);
    setStatus('');
    setError('');
    setInfo(null);
    setCurrentQuality('');
    setResume(0);
    setPosition(0);
    setDecodedAspectRatio(0);
    const preferredQuality = readQuality();
    setQuality(preferredQuality);
    const controller = new AbortController();
    videoJSON<VideoInfo>(`${base}/info?quality=${preferredQuality}`, controller.signal)
      .then((metadata) => {
        if (controller.signal.aborted) return;
        metadataRef.current = metadata;
        setInfo(metadata);
        let position = 0;
        try {
          position = resumePosition(
            localStorage.getItem(`rgallery:video-position:${hash}:${metadata.version}`),
            metadata.duration
          );
        } catch {
          /* optional */
        }
        setResume(position);
      })
      .catch((e) => {
        if (!controller.signal.aborted) setError(e.message);
      });
    return () => {
      mountedRef.current = false;
      controller.abort();
      requestRef.current?.abort();
      savePosition();
      detach();
    };
  }, [hash, active]);

  // Starts playback from the saved or requested position.
  const begin = (position = resume) => {
    setStarted(true);
    setPosition(position);
    void load(quality, position, true);
  };

  // Starts, pauses, or resumes playback.
  const togglePlayback = () => {
    const video = videoRef.current;
    if (!started) {
      if (info) begin();
    } else if (video?.readyState) {
      if (video.paused) void video.play().catch(() => setStatus(''));
      else video.pause();
    }
  };

  // Clears the saved position and plays from the beginning.
  const restart = () => {
    setResume(0);
    setPosition(0);
    const video = videoRef.current;
    if (started && video?.readyState) {
      video.currentTime = 0;
      savePosition();
      void video.play().catch(() => setStatus(''));
    } else {
      requestRef.current?.abort();
      switchRef.current = true;
      detach();
      savePosition();
      begin(0);
    }
  };

  // Marks pointer movement that should count as a gallery drag.
  const trackGesture = (event: React.PointerEvent) => {
    const gesture = gestureRef.current;
    if (gesture && Math.hypot(event.clientX - gesture.x, event.clientY - gesture.y) > 5) gesture.dragged = true;
  };

  // Matches the player to the decoded video dimensions.
  const updateDimensions = () => {
    const video = videoRef.current;
    if (video?.videoWidth && video.videoHeight) setDecodedAspectRatio(video.videoWidth / video.videoHeight);
  };

  // Shows the video once a decoded frame is available.
  const showFrame = () => {
    const video = videoRef.current;
    // Use readyState because Firefox may skip frame callbacks while the video is transparent.
    if (mountedRef.current && video && video.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA) setHasFrame(true);
  };

  // Saves the quality choice and switches the current stream.
  const selectQuality = (value: VideoQuality) => {
    setQuality(value);
    try {
      localStorage.setItem(qualityKey, value);
    } catch {
      /* optional */
    }
    if (started) void load(value, videoRef.current?.currentTime || 0, !videoRef.current?.paused);
  };

  // Restores the intended position after media metadata loads.
  const onMetadata = () => {
    const video = videoRef.current;
    if (!video) return;
    updateDimensions();
    if (resumeRef.current > 0 && Number.isFinite(video.duration))
      video.currentTime = Math.min(resumeRef.current, video.duration - 0.1);
    resumeRef.current = 0;
    const shouldPlay = playingRef.current;
    playingRef.current = false;
    if (shouldPlay) void video.play().catch(() => setStatus(''));
  };

  // Handles playback shortcuts without taking over focused controls.
  const keyboard = (event: React.KeyboardEvent) => {
    event.stopPropagation();
    if ((event.target as HTMLElement).closest('select,button,input,[role="slider"]')) return;
    const video = videoRef.current;
    if (!video) return;
    switch (event.key.toLowerCase()) {
      case ' ':
      case 'k':
        event.preventDefault();
        togglePlayback();
        break;
      case 'arrowleft':
        event.preventDefault();
        video.currentTime = Math.max(0, video.currentTime - 5);
        break;
      case 'arrowright':
        event.preventDefault();
        video.currentTime = Math.min(video.duration || 0, video.currentTime + 5);
        break;
      case 'm':
        video.muted = !video.muted;
        break;
      case 'f':
        event.preventDefault();
        if (document.fullscreenElement) void document.exitFullscreen().catch(() => {});
        else void containerRef.current?.requestFullscreen?.().catch(() => {});
        break;
    }
  };

  // Keeps control interactions from triggering gallery navigation.
  const onGestureStart = (event: React.MouseEvent | React.TouchEvent) => {
    const target = event.target as HTMLElement;
    // Keep control gestures here; let picture drags reach the gallery.
    if (
      target.closest('button, select, input, label, [data-video-controls]') ||
      document.fullscreenElement === containerRef.current
    ) {
      event.stopPropagation();
    }
  };

  if (!active) return <VideoPoster src={poster} title={title} aspectRatio={aspectRatio} />;

  return (
    <VideoJSPlayer>
      <Container
        ref={containerRef}
        className="gallery-video-stage"
        tabIndex={0}
        onMouseDown={onGestureStart}
        onTouchStart={onGestureStart}
        onClick={(e) => e.stopPropagation()}
        onDoubleClick={(e) => e.stopPropagation()}
        onKeyDown={keyboard}
      >
        <div
          className="gallery-video-player"
          data-video-player
          style={
            { '--video-aspect-ratio': Number.isFinite(ratio) && ratio > 0 ? ratio : 16 / 9 } as React.CSSProperties
          }
        >
          {poster && info && (
            <img src={poster} alt="" aria-hidden="true" className="gallery-video-poster-layer" draggable={false} />
          )}
          <Video
            ref={videoRef}
            className="gallery-video-media"
            style={{ opacity: hasFrame ? 1 : 0 }}
            poster={poster}
            controls={false}
            controlsList="noplaybackrate"
            disablePictureInPicture
            playsInline
            preload="none"
            aria-label={title || 'Video'}
            onPointerDown={(event) => {
              gestureRef.current = {
                x: event.clientX,
                y: event.clientY,
                dragged: !event.isPrimary || event.button !== 0,
              };
            }}
            onPointerMove={trackGesture}
            onPointerUp={trackGesture}
            onPointerCancel={() => {
              if (gestureRef.current) gestureRef.current.dragged = true;
            }}
            onClick={() => {
              if (!gestureRef.current?.dragged) togglePlayback();
            }}
            onLoadedMetadata={onMetadata}
            onLoadedData={showFrame}
            onResize={updateDimensions}
            onPlaying={() => {
              showFrame();
              hasPlayedRef.current = true;
              if (!switchRef.current) setStatus('');
              setError('');
            }}
            onCanPlay={() => {
              if (!switchRef.current && !videoRef.current?.seeking) setStatus('');
            }}
            onWaiting={() => {
              if (started && !switchRef.current) setStatus('Buffering…');
            }}
            onSeeking={() => {
              if (started && !switchRef.current) setStatus('Seeking…');
              // Load the requested frame while paused, then stop fetching.
              if (pausedLoadingRef.current) {
                pausedLoadingRef.current = false;
                hlsRef.current?.startLoad(videoRef.current?.currentTime || 0);
              }
            }}
            onSeeked={() => {
              if (!switchRef.current) setStatus('');
              savePosition();
              if (videoRef.current?.paused && hlsRef.current) {
                hlsRef.current.stopLoad();
                pausedLoadingRef.current = true;
              }
            }}
            onPause={() => {
              if (!switchRef.current && videoRef.current?.paused && hasPlayedRef.current) {
                savePosition();
                hlsRef.current?.stopLoad();
                pausedLoadingRef.current = true;
              }
            }}
            onPlay={() => {
              if (pausedLoadingRef.current) {
                pausedLoadingRef.current = false;
                hlsRef.current?.startLoad(videoRef.current?.currentTime || 0);
              }
            }}
            onTimeUpdate={() => {
              const video = videoRef.current;
              // Keep the intended position visible while switching sources resets native time.
              if (!video || video.readyState === HTMLMediaElement.HAVE_NOTHING) return;
              setPosition(video.currentTime);
              if (Date.now() - saveAtRef.current > 3000) {
                saveAtRef.current = Date.now();
                savePosition();
              }
            }}
            onEnded={savePosition}
            onError={() => {
              const video = videoRef.current;
              if (!video?.src || !started) return;
              if (video.dataset.direct === 'true') {
                video.dataset.direct = 'false';
                void load(quality, video.currentTime || resumeRef.current, true, true);
              } else {
                setStatus('');
                setError('Unable to play this video. Retry to continue.');
              }
            }}
          />
          {status && !error && (
            <div role="status" aria-label={status} className="gallery-video-loading">
              <span className="gallery-video-spinner" aria-hidden="true" />
            </div>
          )}
          {error && (
            <div role="alert" className="gallery-video-error">
              <span>{error}</span>
              <button
                className="rounded border border-white/40 px-3 py-1"
                onClick={() => begin(videoRef.current?.currentTime || resume)}
              >
                Retry
              </button>
            </div>
          )}
          <VideoControls
            started={started}
            ready={hasFrame}
            duration={info?.duration}
            position={started ? position : resume}
            preparing={!info || (!!status && !videoRef.current?.readyState)}
            onStart={() => begin()}
            onRestart={restart}
            thumbnailsURL={info ? `${base}/thumbnails.json?v=${info.version}` : undefined}
          >
            <select
              aria-label="Video quality"
              value={quality}
              onChange={(e) => selectQuality(e.target.value as VideoQuality)}
            >
              <option value="auto">Auto{currentQuality && quality === 'auto' ? ` · ${currentQuality}` : ''}</option>
              {(
                info?.profiles || [
                  { id: 'saver', label: 'Data saver' },
                  { id: 'small', label: 'Small' },
                  { id: 'high', label: 'High' },
                ]
              ).map((p) => (
                <option key={p.id} value={p.id}>
                  {p.label}
                  {info && 'longEdge' in p ? ` · ${resolutionLabel(info.width, info.height, Number(p.longEdge))}` : ''}
                </option>
              ))}
            </select>
          </VideoControls>
        </div>
      </Container>
    </VideoJSPlayer>
  );
};

export default VideoPlayer;
