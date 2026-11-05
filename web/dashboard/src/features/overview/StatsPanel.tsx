import { DECISION_OUTCOMES, type DecisionStats } from '../../api/contracts';
import { OUTCOME_LABEL, formatCount, formatLatency, formatPercent } from '../../display/format';

interface StatsPanelProps {
  stats: DecisionStats | null;
  error: string | null;
}

function Stat({ id, label, value, detail }: { id: string; label: string; value: string; detail: string }) {
  return (
    <div className="stat" data-testid={`stat-${id}`}>
      <dt className="stat__label">{label}</dt>
      <dd className="stat__value">{value}</dd>
      <dd className="stat__detail">{detail}</dd>
    </div>
  );
}

function DecisionMix({ stats }: { stats: DecisionStats }) {
  const total = Math.max(stats.total, 1);
  return (
    <figure className="mix" aria-labelledby="mix-caption">
      <figcaption id="mix-caption" className="stat__label">
        Decision mix
      </figcaption>
      <div className="mix__bar" aria-hidden="true">
        {DECISION_OUTCOMES.map((d) => (
          <span key={d} className={`mix__segment mix__segment--${d}`} style={{ width: `${(stats.byDecision[d] / total) * 100}%` }} />
        ))}
      </div>
      <ul className="mix__legend">
        {DECISION_OUTCOMES.map((d) => (
          <li key={d} data-testid={`mix-${d}`}>
            <span className={`mix__swatch mix__swatch--${d}`} aria-hidden="true" />
            {OUTCOME_LABEL[d]} <strong>{formatCount(stats.byDecision[d])}</strong>
          </li>
        ))}
      </ul>
    </figure>
  );
}

export function StatsPanel({ stats, error }: StatsPanelProps) {
  return (
    <section className="panel" aria-labelledby="overview-heading">
      <div className="panel__header">
        <h2 id="overview-heading">Overview</h2>
        {stats && <p className="muted">Retaining the last {formatCount(stats.capacity)} decisions</p>}
      </div>
      {error && (
        <p className="error" role="alert">
          Stats unavailable: {error}
        </p>
      )}
      {!stats && !error && <p className="muted">Loading stats…</p>}
      {stats && (
        <div className="stats">
          <dl className="stats__grid">
            <Stat id="total" label="Requests" value={formatCount(stats.total)} detail={`${formatCount(stats.lastMinute)} in the last minute`} />
            <Stat
              id="blocked"
              label="Blocked or quarantined"
              value={formatCount(stats.byDecision.deny + stats.byDecision.quarantine)}
              detail={`${formatPercent(stats.blockRate)} of requests`}
            />
            <Stat
              id="allowed"
              label="Allowed"
              value={formatCount(stats.byDecision.allow)}
              detail={`${formatPercent(stats.total ? stats.byDecision.allow / stats.total : 0)} of requests`}
            />
            <Stat id="p95" label="p95 latency" value={formatLatency(stats.latencyP95Ms)} detail={`p50 ${formatLatency(stats.latencyP50Ms)}`} />
          </dl>
          <DecisionMix stats={stats} />
        </div>
      )}
    </section>
  );
}
