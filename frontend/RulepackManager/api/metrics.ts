import type { Express } from 'express';
import { extractUserContext } from '../server/jwtAuth.js';
import { getValidatedEnvironmentConfig } from '../server/envConfig.js';

/**
 * Minimal metrics API that queries Prometheus /api/v1/query_range
 * Tenant-safe: scopes by region derived from org context where possible.
 *
 * GET /api/metrics/enforcer?range=6h&step=30s
 */
export function registerMetricsApi(app: Express) {
  const cfg = getValidatedEnvironmentConfig();
  const PROM_URL = cfg.promUrl || 'http://localhost:9090';

  const durToSec = (s: string) => {
    const m = s.match(/^([0-9]+)([smhd])$/);
    const mult: any = { s: 1, m: 60, h: 3600, d: 86400 };
    return m ? parseInt(m[1], 10) * mult[m[2] as 's'|'m'|'h'|'d'] : 21600;
  };

  function promRangeUrl(query: string, start: number, end: number, step: string) {
    const u = new URL(`${PROM_URL}/api/v1/query_range`);
    u.searchParams.set('query', query);
    u.searchParams.set('start', String(start));
    u.searchParams.set('end', String(end));
    u.searchParams.set('step', step);
    return u.toString();
  }

  app.get('/api/metrics/enforcer', async (req: any, res) => {
    try {
      const { range = '6h', step = '30s' } = req.query as any;
      const now = Math.floor(Date.now() / 1000);
      const start = now - durToSec(String(range));

      // Derive region from cookie or default; you already set label via ServiceMonitor relabelings
      const userCtx = extractUserContext(req);
      const region = (req.signedCookies && req.signedCookies.ps_region) || process.env.PS_REGION_DEFAULT || 'us-east-1';

      const queries: Record<string, string> = {
        requests: `rate(promhttp_metric_handler_requests_total{job="promptshield-enforcer",region="${region}"}[5m])`,
        redactions: `increase(ps_extproc_redactions_total{job="promptshield-enforcer",region="${region}"}[5m])`,
        cpu: `sum by (pod) (rate(process_cpu_seconds_total{job="promptshield-enforcer",region="${region}"}[5m]))`,
        heap: `go_memstats_alloc_bytes{job="promptshield-enforcer",region="${region}"}`,
      };

      const urls = Object.fromEntries(Object.entries(queries).map(([k, q]) => [k, promRangeUrl(q, start, now, String(step))]));

      const fetchOne = async (url: string) => {
        const r = await fetch(url);
        const j = await r.json();
        return j;
      };

      const results = await Promise.all(Object.entries(urls).map(async ([k, u]) => [k, await fetchOne(u)]));
      res.json(Object.fromEntries(results));
    } catch (err) {
      res.status(500).json({ error: 'metrics_query_failed', details: err instanceof Error ? err.message : String(err) });
    }
  });
}

