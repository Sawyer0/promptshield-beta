import type { RulePackSummary } from '../../api/contracts';
import { formatDateTime } from '../../display/format';
import type { ActiveRules } from '../../state/useActiveRules';

function PackTable({ pack }: { pack: RulePackSummary }) {
  return (
    <article className="pack">
      <h3 className="pack__title">
        {pack.name} <span className="muted">v{pack.version || '?'}</span>
      </h3>
      {pack.description && <p className="muted">{pack.description}</p>}
      <div className="table-scroll">
        <table className="table">
          <caption className="visually-hidden">Rules in {pack.name}</caption>
          <thead>
            <tr>
              <th scope="col">Rule</th>
              <th scope="col">Name</th>
              <th scope="col">Severity</th>
              <th scope="col">Action</th>
              <th scope="col" className="num">
                Level
              </th>
              <th scope="col" className="num">
                Terms
              </th>
              <th scope="col">State</th>
            </tr>
          </thead>
          <tbody>
            {pack.rules.map((r) => (
              <tr key={r.id} data-testid="rule-row">
                <td className="mono">{r.id}</td>
                <td>{r.name}</td>
                <td>{r.severity && <span className={`severity severity--${r.severity.toLowerCase()}`}>{r.severity}</span>}</td>
                <td>{r.action || <span className="muted">flag</span>}</td>
                <td className="num">{r.level}</td>
                <td className="num" title={`${r.keywords} keywords, ${r.patterns} patterns`}>
                  {r.keywords + r.patterns}
                </td>
                <td>{r.enabled ? 'Enabled' : <span className="muted">Disabled</span>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </article>
  );
}

export function RulesPanel({ rules }: { rules: ActiveRules }) {
  const { snapshot, loadError, reloading, reloadError, reload } = rules;
  return (
    <section className="panel" aria-labelledby="rules-heading">
      <div className="panel__header">
        <div>
          <h2 id="rules-heading">Active rules</h2>
          {snapshot && (
            <p className="muted" data-testid="rules-summary">
              {snapshot.ruleCount} rules in {snapshot.packs.length} {snapshot.packs.length === 1 ? 'pack' : 'packs'} · generation{' '}
              {snapshot.generation} · loaded {formatDateTime(snapshot.loadedAt)}
            </p>
          )}
        </div>
        <button type="button" className="button" onClick={() => void reload()} disabled={reloading}>
          {reloading ? 'Reloading…' : 'Reload from disk'}
        </button>
      </div>
      {snapshot?.source && (
        <p className="muted small">
          Source: <code>{snapshot.source}</code>
        </p>
      )}
      {reloadError && (
        <p className="error" role="alert">
          Reload failed: {reloadError}
        </p>
      )}
      {loadError && (
        <p className="error" role="alert">
          Rules unavailable: {loadError}
        </p>
      )}
      {snapshot && snapshot.packs.length === 0 && (
        <p className="empty">No RulePacks are loaded. Set PS_ENFORCER_RULEPACK and reload.</p>
      )}
      {snapshot?.packs.map((p) => <PackTable key={p.name} pack={p} />)}
    </section>
  );
}
