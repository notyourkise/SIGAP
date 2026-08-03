'use client';

import React, { createContext, useContext, useEffect, useRef, useState, useCallback } from 'react';
import { API_BASE_URL } from '@/lib/api';
import { useAuth } from './AuthContext';

interface EventCallback {
  (payload: any): void;
}

interface WebSocketContextType {
  isConnected: boolean;
  subscribe: (event: string, callback: EventCallback) => () => void;
}

const WebSocketContext = createContext<WebSocketContextType | undefined>(undefined);

export function WebSocketProvider({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth();
  const [isConnected, setIsConnected] = useState(false);
  const socketRef = useRef<WebSocket | null>(null);
  const listenersRef = useRef<Map<string, Set<EventCallback>>>(new Map());
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const shouldReconnectRef = useRef(false);

  const connect = useCallback(() => {
    if (!isAuthenticated || !shouldReconnectRef.current || typeof window === 'undefined') return;

    // Ubah http:// ke ws:// atau https:// ke wss://
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const baseUrl = API_BASE_URL.startsWith('/')
      ? `${wsProtocol}//${window.location.host}${API_BASE_URL}`
      : API_BASE_URL.replace(/^http/, 'ws');
    // Autentikasi WebSocket via cookie httpOnly sigap_session yang dikirim
    // otomatis oleh browser pada handshake same-origin (tanpa token di URL).
    const wsUrl = `${baseUrl}/ws`;

    try {
      const socket = new WebSocket(wsUrl);
      socketRef.current = socket;

      socket.onopen = () => {
        setIsConnected(true);
        console.log('🔌 WebSocket real-time terhubung ke', wsUrl);
      };

      socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data && data.event) {
            const callbacks = listenersRef.current.get(data.event);
            if (callbacks) {
              callbacks.forEach((cb) => cb(data.payload));
            }
          }
        } catch {
          // Abaikan pesan non-JSON (seperti pong)
        }
      };

      socket.onclose = () => {
        if (socketRef.current === socket) {
          socketRef.current = null;
        }
        setIsConnected(false);
        if (!shouldReconnectRef.current) return;

        // Otomatisasi auto-reconnect setiap 5 detik hanya selama sesi aktif.
        reconnectTimeoutRef.current = setTimeout(() => {
          reconnectTimeoutRef.current = null;
          if (shouldReconnectRef.current) connect();
        }, 5000);
      };

      socket.onerror = () => {
        socket.close();
      };
    } catch (err) {
      console.error('Gagal membuka koneksi WebSocket:', err);
    }
  }, [isAuthenticated]);

  useEffect(() => {
    shouldReconnectRef.current = isAuthenticated;

    if (isAuthenticated) {
      connect();
    } else {
      if (socketRef.current) {
        socketRef.current.close();
      }
      setIsConnected(false);
    }

    return () => {
      shouldReconnectRef.current = false;
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      if (socketRef.current) {
        socketRef.current.close();
      }
    };
  }, [isAuthenticated, connect]);

  const subscribe = useCallback((event: string, callback: EventCallback) => {
    if (!listenersRef.current.has(event)) {
      listenersRef.current.set(event, new Set());
    }
    listenersRef.current.get(event)!.add(callback);

    // Return cleanup function untuk unsubscribing
    return () => {
      const callbacks = listenersRef.current.get(event);
      if (callbacks) {
        callbacks.delete(callback);
        if (callbacks.size === 0) {
          listenersRef.current.delete(event);
        }
      }
    };
  }, []);

  return (
    <WebSocketContext.Provider value={{ isConnected, subscribe }}>
      {children}
    </WebSocketContext.Provider>
  );
}

export function useWebSocket() {
  const context = useContext(WebSocketContext);
  if (context === undefined) {
    throw new Error('useWebSocket harus digunakan di dalam WebSocketProvider');
  }
  return context;
}
