import { useEffect, useState } from 'react';
import type { DecisionRecord, StreamStatus } from '../api/contracts';
import { runEventStream } from '../api/eventStream';
import { UnauthorizedError, type GatewayClient } from '../api/gatewayClient';
import { mergeDecisions } from './decisionMerge';

export interface DecisionFeed {
  records: DecisionRecord[];
  status: StreamStatus;
  error: string | null;
  /** Increments on every live decision; drives stats refreshes. */
  eventCount: number;
  /** Increments when the gateway reports a rules reload. */
  rulesVersion: number;
}

const HISTORY_LIMIT = 100;

/**
 * Recent decisions, kept live: loads history from /v1/decisions, then
 * merges "decision" events from /v1/events. Every time the stream becomes
 * live again it re-syncs history, so nothing is lost across reconnects.
 */
export function useDecisionFeed(client: GatewayClient, onUnauthorized: () => void): DecisionFeed {
  const [records, setRecords] = useState<DecisionRecord[]>([]);
  const [status, setStatus] = useState<StreamStatus>('connecting');
  const [error, setError] = useState<string | null>(null);
  const [eventCount, setEventCount] = useState(0);
  const [rulesVersion, setRulesVersion] = useState(0);

  useEffect(() => {
    const controller = new AbortController();
    const resync = async () => {
      try {
        const history = await client.listDecisions({ limit: HISTORY_LIMIT });
        if (controller.signal.aborted) return;
        setRecords((prev) => mergeDecisions(history, prev));
        setError(null);
      } catch (err) {
        if (controller.signal.aborted) return;
        if (err instanceof UnauthorizedError) onUnauthorized();
        else setError(err instanceof Error ? err.message : String(err));
      }
    };

    void resync();
    void runEventStream(
      { client, signal: controller.signal },
      {
        onDecision: (record) => {
          setRecords((prev) => mergeDecisions([record], prev));
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

  return { records, status, error, eventCount, rulesVersion };
}
