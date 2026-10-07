import { useCallback, useLayoutEffect, useRef } from 'react';
import type { RefObject } from 'react';
import { flushSync } from 'react-dom';

const DURATION = 360;
const EASING = 'cubic-bezier(0.22, 0.61, 0.36, 1)';

export function useZoomTransition(
  mediaKey: number,
  imageRef: RefObject<HTMLImageElement | null>,
  viewerRef: RefObject<HTMLDivElement | null>
) {
  const active = useRef<{
    cancel: () => void;
    image: HTMLImageElement;
    backdrop: HTMLDivElement;
  } | null>(null);
  const cancel = useCallback(() => {
    active.current?.cancel();
  }, []);

  useLayoutEffect(() => cancel, [mediaKey, cancel]);
  useLayoutEffect(() => {
    window.addEventListener('resize', cancel);
    return () => window.removeEventListener('resize', cancel);
  }, [cancel]);

  const run = useCallback(
    (update: () => void) => {
      const image = imageRef.current;
      const viewer = viewerRef.current;
      // A reversal starts at the currently visible position, not the previous
      // animation's destination. Keep one decoded image throughout the motion.
      const from = (active.current?.image || image)?.getBoundingClientRect();
      const fromOpacity = active.current
        ? Number(getComputedStyle(active.current.backdrop).opacity)
        : viewer?.getAttribute('role') === 'dialog'
          ? 1
          : 0;
      active.current?.cancel();
      if (!image?.complete || !image.naturalWidth || !viewer || !from?.width || typeof image.animate !== 'function') {
        update();
        return;
      }
      const source = image.currentSrc || image.src;
      flushSync(update);
      const to = image.getBoundingClientRect();
      if (imageRef.current !== image || !to.width || !to.height) return;
      const toOpacity = viewer.getAttribute('role') === 'dialog' ? 1 : 0;

      // The page adopts its final layout immediately underneath this temporary
      // layer. Only transform and opacity animate; layout and srcset do not.
      const layer = document.createElement('div');
      layer.className = 'image-zoom-animation';
      layer.setAttribute('aria-hidden', 'true');
      const backdrop = document.createElement('div');
      backdrop.className = 'image-zoom-animation__backdrop';
      const movingImage = document.createElement('img');
      movingImage.src = source;
      movingImage.alt = '';
      movingImage.decoding = 'sync';
      movingImage.draggable = false;
      const width = Math.max(from.width, to.width);
      const height = (width * to.height) / to.width;
      movingImage.style.cssText = `position:absolute;top:0;left:0;width:${width}px;height:${height}px;max-width:none;max-height:none;transform-origin:0 0;will-change:transform;`;
      layer.append(backdrop, movingImage);
      document.body.append(layer);

      const previousOpacity = image.style.opacity;
      const previousBackground = viewer.style.backgroundColor;
      const previousTransition = viewer.style.transition;
      image.style.opacity = '0';
      viewer.style.transition = 'none';
      viewer.style.backgroundColor = 'transparent';
      viewer.classList.add('image-zoom-animating');
      const transform = (rect: DOMRect) =>
        `translate3d(${rect.left}px, ${rect.top}px, 0) scale(${rect.width / width}, ${rect.height / height})`;
      const timing: KeyframeAnimationOptions = { duration: DURATION, easing: EASING, fill: 'both' };
      const movement = movingImage.animate([{ transform: transform(from) }, { transform: transform(to) }], timing);
      const fade = backdrop.animate([{ opacity: fromOpacity }, { opacity: toOpacity }], timing);
      const cleanup = () => {
        if (active.current !== pending) return;
        movement.cancel();
        fade.cancel();
        image.style.opacity = previousOpacity;
        viewer.style.backgroundColor = previousBackground;
        viewer.classList.remove('image-zoom-animating');
        // Restore the final background without starting a second CSS fade.
        void getComputedStyle(viewer).backgroundColor;
        viewer.style.transition = previousTransition;
        layer.remove();
        active.current = null;
      };
      const pending = { cancel: cleanup, image: movingImage, backdrop };
      active.current = pending;
      void Promise.all([movement.finished, fade.finished]).then(cleanup, cleanup);
    },
    [imageRef, viewerRef]
  );

  return { runZoomTransition: run, cancelZoomTransition: cancel };
}
