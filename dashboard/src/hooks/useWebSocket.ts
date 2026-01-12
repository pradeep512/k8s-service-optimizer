import { useEffect, useRef, useState, useCallback } from 'react';
import type { WebSocketMessage } from '../services/types';

export interface UseWebSocketOptions {
  onMessage?: (message: WebSocketMessage) => void;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
}

export interface UseWebSocketReturn {
  isConnected: boolean;
  error: string | null;
  reconnectCount: number;
  sendMessage: (message: any) => void;
  close: () => void;
}

export function useWebSocket(
  url: string,
  options: UseWebSocketOptions = {}
): UseWebSocketReturn {
  const {
    onMessage,
    reconnectInterval = 5000,
    maxReconnectAttempts = 10,
  } = options;

  const ws = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<number | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reconnectCount, setReconnectCount] = useState(0);
  const shouldReconnect = useRef(true);

  const connect = useCallback(() => {
    try {
      ws.current = new WebSocket(url);

      ws.current.onopen = () => {
        console.log('WebSocket connected');
        setIsConnected(true);
        setError(null);
        setReconnectCount(0);
      };

      ws.current.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          onMessage?.(message);
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };

      ws.current.onerror = (event) => {
        console.error('WebSocket error:', event);
        setError('WebSocket connection error');
      };

      ws.current.onclose = (event) => {
        console.log('WebSocket disconnected', event.code, event.reason);
        setIsConnected(false);

        // Attempt to reconnect if not intentionally closed
        if (shouldReconnect.current && reconnectCount < maxReconnectAttempts) {
          setReconnectCount((prev) => prev + 1);
          reconnectTimer.current = window.setTimeout(() => {
            console.log(`Attempting to reconnect (${reconnectCount + 1}/${maxReconnectAttempts})...`);
            connect();
          }, reconnectInterval);
        } else if (reconnectCount >= maxReconnectAttempts) {
          setError('Max reconnection attempts reached');
        }
      };
    } catch (err) {
      console.error('Failed to create WebSocket connection:', err);
      setError('Failed to create WebSocket connection');
    }
  }, [url, onMessage, reconnectInterval, maxReconnectAttempts, reconnectCount]);

  useEffect(() => {
    shouldReconnect.current = true;
    connect();

    return () => {
      shouldReconnect.current = false;
      if (reconnectTimer.current) {
        clearTimeout(reconnectTimer.current);
      }
      if (ws.current) {
        ws.current.close();
      }
    };
  }, [connect]);

  const sendMessage = useCallback((message: any) => {
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify(message));
    } else {
      console.warn('WebSocket is not connected. Message not sent.');
    }
  }, []);

  const close = useCallback(() => {
    shouldReconnect.current = false;
    if (reconnectTimer.current) {
      clearTimeout(reconnectTimer.current);
    }
    if (ws.current) {
      ws.current.close();
    }
  }, []);

  return { isConnected, error, reconnectCount, sendMessage, close };
}
