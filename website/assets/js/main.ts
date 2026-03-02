import Alpine from 'alpinejs';
import lazySizes from 'lazysizes';
import 'lazysizes/plugins/parent-fit/ls.parent-fit';
import 'lazysizes/plugins/unveilhooks/ls.unveilhooks';

import menu from './components/menu';
import imageExpand from './components/imageExpand';

declare global {
  interface Window { lazySizes: any; Alpine?: any }
}

window.lazySizes = window.lazySizes || {};
window.Alpine = Alpine;

Alpine.data('menu', menu);
Alpine.data('imageExpand', imageExpand);
Alpine.start();
