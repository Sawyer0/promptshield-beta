import { useCallback, useState } from 'react';
import type { RulesSnapshot } from '../api/contracts';
import { UnauthorizedError, type GatewayClient } from '../api/gatewayClient';
import { usePolledResource } from './usePolledResource';

export interface ActiveRules {
  snapshot: RulesSnapshot | null;
  loadError: string | null;
  reloading: boolean;
  reloadError: string | null;
  reload: () => Promise<void>;
}

/** Active RulePacks, refreshed on rules_reloaded events, plus the reload action. */
export function useActiveRules(client: GatewayClient, rulesVersion: number, onUnauthorized: () => void): ActiveRules {
  const load = useCallback(() => client.getRules(), [client]);
  const { data, error, refresh } = usePolledResource(load, {
    intervalMs: 60_000,
    trigger: rulesVersion,
    onUnauthorized,
    throttleMs: 0,
  });
  const [reloading, setReloading] = useState(false);
  const [reloadError, setReloadError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setReloading(true);
    setReloadError(null);
    try {
      await client.reloadRules();
      refresh();
    } catch (err) {
      if (err instanceof UnauthorizedError) onUnauthorized();
      else setReloadError(err instanceof Error ? err.message : String(err));
    } finally {
      setReloading(false);
    }
  }, [client, refresh, onUnauthorized]);

  return { snapshot: data, loadError: error, reloading, reloadError, reload };
}
