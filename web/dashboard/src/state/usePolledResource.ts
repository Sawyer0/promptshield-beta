import { useCallback, useEffect, useRef, useState } from 'react';
import { UnauthorizedError } from '../api/gatewayClient';
import { startPolling } from './pollingSchedule';

export interface PolledResource<T> {
  data: T | null;
  error: string | null;
  refresh: () => void;
}

interface PollOptions {
  intervalMs: number;
  /** Changing this value triggers an immediate (throttled) refresh. */
  trigger?: unknown;
  onUnauthorized?: () => void;
  throttleMs?: number;
}

/**
 * Loads a value, keeps it fresh on an interval, and refreshes early when
 * `trigger` changes (for example, on each live decision). Early refreshes
 * are throttled so a burst of events causes one request, not hundreds.
 */
export function usePolledResource<T>(load: () => Promise<T>, options: PollOptions): PolledResource<T> {
  const { intervalMs, trigger, onUnauthorized, throttleMs = 400 } = options;
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const pending = useRef<ReturnType<typeof setTimeout> | null>(null);
  const alive = useRef(true);

  const run = useCallback(async () => {
    try {
      const value = await load();
      if (!alive.current) return;
      setData(value);
      setError(null);
    } catch (err) {
      if (!alive.current) return;
      if (err instanceof UnauthorizedError) {
        onUnauthorized?.();
        return;
      }
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [load, onUnauthorized]);

  const refresh = useCallback(() => {
    if (pending.current) return;
    pending.current = setTimeout(() => {
      pending.current = null;
      void run();
    }, throttleMs);
  }, [run, throttleMs]);

  useEffect(() => {
    alive.current = true;
    const stop = startPolling(() => void run(), intervalMs);
    return () => {
      alive.current = false;
      stop();
      if (pending.current) clearTimeout(pending.current);
      pending.current = null;
    };
  }, [run, intervalMs]);

  const first = useRef(true);
  useEffect(() => {
    if (first.current) {
      first.current = false;
      return;
    }
    refresh();
  }, [trigger, refresh]);

  return { data, error, refresh };
}
