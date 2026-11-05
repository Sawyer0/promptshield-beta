import { useMemo, useState } from 'react';
import type { DecisionRecord, StreamStatus } from '../../api/contracts';
import { OutcomeBadge } from '../../display/OutcomeBadge';
import { formatClock, formatLatency } from '../../display/format';
import { filterDecisions, type FeedFilter } from '../../state/decisionMerge';
import type { OlderDecisions } from '../../state/useDecisionFeed';
import { FeedFilters } from './FeedFilters';

interface DecisionFeedProps {
  records: DecisionRecord[];
  status: StreamStatus;
  error: string | null;
  selectedId: string | null;
  onSelect: (record: DecisionRecord) => void;
  older?: OlderDecisions;
}

function ruleOf(record: DecisionRecord): string {
  return record.matches[0]?.ruleId ?? (record.reason === 'no_signals' ? '—' : record.reason);
}

export function DecisionFeed({ records, status, error, selectedId, onSelect, older }: DecisionFeedProps) {
  const [filter, setFilter] = useState<FeedFilter>({ outcome: 'all', search: '' });
  const visible = useMemo(() => filterDecisions(records, filter), [records, filter]);

  return (
    <section className="panel panel--feed" aria-labelledby="feed-heading">
      <div className="panel__header">
        <h2 id="feed-heading">Live decisions</h2>
        <p className="muted" aria-live="polite" data-testid="feed-count">
          Showing {visible.length} of {records.length}
        </p>
      </div>
      <FeedFilters filter={filter} onChange={setFilter} />
      {error && (
        <p className="error" role="alert">
          Could not load history: {error}
        </p>
      )}
      <div className="table-scroll">
        <table className="table">
          <caption className="visually-hidden">Recent enforcement decisions, newest first</caption>
          <thead>
            <tr>
              <th scope="col">Time</th>
              <th scope="col">Decision</th>
              <th scope="col">Rule</th>
              <th scope="col">Path</th>
              <th scope="col" className="num">
                Latency
              </th>
              <th scope="col" className="num">
                Status
              </th>
            </tr>
          </thead>
          <tbody>
            {visible.map((r) => (
              <tr key={r.requestId} className={r.requestId === selectedId ? 'is-selected' : undefined} data-testid="decision-row">
                <td>
                  <button
                    type="button"
                    className="link-button"
                    onClick={() => onSelect(r)}
                    aria-label={`Inspect decision ${r.requestId}`}
                    aria-current={r.requestId === selectedId ? 'true' : undefined}
                  >
                    {formatClock(r.time)}
                  </button>
                </td>
                <td>
                  <OutcomeBadge outcome={r.decision} />
                </td>
                <td className="mono">{ruleOf(r)}</td>
                <td className="mono">{r.path}</td>
                <td className="num">{formatLatency(r.latencyMs)}</td>
                <td className="num">{r.status}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {visible.length === 0 && (
          <p className="empty">
            {records.length === 0
              ? status === 'live'
                ? 'Waiting for traffic. Send a request to /check to see it here.'
                : 'Connecting to the event stream…'
              : 'No decisions match the current filter.'}
          </p>
        )}
      </div>
      {older?.error && (
        <p className="error" role="alert">
          Could not load older decisions: {older.error}
        </p>
      )}
      {older?.available && (
        <div className="feed__more">
          <button type="button" className="button" onClick={older.load} disabled={older.loading} aria-busy={older.loading}>
            {older.loading ? 'Loading older…' : 'Load older'}
          </button>
        </div>
      )}
    </section>
  );
}
