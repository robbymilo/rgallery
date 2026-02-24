import { useEffect } from 'react';
import { useWebSocket } from '../context/WebSocketContext';
import { useToast } from '../context/ToastContext';

/**
 * Hook to handle WebSocket notifications and display them as toasts
 */
export const useWebSocketNotifications = () => {
  const { lastMessage } = useWebSocket();
  const { addToast } = useToast();

  useEffect(() => {
    if (!lastMessage) return;

    // Don't show toast for initial connection message or "none" messages
    if (lastMessage.message === 'connected' || lastMessage.message === 'none') {
      return;
    }

    // Determine toast type based on message status
    let toastType: 'success' | 'error' | 'info' | 'notice' = 'info';

    if (lastMessage.status === 'complete' || lastMessage.status === 'canceled') {
      toastType = 'success';
    } else if (lastMessage.status === 'error') {
      toastType = 'error';
    } else if (lastMessage.status === 'scanning') {
      toastType = 'notice';
    }

    // Show toast notification
    addToast(lastMessage.message, toastType);
  }, [lastMessage, addToast]);
};
