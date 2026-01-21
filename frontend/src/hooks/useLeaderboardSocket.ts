import { useEffect, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { Platform } from 'react-native';

interface VersionUpdate {
  type: 'VERSION_UPDATE';
  version: number;
}

interface UseLeaderboardSocketOptions {
  enabled?: boolean;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Error) => void;
  isSearchActive?: boolean;
}

/**
 * Custom hook for managing WebSocket connection to receive real-time leaderboard updates
 * 
 * Features:
 * - Version-based updates (only fetches when version changes)
 * - Automatic reconnection on disconnect
 * - Heartbeat (ping/pong) to keep connection alive
 * - Updates React Query cache automatically
 * - Only runs on web platform (mobile uses polling)
 * - Respects search mode: when isSearchActive=true, skips cache invalidation
 * 
 * Architecture:
 * - Backend broadcasts version number every 2 seconds (if changed)
 * - Frontend tracks lastVersion and only invalidates cache when version increases
 * - Eliminates "request storm" by decoupling updates from broadcasts
 * - Search state protection: prevents background updates from clearing search results
 */
export const useLeaderboardSocket = (options: UseLeaderboardSocketOptions = {}) => {
  const { enabled = true, onConnect, onDisconnect, onError, isSearchActive = false } = options;
  const queryClient = useQueryClient();
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const heartbeatIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const lastVersionRef = useRef<number>(0);
  
  // Use refs to avoid triggering reconnections when values change
  const isSearchActiveRef = useRef(isSearchActive);
  const onConnectRef = useRef(onConnect);
  const onDisconnectRef = useRef(onDisconnect);
  const onErrorRef = useRef(onError);
  
  // Update refs when values change
  useEffect(() => {
    isSearchActiveRef.current = isSearchActive;
  }, [isSearchActive]);
  
  useEffect(() => {
    onConnectRef.current = onConnect;
  }, [onConnect]);
  
  useEffect(() => {
    onDisconnectRef.current = onDisconnect;
  }, [onDisconnect]);
  
  useEffect(() => {
    onErrorRef.current = onError;
  }, [onError]);

  const MAX_RECONNECT_ATTEMPTS = 5;
  const RECONNECT_INTERVAL = 3000; // 3 seconds
  const HEARTBEAT_INTERVAL = 30000; // 30 seconds

  // Only enable WebSocket on web platform
  const isWebSocketSupported = Platform.OS === 'web' && typeof WebSocket !== 'undefined';

  useEffect(() => {
    if (!isWebSocketSupported || !enabled) {
      return;
    }

    const connect = () => {
      try {
        // Determine WebSocket URL based on current location
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = window.location.hostname;
        const port = '8000'; // Backend port
        const wsUrl = `${protocol}//${host}:${port}/ws`;

        console.log(`[WebSocket] Connecting to ${wsUrl}...`);

        const ws = new WebSocket(wsUrl);
        wsRef.current = ws;

        ws.onopen = () => {
          console.log('[WebSocket] Connected successfully');
          console.log('[WebSocket] Ready state:', ws.readyState);
          reconnectAttemptsRef.current = 0;
          onConnectRef.current?.();

          // Start heartbeat
          heartbeatIntervalRef.current = setInterval(() => {
            if (ws.readyState === WebSocket.OPEN) {
              // WebSocket ping/pong is handled automatically by the browser
              console.log('[WebSocket] Heartbeat check - Connection alive');
            } else {
              console.log('[WebSocket] Heartbeat check - Connection NOT alive, state:', ws.readyState);
            }
          }, HEARTBEAT_INTERVAL);
        };

        ws.onmessage = (event) => {
          try {
            const update: VersionUpdate = JSON.parse(event.data);
            
            if (update.type === 'VERSION_UPDATE') {
              const currentSearchState = isSearchActiveRef.current;
              console.log(`[WebSocket] Received version update: ${update.version} (previous: ${lastVersionRef.current}, searchActive: ${currentSearchState})`);

              // Only invalidate queries if version has changed
              if (update.version > lastVersionRef.current) {
                lastVersionRef.current = update.version;
                
                // Skip cache invalidation if search is active to preserve search results
                if (currentSearchState) {
                  console.log('[WebSocket] ⚠️ Search active - skipping cache invalidation to preserve search results');
                  return;
                }
                
                console.log('[WebSocket] ✅ Version increased - invalidating leaderboard cache');

                // Invalidate and refetch the leaderboard query to get fresh data
                queryClient.invalidateQueries({ 
                  queryKey: ['leaderboard'],
                  refetchType: 'active', // Refetch active queries
                });
              } else {
                console.log('[WebSocket] ℹ️ Version unchanged - skipping cache invalidation');
              }
            } else {
              console.warn('[WebSocket] Unknown message type:', update.type);
            }

          } catch (error) {
            console.error('[WebSocket] Failed to parse message:', error);
          }
        };

        ws.onerror = (error) => {
          console.error('[WebSocket] Error occurred:', error);
          console.error('[WebSocket] Ready state on error:', ws.readyState);
          console.error('[WebSocket] Error event:', {
            type: error.type,
            target: error.target,
            currentTarget: error.currentTarget,
          });
          onErrorRef.current?.(new Error('WebSocket connection error'));
        };

        ws.onclose = (event) => {
          console.log(`[WebSocket] Connection closed (code: ${event.code}, reason: ${event.reason})`);
          onDisconnectRef.current?.();

          // Clear heartbeat
          if (heartbeatIntervalRef.current) {
            clearInterval(heartbeatIntervalRef.current);
            heartbeatIntervalRef.current = null;
          }

          // Attempt to reconnect if not intentionally closed
          if (event.code !== 1000 && reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
            reconnectAttemptsRef.current++;
            console.log(
              `[WebSocket] Reconnecting... (attempt ${reconnectAttemptsRef.current}/${MAX_RECONNECT_ATTEMPTS})`
            );

            reconnectTimeoutRef.current = setTimeout(() => {
              connect();
            }, RECONNECT_INTERVAL);
          } else if (reconnectAttemptsRef.current >= MAX_RECONNECT_ATTEMPTS) {
            console.error('[WebSocket] Max reconnection attempts reached');
            onErrorRef.current?.(new Error('Failed to reconnect after multiple attempts'));
          }
        };

      } catch (error) {
        console.error('[WebSocket] Connection failed:', error);
        onErrorRef.current?.(error as Error);
      }
    };

    // Initial connection
    connect();

    // Cleanup on unmount
    return () => {
      if (wsRef.current) {
        wsRef.current.close(1000, 'Component unmounted');
        wsRef.current = null;
      }

      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }

      if (heartbeatIntervalRef.current) {
        clearInterval(heartbeatIntervalRef.current);
        heartbeatIntervalRef.current = null;
      }
    };
  }, [isWebSocketSupported, enabled, queryClient]);

  return {
    isConnected: wsRef.current?.readyState === WebSocket.OPEN,
    isSupported: isWebSocketSupported,
  };
};
