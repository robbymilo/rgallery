import { useLayoutEffect } from 'react';
import type { RefObject } from 'react';

export function useViewerOverlay(isOpen: boolean, viewerRef: RefObject<HTMLDivElement | null>) {
  useLayoutEffect(() => {
    const viewer = viewerRef.current;
    if (!isOpen || !viewer) return;
    const previousFocus = document.activeElement as HTMLElement | null;
    const backgrounds: { element: HTMLElement; inert: boolean }[] = [];
    const scrollContainers: { element: HTMLElement; overflow: string; top: number; left: number }[] = [];

    // Keep the existing viewer node (and its pointer captures) in place while
    // making the rest of the page inert and preserving its scroll position.
    for (let node: HTMLElement = viewer; node.parentElement; node = node.parentElement) {
      const parent = node.parentElement;
      for (const sibling of parent.children) {
        if (sibling !== node && sibling instanceof HTMLElement) {
          backgrounds.push({ element: sibling, inert: sibling.inert });
        }
      }
      if (
        parent.scrollHeight > parent.clientHeight ||
        parent === document.body ||
        parent === document.documentElement
      ) {
        scrollContainers.push({
          element: parent,
          overflow: parent.style.overflow,
          top: parent.scrollTop,
          left: parent.scrollLeft,
        });
      }
    }
    backgrounds.forEach(({ element }) => {
      element.inert = true;
    });
    scrollContainers.forEach(({ element }) => {
      element.style.overflow = 'hidden';
    });
    viewer.focus({ preventScroll: true });

    const trapFocus = (event: KeyboardEvent) => {
      if (event.key !== 'Tab') return;
      const controls = [
        ...viewer.querySelectorAll<HTMLElement>('a[href], button:not(:disabled), video[controls], [tabindex="0"]'),
      ].filter((element) => !element.closest('[inert]') && element.getClientRects().length > 0);
      const first = controls[0];
      const last = controls[controls.length - 1];
      if (!first) {
        event.preventDefault();
        return;
      }
      if (event.shiftKey && (document.activeElement === first || document.activeElement === viewer)) {
        event.preventDefault();
        last.focus({ preventScroll: true });
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus({ preventScroll: true });
      }
    };
    viewer.addEventListener('keydown', trapFocus);
    return () => {
      viewer.removeEventListener('keydown', trapFocus);
      backgrounds.forEach(({ element, inert }) => {
        element.inert = inert;
      });
      scrollContainers.forEach(({ element, overflow, top, left }) => {
        element.style.overflow = overflow;
        element.scrollTop = top;
        element.scrollLeft = left;
      });
      if (previousFocus?.isConnected && previousFocus !== document.body) previousFocus.focus({ preventScroll: true });
      else if (viewer.isConnected)
        viewer.querySelector<HTMLElement>('[data-zoom-control]')?.focus({ preventScroll: true });
    };
  }, [isOpen, viewerRef]);
}
