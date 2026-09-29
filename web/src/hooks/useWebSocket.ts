import { useEffect, useRef } from 'react';
import { useInspectorStore } from '../stores/useInspectorStore';
import { WebSocketEvent } from '../types';

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const retryDelayRef = useRef(1000);

  const addRequest = useInspectorStore((s) => s.addRequest);
  const setWsConnected = useInspectorStore((s) => s.setWsConnected);
  const setTunnel = useInspectorStore((s) => s.setTunnel);
  const setRequests = useInspectorStore((s) => s.setRequests);

  useEffect(() => {
    let unmounted = false;

    function connect() {
      if (unmounted) return;

      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${protocol}//${window.location.host}/ws`;

      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        if (unmounted) {
          ws.close();
          return;
        }
        setWsConnected(true);
        retryDelayRef.current = 1000; // Reset backoff
      };

      ws.onmessage = (event) => {
        try {
          const lines = event.data.split('\n');
          for (const line of lines) {
            if (!line.trim()) continue;
            const data: WebSocketEvent = JSON.parse(line);

            if (data.type === 'request.completed' && data.request) {
              addRequest(data.request);
            } else if (data.type === 'requests.cleared') {
              setRequests([], 0);
            } else if (data.type === 'tunnel.status') {
              if (data.status) {
                setTunnel({
                  status: data.status as any,
                  url: data.url || '',
                });
              }
            }
          }
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };

      ws.onclose = () => {
        setWsConnected(false);
        wsRef.current = null;

        if (!unmounted) {
          // Reconnect with backoff
          const delay = retryDelayRef.current;
          retryDelayRef.current = Math.min(delay * 1.5, 10000);
          reconnectTimeoutRef.current = window.setTimeout(connect, delay);
        }
      };

      ws.onerror = (err) => {
        console.warn('WebSocket encountered error:', err);
        ws.close();
      };
    }

    connect();

    return () => {
      unmounted = true;
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [addRequest, setWsConnected, setTunnel, setRequests]);
}
