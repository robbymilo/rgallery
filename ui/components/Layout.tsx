import React, { useState, useRef, useEffect, useCallback } from 'react';
import { Link, useNavigate, useLocation, useMatches } from 'react-router-dom';
import MemoriesWidget from '../components/MemoriesWidget';
import { useAuth } from '../context/AuthContext';
import { useFullscreen } from '../context/FullscreenContext';
import { useWebSocketNotifications } from '../hooks/useWebSocketNotifications';
import { getMemories } from '../services/memories';
import { useWebSocket } from '../context/WebSocketContext';
import { useToast } from '../context/ToastContext';
import { Memory, Notification } from '../types';
import NotificationsSidebar from './NotificationsSidebar';
import version from '../version.json';
import Logo from '../svg/logo.svg?react';
import Refresh from '../svg/refresh.svg?react';
import Timeline from '../svg/timeline.svg?react';
import Folders from '../svg/folders.svg?react';
import Tags from '../svg/tags.svg?react';
import Gear from '../svg/gear.svg?react';
import Map from '../svg/map.svg?react';
import Bookmark from '../svg/bookmark.svg?react';
import Bell from '../svg/bell.svg?react';
import Shield from '../svg/shield.svg?react';
import Moon from '../svg/moon.svg?react';
import Sun from '../svg/sun.svg?react';
import LogOut from '../svg/logout.svg?react';
import ChevronDown from '../svg/chevron-down.svg?react';
import X from '../svg/close.svg?react';
import Menu from '../svg/menu.svg?react';

const Layout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { user, logout } = useAuth();
  const { isFullscreen } = useFullscreen();
  const { addToast, toasts, removeToast, clearToasts } = useToast();
  useWebSocketNotifications();
  const navigate = useNavigate();
  const location = useLocation();
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const [isDarkMode, setIsDarkMode] = useState(true);
  const [memories, setMemories] = useState<Memory[]>([]);
  const [isNotificationsOpen, setIsNotificationsOpen] = useState(false);
  const [notifications, setNotifications] = useState<Notification[]>([]);

  // WebSocket scan state
  const { isScanInProgress, lastMessage } = useWebSocket();

  // update title
  const matches = useMatches();
  useEffect(() => {
    type MatchWithHandle = { handle?: { title?: string } };
    const title = (matches as MatchWithHandle[])
      .slice()
      .reverse()
      .find((m) => m.handle?.title)?.handle?.title;

    if (title) document.title = title + ` | rgallery`;
  }, [matches]);

  // Sync dark mode
  useEffect(() => {
    const isDark = document.documentElement.classList.contains('dark');
    setIsDarkMode(isDark);

    // Close dropdown when clicking outside
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsDropdownOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // get notifications
  useEffect(() => {
    fetch('/api/notifications')
      .then((res) => {
        if (res.ok) return res.json();
        return [];
      })
      .then((data) => setNotifications(data || []))
      .catch((err) => console.error('Failed to fetch notifications', err));
  }, [user]);

  // handle incoming notification messages
  useEffect(() => {
    if (lastMessage && lastMessage.id) {
      // It's a notification object
      const newNotif = lastMessage as unknown as Notification;
      setNotifications((prev) => {
        if (prev.some((n) => n.id === newNotif.id)) return prev;
        return [newNotif, ...prev];
      });
    }
  }, [lastMessage, user]);

  // Notification handlers for sidebar
  const handleMarkAsRead = useCallback((id: number) => {
    fetch(`/api/notifications/${id}`, { method: 'PATCH' }).then((res) => {
      if (res.ok) {
        setNotifications((prev) => prev.map((n) => (n.id === id ? { ...n, is_read: true } : n)));
      }
    });
  }, []);

  // Clear all notifications
  const handleClearAll = useCallback(async () => {
    await fetch('/api/notifications/clear', { method: 'POST' });
    setNotifications([]);
  }, []);

  const handleCloseNotifications = useCallback(() => setIsNotificationsOpen(false), []);

  const unreadCount = notifications.filter((n) => !n.is_read).length;

  const dropdownRef = useRef<HTMLDivElement>(null);
  const notificationsSidebarRef = useRef<HTMLDivElement>(null);
  const notificationBellRef = useRef<HTMLButtonElement>(null);
  const prevToastCount = useRef<number>(toasts.length);
  const [animateBell, setAnimateBell] = useState(false);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  // Trigger scan
  const handleScan = useCallback(async () => {
    if (isScanInProgress) {
      addToast('Scan already in progress', 'error');
      return;
    }
    try {
      const res = await fetch('/api/scan?type=scan', { credentials: 'include' });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        if (data?.error && data.error.includes('already in progress')) {
          addToast('Scan already in progress', 'error');
        } else {
          addToast('Failed to start scan', 'error');
        }
      }
    } catch (e) {
      addToast('Failed to start scan', 'error');
    }
  }, [addToast, isScanInProgress]);

  // Cancel scan
  const handleCancelScan = useCallback(async () => {
    try {
      const res = await fetch('/api/scan/cancel', { method: 'POST', credentials: 'include' });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        addToast(data?.msg || 'Failed to cancel scan', 'error');
      }
    } catch (e) {
      addToast('Failed to cancel scan', 'error');
    }
  }, [addToast]);

  // Close notifications offcanvas when clicking outside
  useEffect(() => {
    const handleClickOutsideNotifications = (event: MouseEvent) => {
      const off = notificationsSidebarRef.current;
      const bell = notificationBellRef.current;
      if (
        isNotificationsOpen &&
        off &&
        !off.contains(event.target as Node) &&
        bell &&
        !bell.contains(event.target as Node)
      ) {
        setIsNotificationsOpen(false);
      }
    };

    if (isNotificationsOpen) {
      document.addEventListener('mousedown', handleClickOutsideNotifications);
    }
    return () => document.removeEventListener('mousedown', handleClickOutsideNotifications);
  }, [isNotificationsOpen]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isNotificationsOpen) {
        setIsNotificationsOpen(false);
      }
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [isNotificationsOpen]);

  useEffect(() => {
    const current = toasts.length;
    if (current > prevToastCount.current) {
      setAnimateBell(true);
      const t = setTimeout(() => setAnimateBell(false), 900);
      prevToastCount.current = current;
      return () => clearTimeout(t);
    }
    prevToastCount.current = current;
  }, [toasts.length]);

  const toggleTheme = () => {
    const newMode = !isDarkMode;
    setIsDarkMode(newMode);

    if (newMode) {
      document.documentElement.classList.add('dark');
      document.documentElement.style.colorScheme = 'dark';
      localStorage.setItem('theme', 'dark');
    } else {
      document.documentElement.classList.remove('dark');
      document.documentElement.style.colorScheme = 'light';
      localStorage.setItem('theme', 'light');
    }
  };

  // Close mobile menu on route change
  useEffect(() => {
    setIsMobileMenuOpen(false);
  }, [location.pathname]);

  // Check for memories
  useEffect(() => {
    if (!isTimelineRoute) return;

    let mounted = true;
    const checkMemories = async () => {
      try {
        if (user) {
          const data = await getMemories();
          if (mounted && data.length > 0) {
            setMemories(data);
          } else if (mounted) {
            setMemories([]);
          }
        } else if (mounted) {
          setMemories([]);
        }
      } catch (e) {
        if (mounted) setMemories([]);
      }
    };

    checkMemories();
    return () => {
      mounted = false;
    };
  }, [user]);

  const isActive = (path: string) => {
    return location.pathname === path;
  };

  // timeline route ("/") should be overflow-hidden, other pages overflow-auto
  const isTimelineRoute = location.pathname === '/';
  const contentOverflowClass = isTimelineRoute ? 'overflow-hidden' : 'overflow-auto';

  return (
    <div className="bg-charcoal-900 relative flex min-h-screen flex-col transition-colors duration-300">
      {/* Navbar */}
      <header
        className={`border-charcoal-700 bg-charcoal-900/90 sticky top-0 z-50 border-b backdrop-blur-md transition-all duration-200 ${
          isFullscreen ? 'opacity-0' : 'opacity-100'
        }`}
        style={{
          pointerEvents: isFullscreen ? 'none' : 'auto',
          height: isFullscreen ? 0 : 40,
          transition: 'all 0.2s',
          overflow: 'visible',
        }}
      >
        <div className="mx-auto h-full w-[90vw] md:w-[80vw]">
          <div className="flex h-full items-center justify-between">
            <div className="flex h-[25px] w-[100px] shrink-0 items-center justify-start">
              <Link to="/" className="group flex h-full w-full items-center text-gray-100">
                <Logo />
              </Link>
            </div>

            <div className="ml-auto flex items-center">
              {/* desktop */}
              <nav className="hidden space-x-1 md:flex">
                <Link to="/" className={`nav-link ${isActive('/') ? 'nav-link-active' : ''}`}>
                  <Timeline className="h-3.5 w-3.5" />
                  Timeline
                </Link>
                <Link to="/memories" className={`nav-link ${isActive('/memories') ? 'nav-link-active' : ''}`}>
                  <div className="relative">
                    <Bookmark className="h-3.5 w-3.5" />
                  </div>
                  Memories
                </Link>
                <Link to="/folders" className={`nav-link ${isActive('/folders') ? 'nav-link-active' : ''}`}>
                  <Folders className="h-3.5 w-3.5" />
                  Folders
                </Link>

                <Link to="/tags" className={`nav-link ${isActive('/tags') ? 'nav-link-active' : ''}`}>
                  <Tags className="h-3.5 w-3.5" />
                  Tags
                </Link>
                <Link to="/gear" className={`nav-link ${isActive('/gear') ? 'nav-link-active' : ''}`}>
                  <Gear className="h-3.5 w-3.5" />
                  Gear
                </Link>
                <Link to="/map" className={`nav-link ${isActive('/map') ? 'nav-link-active' : ''}`}>
                  <Map className="h-3.5 w-3.5" />
                  Map
                </Link>
              </nav>
              <button
                onClick={handleScan}
                className="text-charcoal-300 flex items-center text-sm transition-colors hover:text-white md:hidden"
                role="status"
                title={isScanInProgress ? 'Scanning in progress' : 'Start scan'}
              >
                <Refresh className={`mr-2 h-3.5 w-3.5 ${isScanInProgress ? 'animate-spin' : ''}`} />
              </button>
              <button
                ref={notificationBellRef}
                id="notification-bell-button"
                onClick={() => setIsNotificationsOpen(!isNotificationsOpen)}
                aria-expanded={isNotificationsOpen}
                aria-controls="notifications-offcanvas"
                className="text-charcoal-300 flex items-center gap-1.5 rounded-md px-2 py-1 text-sm font-medium transition-colors hover:text-white"
              >
                <span className="text-charcoal-300 relative inline-flex hover:text-white">
                  <Bell
                    className={`h-3.5 w-3.5 transition-all duration-200 ${unreadCount > 0 ? 'text-primary-400' : ''}`}
                    aria-hidden="true"
                  />
                </span>
              </button>
              {/* desktop dropdown */}
              <div className="relative ml-2 hidden md:block" ref={dropdownRef}>
                <button
                  onClick={() => setIsDropdownOpen(!isDropdownOpen)}
                  className="text-charcoal-300 flex cursor-pointer items-center space-x-1 text-sm transition-colors hover:text-white focus:outline-none"
                >
                  <div className="relative">
                    {isScanInProgress && (
                      <div
                        className="border-t-primary-700 pointer-events-none absolute inset-0 z-20 animate-spin rounded-full border border-white"
                        aria-hidden="true"
                      />
                    )}
                    <div
                      className={`bg-primary-700 relative z-10 flex h-7 w-7 items-center justify-center rounded-full border border-white text-xs font-bold text-white shadow-sm`}
                    >
                      {user?.username.charAt(0).toUpperCase()}
                    </div>
                  </div>
                  <ChevronDown
                    className={`h-3 w-3 transition-transform duration-200 ${isDropdownOpen ? 'rotate-180' : ''}`}
                  />
                </button>
                {isDropdownOpen && (
                  <div className="ring-opacity-5 animate-fade-in border-charcoal-700 bg-charcoal-800 absolute right-0 z-50 mt-2 w-56 rounded-md border py-1 shadow-lg ring-1 ring-black focus:outline-none">
                    <div className="border-charcoal-700 border-b px-4 py-2">
                      <p className="truncate text-sm font-medium text-white">{user?.username}</p>
                    </div>
                    <Link
                      to="/admin"
                      onClick={() => setIsDropdownOpen(false)}
                      className="hover:bg-charcoal-700 flex items-center px-4 py-2 text-sm font-medium text-white"
                    >
                      <Shield className="text-primary-400 mr-3 h-4 w-4" />
                      Admin
                    </Link>
                    {user?.role === 'admin' &&
                      (!isScanInProgress ? (
                        <button
                          onClick={handleScan}
                          className="hover:bg-charcoal-700 flex w-full cursor-pointer items-center px-4 py-2 text-sm font-medium text-white"
                          title="Trigger scan"
                        >
                          <Refresh className="text-primary-400 mr-3 h-4 w-4" />
                          Scan
                        </button>
                      ) : (
                        <button
                          onClick={handleCancelScan}
                          className="hover:bg-charcoal-700 flex w-full cursor-pointer items-center px-4 py-2 text-sm text-white"
                          title="Cancel scan"
                        >
                          <Refresh className="text-primary-400 mr-3 h-4 w-4 animate-spin" />
                          Cancel scan
                        </button>
                      ))}
                    {/* dark mode */}
                    <button
                      onClick={toggleTheme}
                      className="hover:bg-charcoal-700 flex w-full cursor-pointer px-4 py-2 text-sm text-white"
                    >
                      {isDarkMode ? (
                        <>
                          <Moon className="text-primary-400 mr-3 h-4 w-4" />
                          <span className="text-white">Dark mode</span>
                        </>
                      ) : (
                        <>
                          <Sun className="text-primary-400 mr-3 h-4 w-4" />
                          <span className="text-white">Light mode</span>
                        </>
                      )}
                      <div className={`relative ml-2 h-5 w-10 rounded-full bg-gray-300`}>
                        <div
                          className={`absolute top-1 h-3 w-3 rounded-full bg-white shadow-sm transition-all duration-300 ${
                            isDarkMode ? 'left-6' : 'left-1'
                          }`}
                        />
                      </div>
                    </button>
                    <button
                      onClick={handleLogout}
                      className="hover:bg-charcoal-700 flex w-full cursor-pointer items-center px-4 py-2 text-sm text-red-400"
                    >
                      <LogOut className="mr-3 h-4 w-4" />
                      Sign out
                    </button>
                  </div>
                )}
              </div>
              {/* mobile menu */}
              <div className="flex items-center md:hidden">
                <button
                  onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
                  className="text-charcoal-300 hover:bg-charcoal-800 cursor-pointer rounded-md p-1"
                >
                  {isMobileMenuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
                </button>
              </div>
            </div>
          </div>
        </div>
        {/* mobile dropdown */}
        {isMobileMenuOpen && (
          <div className="animate-fade-in border-charcoal-800 bg-charcoal-900 absolute top-10 left-0 z-50 h-[calc(100vh-40px)] w-full overflow-y-auto border-t shadow-xl md:hidden">
            <div className="mx-auto w-[90vw] space-y-1 py-3">
              <Link to="/" className={`mobile-nav-link ${isActive('/') ? 'mobile-nav-link-active' : ''}`}>
                <Timeline className="h-5 w-5" />
                Timeline
              </Link>
              <Link
                to="/memories"
                className={`mobile-nav-link ${isActive('/memories') ? 'mobile-nav-link-primary-active' : ''}`}
              >
                <Bookmark className="h-5 w-5" />
                Memories
              </Link>
              <Link
                to="/folders"
                className={`mobile-nav-link ${isActive('/folders') ? 'mobile-nav-link-primary-active' : ''}`}
              >
                <Folders className="h-5 w-5" />
                Folders
              </Link>
              <Link to="/tags" className={`mobile-nav-link ${isActive('/tags') ? 'mobile-nav-link-active' : ''}`}>
                <Tags className="h-5 w-5" />
                Tags
              </Link>
              <Link to="/gear" className={`mobile-nav-link ${isActive('/gear') ? 'mobile-nav-link-active' : ''}`}>
                <Gear className="h-5 w-5" />
                Gear
              </Link>
              <Link to="/map" className={`mobile-nav-link ${isActive('/map') ? 'mobile-nav-link-active' : ''}`}>
                <Map className="h-5 w-5" />
                Map
              </Link>
            </div>
            <div className="border-charcoal-800 mx-auto w-[90vw] border-t pt-4 pb-4">
              <div className="mb-3 flex items-center px-3">
                <div className="text-base leading-none font-medium text-white">{user?.username}</div>
              </div>
              <div className="space-y-1">
                {user?.role === 'admin' && (
                  <Link
                    to="/admin"
                    className="text-charcoal-300 hover:bg-charcoal-800 block rounded-md px-3 py-2 text-base font-medium hover:text-white"
                  >
                    Admin
                  </Link>
                )}
                {/* mobile dark mode */}
                <button onClick={toggleTheme} className="h text-charcoal-300 flex w-full px-3 py-2 text-sm">
                  {isDarkMode ? (
                    <>
                      <span className="text-charcoal-200">Dark mode</span>
                    </>
                  ) : (
                    <>
                      <span className="text-gray-200">Light mode</span>
                    </>
                  )}
                  <div
                    className={`relative ml-2 h-5 w-10 rounded-full transition-colors ${
                      isDarkMode ? 'bg-primary-600' : 'bg-gray-300'
                    }`}
                  >
                    <div
                      className={`absolute top-1 h-3 w-3 rounded-full bg-white shadow-sm transition-all duration-300 ${
                        isDarkMode ? 'left-6' : 'left-1'
                      }`}
                    />
                  </div>
                </button>
                <button
                  onClick={handleLogout}
                  className="block w-full rounded-md px-3 py-2 text-left text-base font-medium text-red-400 hover:bg-red-900/20"
                >
                  Sign out
                </button>
              </div>
            </div>
          </div>
        )}
      </header>

      {/* notifications offcanvas */}
      <div ref={notificationsSidebarRef}>
        <NotificationsSidebar
          user={user}
          isOpen={isNotificationsOpen}
          notifications={notifications}
          onClose={handleCloseNotifications}
          onClearAll={handleClearAll}
          onMarkAsRead={handleMarkAsRead}
        />
      </div>

      {/* Memories spine */}
      {memories.length > 0 && isTimelineRoute && <MemoriesWidget memories={memories} />}

      {/* Main content */}
      <main
        className={`dark:bg-charcoal-900 grow ${contentOverflowClass} bg-gray-50`}
        style={{
          height: 'calc(100vh - 80px)',
        }}
      >
        {children}
      </main>

      {/* Footer */}
      <footer
        className={`dark:border-charcoal-700 dark:bg-charcoal-800 border-t border-gray-200 bg-white px-2 transition-all duration-200 ${
          isFullscreen ? 'opacity-0' : 'opacity-100'
        }`}
        style={{
          pointerEvents: isFullscreen ? 'none' : 'auto',
          height: isFullscreen ? 0 : 32,
          transition: 'all 0.2s',
          overflow: 'hidden',
        }}
      >
        <div className="dark:text-charcoal-300 mx-auto flex h-full w-full max-w-full flex-row flex-nowrap items-center justify-center gap-2 overflow-x-auto text-center text-xs text-gray-500">
          <a
            href="https://rgallery.app/"
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary-400 hover:underline"
          >
            rgallery
          </a>
          <span>|</span>
          <span>&copy; {new Date().getFullYear()} Robby Milo</span>
          <span>|</span>
          <a
            href={
              version.tag
                ? `https://github.com/robbymilo/rgallery/releases/tag/${version.tag}`
                : `https://github.com/robbymilo/rgallery/commit/${version.sha}`
            }
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary-400 ml-0 whitespace-nowrap hover:underline sm:ml-1"
          >
            {version.tag ? `v${version.tag}` : `${version.sha.slice(0, 7)}`}
          </a>
        </div>
      </footer>
    </div>
  );
};

export default Layout;
