import { useCallback, useEffect, useRef, useState } from 'react';
import type { DecisionRecord, StreamStatus } from '../api/contracts';
import { runEventStream } from '../api/eventStream';
import { UnauthorizedError, type GatewayClient } from '../api/gatewayClient';
import { appendOlder, mergeNewer, oldestSeq, type FeedWindow } from './decisionMerge';

export interface OlderDecisions {
  /** The gateway still holds decisions older than the feed's last row. */
  available: boolean;
  loading: boolean;
  error: string | null;
  load: () => void;
}

export interface DecisionFeed {
  records: DecisionRecord[];
  status: StreamStatus;
  error: string | null;
  older: OlderDecisions;
  /** Increments on every live decision; drives stats refreshes. */
  eventCount: number;
  /** Increments when the gateway reports a rules reload. */
  rulesVersion: number;
}

export const HISTORY_LIMIT = 100;

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

/**
 * Recent decisions, kept live: loads history from /v1/decisions, then
 * merges "decision" events from /v1/events. Every time the stream becomes
 * live again it re-syncs history, so nothing is lost across reconnects.
 * older.load() pages further back with the before cursor.
 */
export function useDecisionFeed(client: GatewayClient, onUnauthorized: () => void): DecisionFeed {
  const [feed, setFeed] = useState<FeedWindow>({ records: [], hasOlder: false });
  const [status, setStatus] = useState<StreamStatus>('connecting');
  const [error, setError] = useState<string | null>(null);
  const [olderLoading, setOlderLoading] = useState(false);
  const [olderError, setOlderError] = useState<string | null>(null);
  const [eventCount, setEventCount] = useState(0);
  const [rulesVersion, setRulesVersion] = useState(0);
  const unmounted = useRef(new AbortController());

  useEffect(() => {
    const controller = new AbortController();
    unmounted.current = controller;
    const resync = async () => {
      try {
        const history = await client.listDecisions({ limit: HISTORY_LIMIT });
        if (controller.signal.aborted) return;
        setFeed((prev) => mergeNewer(history.records, { ...prev, hasOlder: prev.hasOlder || history.next !== null }));
        setError(null);
      } catch (err) {
        if (controller.signal.aborted) return;
        if (err instanceof UnauthorizedError) onUnauthorized();
        else setError(messageOf(err));
      }
    };

    void resync();
    void runEventStream(
      { client, signal: controller.signal },
      {
        onDecision: (record) => {
          setFeed((prev) => mergeNewer([record], prev));
          setEventCount((n) => n + 1);
        },
        onRulesReloaded: () => setRulesVersion((n) => n + 1),
        onStatus: (next) => {
          if (controller.signal.aborted) return;
          setStatus(next);
          if (next === 'live') void resync();
          if (next === 'unauthorized') onUnauthorized();
        },
      },
    );
    return () => controller.abort();
  }, [client, onUnauthorized]);

  const before = oldestSeq(feed.records);
  const load = useCallback(() => {
    if (before === null || olderLoading) return;
    const signal = unmounted.current.signal;
    setOlderLoading(true);
    setOlderError(null);
    client
      .listDecisions({ limit: HISTORY_LIMIT, before })
      .then((page) => {
        if (!signal.aborted) setFeed((prev) => appendOlder(page.records, page.next, prev));
      })
      .catch((err: unknown) => {
        if (signal.aborted) return;
        if (err instanceof UnauthorizedError) onUnauthorized();
        else setOlderError(messageOf(err));
      })
      .finally(() => {
        if (!signal.aborted) setOlderLoading(false);
      });
  }, [client, before, olderLoading, onUnauthorized]);

  return {
    records: feed.records,
    status,
    error,
    older: { available: feed.hasOlder && before !== null, loading: olderLoading, error: olderError, load },
    eventCount,
    rulesVersion,
  };
}
