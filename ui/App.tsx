import React, { useEffect } from 'react';
import { createBrowserRouter, RouterProvider, Navigate, Outlet, useLocation } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { FullscreenProvider } from './context/FullscreenContext';
import { WebSocketProvider } from './context/WebSocketContext';
import { ToastProvider } from './context/ToastContext';
import Layout from './components/Layout';
import Timeline from './pages/Timeline';
import MediaDetail from './pages/MediaDetail';
import Admin from './pages/Admin';
import FolderBrowser from './pages/FolderBrowser';
import Login from './pages/Login';
import Memories from './pages/Memories';
import Gear from './pages/Gear';
import Tags from './pages/Tags';
import Map from './pages/Map';

const ProtectedRoute = () => {
  const { isAuthenticated, isLoading } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return (
      <div className="bg-charcoal-900 text-charcoal-400 flex min-h-screen items-center justify-center">
        <svg
          className="text-primary-500 h-8 w-8 animate-spin"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          ></path>
        </svg>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to={`/login?redirect=${encodeURIComponent(location.pathname)}`} replace />;
  }

  return <Outlet />;
};

// Route Configuration
const router = createBrowserRouter([
  {
    path: '/login',
    element: <Login />,
    handle: {
      title: 'Login',
    },
  },
  {
    path: '/',
    element: (
      <Layout>
        <ProtectedRoute />
      </Layout>
    ),
    children: [
      {
        index: true,
        element: <Timeline />,
        handle: {
          title: 'Timeline',
        },
      },
      {
        path: 'media/:mediaID',
        element: <MediaDetail />,
      },
      {
        path: 'folders',
        element: <FolderBrowser />,
        handle: {
          title: 'Folders',
        },
      },
      {
        path: 'memories',
        element: <Memories />,
        handle: {
          title: 'Memories',
        },
      },
      {
        path: 'tags',
        element: <Tags />,
        handle: {
          title: 'Tags',
        },
      },
      {
        path: 'gear',
        element: <Gear />,
        handle: {
          title: 'Gear stats',
        },
      },
      {
        path: 'admin',
        element: <Admin />,
        handle: {
          title: 'Admin',
        },
      },
      {
        path: 'map',
        element: <Map />,
        handle: {
          title: 'Map',
        },
      },
    ],
  },
  {
    path: '*',
    element: <Navigate to="/" replace />,
  },
]);

const InnerApp = () => {
  const { isLoading } = useAuth();

  if (isLoading) return null;

  return <RouterProvider router={router} />;
};

const App: React.FC = () => {
  return (
    <AuthProvider>
      <ToastProvider>
        <WebSocketProvider>
          <FullscreenProvider>
            <InnerApp />
          </FullscreenProvider>
        </WebSocketProvider>
      </ToastProvider>
    </AuthProvider>
  );
};

export default App;
