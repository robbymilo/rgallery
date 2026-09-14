import React from 'react';
import { Controls, FullscreenButton, MuteButton, PlayButton, Time, TimeSlider, VolumeSlider } from '@videojs/react';
import {
  FullscreenEnterIcon,
  FullscreenExitIcon,
  PauseIcon,
  PlayIcon,
  RestartIcon,
  VolumeHighIcon,
  VolumeOffIcon,
} from '@videojs/react/icons/minimal';
import './VideoPlayer.css';
import { displayTime } from '../lib/video';
import VideoSeekPreview from './VideoSeekPreview';

interface VideoControlsProps {
  started: boolean;
  ready: boolean;
  duration?: number;
  position: number;
  preparing: boolean;
  onStart: () => void;
  onRestart: () => void;
  thumbnailsURL?: string;
  children: React.ReactNode;
}

// Build player controls with Video.js while keeping gallery gestures and quality selection.
export default function VideoControls({
  started,
  ready,
  duration,
  position,
  preparing,
  onStart,
  onRestart,
  thumbnailsURL,
  children,
}: VideoControlsProps) {
  // Show the saved position until the video timeline is ready.
  const pending = !started || !ready;
  const progress = duration > 0 ? Math.max(0, Math.min(100, (position / duration) * 100)) : 0;
  return (
    <Controls.Root visibility={started ? 'auto' : 'always'}>
      <Controls.Content className="gallery-video-controls" data-video-controls>
        <TimeSlider.Root
          className="gallery-video-slider gallery-video-timeline"
          disabled={pending}
          style={pending ? ({ '--media-slider-fill': `${progress}%` } as React.CSSProperties) : undefined}
        >
          <TimeSlider.Track className="gallery-video-track">
            <TimeSlider.Buffer className="gallery-video-buffer" />
            <TimeSlider.Fill className="gallery-video-fill" />
          </TimeSlider.Track>
          <TimeSlider.Thumb
            className="gallery-video-thumb"
            {...(pending && {
              'aria-valuemin': 0,
              'aria-valuemax': duration || 0,
              'aria-valuenow': position,
              'aria-valuetext': `${displayTime(position)} of ${displayTime(duration || 0)}`,
            })}
          />
          <TimeSlider.Preview
            className="gallery-video-preview"
            overflow="clamp"
            render={(props, state) => (
              <div {...props}>
                <VideoSeekPreview url={thumbnailsURL} visible={state.pointing || state.dragging} />
              </div>
            )}
          />
        </TimeSlider.Root>
        <div className="gallery-video-buttons">
          {started ? (
            <PlayButton
              className="gallery-video-button"
              disabled={preparing}
              render={(props, state) => <button {...props}>{state.paused ? <PlayIcon /> : <PauseIcon />}</button>}
            />
          ) : (
            <button
              className="gallery-video-button"
              aria-label={position > 0 ? `Resume at ${displayTime(position)}` : 'Play video'}
              title={position > 0 ? `Resume at ${displayTime(position)}` : 'Play video'}
              disabled={preparing}
              onClick={onStart}
            >
              <PlayIcon />
            </button>
          )}
          <button
            className="gallery-video-button"
            aria-label="Restart video"
            title="Restart video"
            disabled={position <= 0 || preparing}
            onClick={onRestart}
          >
            <RestartIcon />
          </button>
          <MuteButton
            className="gallery-video-button"
            render={(props, state) => (
              <button {...props}>
                {state.muted || state.volumeLevel === 'off' ? <VolumeOffIcon /> : <VolumeHighIcon />}
              </button>
            )}
          />
          <VolumeSlider.Root className="gallery-video-slider gallery-video-volume" thumbAlignment="edge">
            <VolumeSlider.Track className="gallery-video-track">
              <VolumeSlider.Fill className="gallery-video-fill" />
            </VolumeSlider.Track>
            <VolumeSlider.Thumb className="gallery-video-thumb" />
          </VolumeSlider.Root>
          {!pending ? (
            <Time.Group className="gallery-video-time">
              <Time.Value type="current" />
              <Time.Separator />
              <Time.Value type="duration" />
            </Time.Group>
          ) : (
            <span className="gallery-video-time">
              <span>{displayTime(position)}</span>
              <span aria-hidden="true">/</span>
              <span data-type="duration">{displayTime(duration || 0)}</span>
            </span>
          )}
          <div className="gallery-video-quality">{children}</div>
          <FullscreenButton
            className="gallery-video-button"
            render={(props, state) => (
              <button {...props}>{state.fullscreen ? <FullscreenExitIcon /> : <FullscreenEnterIcon />}</button>
            )}
          />
        </div>
      </Controls.Content>
    </Controls.Root>
  );
}
