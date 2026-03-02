export default function menu(): Record<string, any> {
  return {
    toggle(e: Event) {
      const target = e.target as HTMLElement;
      const parent = target.parentElement?.parentElement as HTMLElement | null;
      const sub = parent?.querySelector('.menu--sub') as HTMLElement | null;
      if (sub) sub.classList.toggle('hidden');
      if (parent) parent.classList.toggle('menu--open');
    },
  };
}
