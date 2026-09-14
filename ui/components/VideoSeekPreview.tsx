import React, { useEffect, useState } from 'react';
import { Slider, TimeSlider } from '@videojs/react';
import { videoJSON } from '../lib/video';

type Thumbnails = NonNullable<React.ComponentProps<typeof Slider.Thumbnail.Root>['thumbnails']>;

// Shows the seek thumbnail and timestamp at the pointer.
export default function VideoSeekPreview({ url, visible }: { url?: string; visible: boolean }) {
  const [loaded, setLoaded] = useState<{ url: string; thumbnails: Thumbnails } | null>(null);
  const thumbnails = loaded && loaded.url === url ? loaded.thumbnails : undefined;

  useEffect(() => {
    if (!url || !visible || thumbnails) return;
    const controller = new AbortController();
    videoJSON<Thumbnails>(url, controller.signal)
      .then((thumbnails) => {
        if (!controller.signal.aborted) setLoaded({ url, thumbnails });
      })
      .catch(() => {
        // Keep the timestamp on failure and retry thumbnails on the next hover.
      });
    return () => controller.abort();
  }, [url, visible, thumbnails]);

  return (
    <>
      {visible && thumbnails?.length > 0 && (
        <Slider.Thumbnail.Root className="gallery-video-thumbnail" thumbnails={thumbnails}>
          <Slider.Thumbnail.Image draggable={false} fetchPriority="low" />
        </Slider.Thumbnail.Root>
      )}
      <TimeSlider.Value type="pointer" />
    </>
  );
}
