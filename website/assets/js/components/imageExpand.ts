export default function imageExpand(): Record<string, any> {
  return {
    expanded: false,
    toggleImage(this: any) {
      const container = this.$root as HTMLElement | null;
      if (!container) return;
      const image = container.querySelector('img') as HTMLImageElement | null;
      const button = container.querySelector('.expand-btn') as HTMLElement | null;

      if (!image || !button) return;

      if (!this.expanded) {
        container.style.height = image.scrollHeight + 'px';
        button.classList.add('rotated');
      } else {
        container.style.height = '400px';
        button.classList.remove('rotated');
      }

      this.expanded = !this.expanded;
    },
  };
}
