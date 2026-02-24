import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import './src/style.css';
import Hls from 'hls.js';

declare global {
  interface Window {
    Hls: typeof Hls;
  }
}

window.Hls = Hls;

const rootElement = document.getElementById('root');
if (!rootElement) {
  throw new Error('Could not find root element to mount to');
}

const root = ReactDOM.createRoot(rootElement);
root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);

// Register service worker for PWA support
try {
  if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
      navigator.serviceWorker.register('/service-worker.js').catch((err) => {
        console.warn('Service worker registration failed:', err);
      });
    });
  }
} catch (err) {
  console.warn('Service worker registration not available:', err);
}
