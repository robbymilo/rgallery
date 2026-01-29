import path from 'path';
import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import svgr from 'vite-plugin-svgr';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, '.', '');
  return {
    server: {
      host: '0.0.0.0',
      proxy: {
        '/api/ws': {
          target: 'ws://localhost:3000',
          ws: true,
          changeOrigin: true,
        },
        '/api': {
          target: 'http://localhost:3000/',
        },
        '/dist': {
          target: 'http://localhost:3000/',
        },
        '/fonts': {
          target: 'http://localhost:3000/',
        },
        '/favicon.ico': {
          target: 'http://localhost:3000/',
        },
        '/static': {
          target: 'http://localhost:3000/',
        },
        '/tiles': {
          target: 'http://localhost:3000/',
        },
      },
    },
    plugins: [react(), tailwindcss(), svgr()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, '.'),
      },
    },
    build: {
      outDir: '../pkg/dist/spa',
      rollupOptions: {
        input: {
          index: 'index.html',
          sw: 'src/service-worker.js',
        },
      },
    },
  };
});
