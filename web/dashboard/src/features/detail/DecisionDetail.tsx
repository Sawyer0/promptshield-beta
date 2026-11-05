import { useEffect, useRef, useState } from 'react';
import type { DecisionRecord } from '../../api/contracts';
import { GatewayError, UnauthorizedError } from '../../api/gatewayClient';
import { OutcomeBadge } from '../../display/OutcomeBadge';
import { formatBytes, formatDateTime, formatLatency } from '../../display/format';
import { useClient, useSession } from '../../session/SessionContext';

interface DecisionDetailProps {
  /** The feed's copy; shown immediately while the gateway copy loads. */
  record: DecisionRecord;
  onClose: () => void;
}

/** Full view of one decision, read from GET /v1/decisions/{id}. */
export function DecisionDetail({ record, onClose }: DecisionDetailProps) {
  const client = useClient();
  const { expire } = useSession();
  const [loaded, setLoaded] = useState<{ id: string; record: DecisionRecord | null; note: string | null } | null>(null);
  const heading = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    let cancelled = false;
    client
      .getDecision(record.requestId)
      .then((fresh) => !cancelled && setLoaded({ id: record.requestId, record: fresh, note: null }))
      .catch((err: unknown) => {
        if (cancelled) return;
        if (err instanceof UnauthorizedError) expire();
        const evicted = err instanceof GatewayError && err.status === 404;
        setLoaded({ id: record.requestId, record: null, note: evicted ? 'No longer retained by the gateway; showing the feed copy.' : 'Could not refresh from the gateway.' });
      });
    return () => {
      cancelled = true;
    };
  }, [client, expire, record.requestId]);

  useEffect(() => heading.current?.focus(), [record.requestId]);

  const current = loaded?.id === record.requestId ? loaded : null;
  const shown = current?.record ?? record;

  return (
    <aside
      className="panel detail"
      aria-labelledby="detail-heading"
      onKeyDown={(e) => {
        if (e.key === 'Escape') onClose();
      }}
    >
      <div className="panel__header">
        <h2 id="detail-heading" tabIndex={-1} ref={heading}>
          Decision detail
        </h2>
        <button type="button" className="button" onClick={onClose}>
          Close
        </button>
      </div>
      {current?.note && <p className="notice">{current.note}</p>}
      <dl className="detail__grid">
        <dt>Outcome</dt>
        <dd>
          <OutcomeBadge outcome={shown.decision} />
        </dd>
        <dt>Request id</dt>
        <dd className="mono" data-testid="detail-request-id">
          {shown.requestId}
        </dd>
        <dt>Time</dt>
        <dd>{formatDateTime(shown.time)}</dd>
        <dt>Endpoint</dt>
        <dd className="mono">
          POST {shown.path} → {shown.status}
        </dd>
        <dt>Reason</dt>
        <dd className="mono">{shown.reason}</dd>
        <dt>Latency</dt>
        <dd>{formatLatency(shown.latencyMs)}</dd>
        <dt>Scanned</dt>
        <dd>{formatBytes(shown.bytesIn)}</dd>
      </dl>
      <h3>Matched rules ({shown.violations})</h3>
      {shown.matches.length === 0 ? (
        <p className="muted">No rules matched.</p>
      ) : (
        <ul className="matches">
          {shown.matches.map((m, i) => (
            <li key={`${m.ruleId}-${i}`} className="match">
              <p className="match__title">
                <span className="mono">{m.ruleId}</span>
                {m.severity && <span className={`severity severity--${m.severity.toLowerCase()}`}>{m.severity}</span>}
                {m.action && <span className="muted">action: {m.action}</span>}
              </p>
              {m.message && <p className="match__message">{m.message}</p>}
            </li>
          ))}
        </ul>
      )}
      <p className="muted small">Prompt text is never stored by the gateway; only the outcome is kept.</p>
    </aside>
  );
}
