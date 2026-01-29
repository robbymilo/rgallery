import React from 'react';
import Close from '../svg/close.svg?react';
import { Notification, User } from '../types';

interface NotificationsSidebarProps {
  user: User;
  isOpen: boolean;
  notifications: Notification[];
  onClose: () => void;
  onClearAll: () => void;
  onMarkAsRead: (id: number) => void;
}

const NotificationsSidebar: React.FC<NotificationsSidebarProps> = ({
  user,
  isOpen,
  notifications,
  onClose,
  onClearAll,
  onMarkAsRead,
}) => {
  return (
    <div
      id="notifications-offcanvas"
      className={`bg-charcoal-900/95 fixed inset-y-0 right-0 z-60 w-[320px] transform transition-transform duration-300 md:shadow-lg ${
        isOpen ? 'translate-x-0' : 'translate-x-full'
      }`}
      aria-hidden={!isOpen}
    >
      <div className="flex h-full flex-col">
        <div className="border-charcoal-700 flex items-center justify-between border-b px-4 py-3">
          <h3 className="text-sm font-medium text-white">Notifications</h3>
          <div className="flex items-center gap-2">
            <button
              onClick={onClose}
              className="text-charcoal-300 rounded-md p-1 hover:text-white"
              aria-label="Close notifications"
            >
              <Close className="h-4 w-4" />
            </button>
          </div>
        </div>
        <div className="grow overflow-auto px-2 py-3 text-xs">
          {notifications.length === 0 ? (
            <p className="text-charcoal-400 px-3">No notifications.</p>
          ) : (
            <div className="space-y-2 px-1">
              {notifications.map((n) => (
                <div
                  key={n.id}
                  className={`relative flex flex-col gap-1 rounded-md border px-2 py-2 ${
                    n.is_read
                      ? 'bg-charcoal-800 border-charcoal-700 text-charcoal-400'
                      : 'bg-charcoal-700 border-charcoal-600 text-charcoal-100'
                  }`}
                >
                  {user?.role === 'admin' && (
                    <button
                      aria-label={n.is_read ? 'Already read' : 'Mark notification as read'}
                      onClick={(e) => {
                        e.stopPropagation();
                        if (!n.is_read) onMarkAsRead(n.id);
                      }}
                      disabled={n.is_read}
                      className={`absolute top-1 right-1 inline-flex items-center justify-center rounded p-1 text-xs transition-colors focus:outline-none ${
                        n.is_read
                          ? 'text-charcoal-500 cursor-default'
                          : 'text-charcoal-200 cursor-pointer hover:text-white'
                      }`}
                      title={n.is_read ? 'Already read' : 'Mark as read'}
                    >
                      <Close className="h-3 w-3" />
                    </button>
                  )}

                  <div className="mr-4 text-xs">{n.message}</div>
                  <div className="text-[10px] opacity-70">{new Date(n.created_at).toISOString()}</div>
                </div>
              ))}
            </div>
          )}
          {user?.role === 'admin' && (
            <div className="mt-4 flex gap-2 px-1">
              <button
                onClick={onClearAll}
                className={`border-charcoal-700 flex-1 rounded-md border px-3 py-2 text-xs font-medium transition-colors ${
                  notifications.length === 0
                    ? 'bg-charcoal-800 text-charcoal-500 cursor-not-allowed'
                    : 'bg-red-700 text-white hover:bg-red-800'
                }`}
                disabled={notifications.length === 0}
              >
                Clear all
              </button>
            </div>
          )}
        </div>
        <div className="border-charcoal-700 border-t px-4 py-2">
          <button
            onClick={onClose}
            className="bg-primary-700 w-full rounded-md px-3 py-2 text-sm font-medium text-white"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};

export default NotificationsSidebar;
