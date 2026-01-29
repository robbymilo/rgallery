import React, { createContext, useContext, useEffect, useState, useRef, useCallback } from 'react';
import { useAuth } from './AuthContext';

interface WebSocketMessage {
  message: string;
  status?: string;
  id?: number;
  username?: string;
  is_read?: boolean;
  created_at?: string;
}

interface WebSocketContextType {
  isConnected: boolean;
  isScanInProgress: boolean;
  lastMessage: WebSocketMessage | null;
  sendMessage: (message: string) => void;
  reconnectWebSocket: () => void;
}

const WebSocketContext = createContext<WebSocketContextType | undefined>(undefined);

export const useWebSocket = () => {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error('useWebSocket must be used within a WebSocketProvider');
  }
  return context;
};

interface WebSocketProviderProps {
  children: React.ReactNode;
}

export const WebSocketProvider: React.FC<WebSocketProviderProps> = ({ children }) => {
  const { isAuthenticated } = useAuth();
  const [isConnected, setIsConnected] = useState(false);
  const [isScanInProgress, setIsScanInProgress] = useState(false);
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptsRef = useRef(0);

  const connect = useCallback(() => {
    if (!isAuthenticated) return;

    // Determine WebSocket URL based on current location
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/ws`;

    try {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        console.log('WebSocket connected');
        setIsConnected(true);
        reconnectAttemptsRef.current = 0;
      };

      ws.onmessage = (event) => {
        try {
          const data: WebSocketMessage = JSON.parse(event.data);
          console.log('WebSocket message received:', data);
          setLastMessage(data);

          // Update scan status based on message content or status field
          const msg = (data.message || '').toString().toLowerCase();
          if (msg.includes('scan canceled') || msg.includes('scan started') || msg.includes('scan complete')) {
            // If canceled or complete, set to false; if started, set to true
            if (msg.includes('scan canceled') || msg.includes('scan complete')) {
              setIsScanInProgress(false);
            } else {
              setIsScanInProgress(true);
            }
          } else if (typeof data.status === 'string') {
            // Fallback to status field if present
            if (data.status === 'scanning') {
              setIsScanInProgress(true);
            } else if (data.status === 'complete' || data.status === 'canceled' || data.status === 'error') {
              setIsScanInProgress(false);
            }
          }
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
      };

      ws.onclose = (event) => {
        console.log('WebSocket disconnected', event.code, event.reason);
        setIsConnected(false);
        wsRef.current = null;

        // Only reconnect if it wasn't a normal closure
        if (event.code !== 1000) {
          // Attempt to reconnect with exponential backoff
          const backoffDelay = Math.min(100 * Math.pow(2, reconnectAttemptsRef.current), 30000);
          reconnectAttemptsRef.current++;

          console.log(`Reconnecting in ${backoffDelay}ms (attempt ${reconnectAttemptsRef.current})...`);
          reconnectTimeoutRef.current = setTimeout(connect, backoffDelay);
        }
      };
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
    }
  }, [isAuthenticated]);

  // Expose a reconnect function
  const reconnectWebSocket = useCallback(() => {
    if (!isAuthenticated) return;
    if (wsRef.current) {
      wsRef.current.close(); // onclose will trigger reconnect
    } else {
      connect();
    }
  }, [connect, isAuthenticated]);

  useEffect(() => {
    if (isAuthenticated) {
      connect();
    } else {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
        setIsConnected(false);
      }
    }

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [connect, isAuthenticated]);

  const sendMessage = useCallback((message: string) => {
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      wsRef.current.send(message);
    } else {
      console.warn('WebSocket is not connected');
    }
  }, []);

  const value: WebSocketContextType = {
    isConnected,
    isScanInProgress,
    lastMessage,
    sendMessage,
    reconnectWebSocket,
  };

  return <WebSocketContext.Provider value={value}>{children}</WebSocketContext.Provider>;
};
