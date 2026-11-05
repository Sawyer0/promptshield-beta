import type { GatewayHealth, StreamStatus } from '../../api/contracts';

const STREAM_LABEL: Record<StreamStatus, string> = {
  connecting: 'Connecting',
  live: 'Live',
  reconnecting: 'Reconnecting',
  unauthorized: 'Signed out',
  closed: 'Disconnected',
};

interface AppHeaderProps {
  health: GatewayHealth | null;
  stream: StreamStatus;
  onSignOut: () => void;
}

function Indicator({ ok, label }: { ok: boolean | null; label: string }) {
  const state = ok === null ? 'unknown' : ok ? 'ok' : 'down';
  return (
    <li className={`indicator indicator--${state}`}>
      <span className="indicator__dot" aria-hidden="true" />
      {label}: {ok === null ? 'checking' : ok ? 'yes' : 'no'}
    </li>
  );
}

export function AppHeader({ health, stream, onSignOut }: AppHeaderProps) {
  return (
    <header className="app-header">
      <div className="app-header__brand">
        <span className="app-header__logo" aria-hidden="true">
          ◆
        </span>
        <div>
          <p className="app-header__title">PromptShield Gateway</p>
          <p className="app-header__subtitle">Version {health?.version ?? '…'}</p>
        </div>
      </div>
      <ul className="app-header__status" aria-label="Gateway status">
        <Indicator ok={health ? health.live : null} label="Healthy" />
        <Indicator ok={health ? health.ready : null} label="Ready" />
        <li className={`stream stream--${stream}`} role="status" aria-live="polite">
          <span className="indicator__dot" aria-hidden="true" />
          Event stream: {STREAM_LABEL[stream]}
        </li>
      </ul>
      <button type="button" className="button" onClick={onSignOut}>
        Sign out
      </button>
    </header>
  );
}
