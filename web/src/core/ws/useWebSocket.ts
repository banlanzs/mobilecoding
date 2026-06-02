// useWebSocket hook：包装 WSClient，集成 React 生命周期
import { useEffect, useRef, useState, useCallback } from 'react';
import { WSClient } from './ws-client';
import type { ConnectionStatus } from './types';

export function useWebSocket() {
  const clientRef = useRef<WSClient | null>(null);
  const [status, setStatus] = useState<ConnectionStatus>('idle');
  const [error, setError] = useState<string | null>(null);

  // 单例
  if (!clientRef.current) {
    clientRef.current = new WSClient();
  }
  const client = clientRef.current;

  useEffect(() => {
    const offStatus = client.onStatus((s) => {
      setStatus(s);
      if (s === 'connected') setError(null);
    });
    return () => {
      offStatus();
    };
  }, [client]);

  const connect = useCallback(
    (token: string) => {
      try {
        client.connect(token);
      } catch (e) {
        setError(e instanceof Error ? e.message : 'connect failed');
      }
    },
    [client]
  );

  const close = useCallback(() => {
    client.close();
  }, [client]);

  return {
    client,
    status,
    error,
    connect,
    close,
  };
}