import { useCallback, useState } from 'react';
import type { DecisionRecord } from './api/contracts';
import type { FetchLike } from './api/gatewayClient';
import { TokenForm } from './features/auth/TokenForm';
import { DecisionDetail } from './features/detail/DecisionDetail';
import { DecisionFeed } from './features/feed/DecisionFeed';
import { AppHeader } from './features/layout/AppHeader';
import { StatsPanel } from './features/overview/StatsPanel';
import { PromptTester } from './features/playground/PromptTester';
import { RulesPanel } from './features/rules/RulesPanel';
import { SessionProvider, useClient, useSession } from './session/SessionContext';
import { useActiveRules } from './state/useActiveRules';
import { useDecisionFeed } from './state/useDecisionFeed';
import { usePolledResource } from './state/usePolledResource';

const STATS_INTERVAL_MS = 5_000;
const HEALTH_INTERVAL_MS = 10_000;

function Dashboard() {
  const client = useClient();
  const { signOut, expire } = useSession();
  const feed = useDecisionFeed(client, expire);
  const loadStats = useCallback(() => client.getStats(), [client]);
  const loadHealth = useCallback(() => client.getHealth(), [client]);
  const stats = usePolledResource(loadStats, { intervalMs: STATS_INTERVAL_MS, trigger: feed.eventCount, onUnauthorized: expire });
  const health = usePolledResource(loadHealth, { intervalMs: HEALTH_INTERVAL_MS });
  const rules = useActiveRules(client, feed.rulesVersion, expire);
  const [selected, setSelected] = useState<DecisionRecord | null>(null);

  return (
    <div className="app">
      <a className="skip-link" href="#main">
        Skip to content
      </a>
      <AppHeader health={health.data} stream={feed.status} onSignOut={signOut} />
      <main id="main" className="layout">
        <StatsPanel stats={stats.data} error={stats.error} />
        <div className="layout__columns">
          <DecisionFeed
            records={feed.records}
            status={feed.status}
            error={feed.error}
            selectedId={selected?.requestId ?? null}
            onSelect={setSelected}
          />
          <div className="layout__side">
            {selected && <DecisionDetail record={selected} onClose={() => setSelected(null)} />}
            <PromptTester />
          </div>
        </div>
        <RulesPanel rules={rules} />
      </main>
    </div>
  );
}

function Root() {
  const { client } = useSession();
  return client ? <Dashboard key={client.token} /> : <TokenForm />;
}

interface AppProps {
  baseUrl?: string;
  fetchFn?: FetchLike;
}

export function App({ baseUrl, fetchFn }: AppProps) {
  return (
    <SessionProvider baseUrl={baseUrl ?? ''} fetchFn={fetchFn}>
      <Root />
    </SessionProvider>
  );
}
