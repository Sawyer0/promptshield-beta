import { useEffect, useMemo, useState } from 'react';
import { Layout } from '@/components/Layout';
import { PageHeader } from '@/components/PageHeader';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

type Tenant = { id: string; name: string };

type Snapshot = {
  method: string;
  endpoint_template: string;
  rulepack_ids: string[];
  generated_at: string;
};

type MissAgg = {
  method: string;
  template: string;
  misses: number;
  first_seen: string;
  last_seen: string;
};

export default function Snapshots() {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [tenantId, setTenantId] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [snaps, setSnaps] = useState<Snapshot[]>([]);
  const [misses, setMisses] = useState<MissAgg[]>([]);
  const [range, setRange] = useState<{ from: string; to: string }>(() => {
    const to = new Date();
    const from = new Date(Date.now() - 24*60*60*1000);
    return { from: from.toISOString(), to: to.toISOString() };
  });

  useEffect(() => {
    // load tenants (admin only)
    fetch('/v1/admin/tenants', { credentials: 'include' })
      .then(r => r.ok ? r.json() : Promise.reject(new Error(`${r.status} ${r.statusText}`)))
      .then(j => {
        const list = Array.isArray(j.tenants) ? j.tenants : [];
        const t = list.map((x: any) => ({ id: x.id, name: x.name }));
        setTenants(t);
        if (t.length && !tenantId) setTenantId(t[0].id);
      })
      .catch(e => setError(String(e?.message || e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const reload = () => {
    if (!tenantId) return;
    setLoading(true); setError(null);
    Promise.all([
      fetch(`/v1/admin/tenants/${tenantId}/snapshots`, { credentials: 'include' })
        .then(r => r.ok ? r.json() : Promise.reject(new Error(`${r.status} ${r.statusText}`))),
      fetch(`/v1/admin/tenants/${tenantId}/snapshots/misses?from=${encodeURIComponent(range.from)}&to=${encodeURIComponent(range.to)}`, { credentials: 'include' })
        .then(r => r.ok ? r.json() : Promise.reject(new Error(`${r.status} ${r.statusText}`)))
    ]).then(([a, b]) => {
      setSnaps(Array.isArray(a.snapshots) ? a.snapshots : []);
      setMisses(Array.isArray(b.misses) ? b.misses : []);
    }).catch(e => setError(String(e?.message || e))).finally(() => setLoading(false));
  };

  useEffect(() => { reload(); /* eslint-disable-next-line react-hooks/exhaustive-deps */ }, [tenantId]);

  const doRefresh = () => {
    if (!tenantId) return;
    fetch(`/v1/admin/tenants/${tenantId}/snapshots/refresh`, { method: 'POST', credentials: 'include' })
      .then(() => reload())
      .catch(() => reload());
  };

  const sealAudits = () => {
    fetch('/v1/admin/system/audits/seal', { method: 'POST', credentials: 'include' })
      .then(() => {/* no-op */})
      .catch(() => {/* ignore */});
  };

  return (
    <Layout title="Endpoint Snapshots" description="Materialized endpoint→rulepack mappings and misses">
      <div className="container mx-auto px-4 py-6 sm:py-8 space-y-6">
        <PageHeader title="Endpoint Snapshots" subtitle="Admin visibility into snapshot mappings and misses" />

        <Card>
          <CardHeader>
            <CardTitle>Controls</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col sm:flex-row sm:items-end gap-4">
              <div>
                <div className="text-sm font-medium mb-1">Tenant</div>
                <select className="border rounded px-2 py-1" value={tenantId} onChange={e => setTenantId(e.target.value)}>
                  {tenants.map(t => <option key={t.id} value={t.id}>{t.name}</option>)}
                </select>
              </div>
              <div>
                <div className="text-sm font-medium mb-1">From</div>
                <input className="border rounded px-2 py-1 w-64" value={range.from} onChange={e => setRange(s => ({ ...s, from: e.target.value }))} />
              </div>
              <div>
                <div className="text-sm font-medium mb-1">To</div>
                <input className="border rounded px-2 py-1 w-64" value={range.to} onChange={e => setRange(s => ({ ...s, to: e.target.value }))} />
              </div>
              <div className="flex gap-2">
                <button className="bg-emerald-600 hover:bg-emerald-700 text-white px-3 py-1 rounded" onClick={reload} disabled={loading}>Reload</button>
                <button className="bg-indigo-600 hover:bg-indigo-700 text-white px-3 py-1 rounded" onClick={doRefresh}>Rebuild Snapshots</button>
                <button className="bg-slate-600 hover:bg-slate-700 text-white px-3 py-1 rounded" onClick={sealAudits}>Seal Yesterday's Audit Root</button>
              </div>
            </div>
            {error && <div className="text-red-600 mt-3">{error}</div>}
          </CardContent>
        </Card>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <Card>
            <CardHeader><CardTitle>Snapshots</CardTitle></CardHeader>
            <CardContent>
              <div className="overflow-auto">
                <table className="min-w-full text-sm">
                  <thead>
                    <tr className="text-left border-b">
                      <th className="py-2 pr-4">Method</th>
                      <th className="py-2 pr-4">Template</th>
                      <th className="py-2 pr-4">Rulepacks</th>
                      <th className="py-2 pr-4">Generated</th>
                    </tr>
                  </thead>
                  <tbody>
                    {snaps.map((s, i) => (
                      <tr key={i} className="border-b hover:bg-muted/40">
                        <td className="py-2 pr-4 font-mono">{s.method}</td>
                        <td className="py-2 pr-4 font-mono">{s.endpoint_template}</td>
                        <td className="py-2 pr-4 font-mono">{s.rulepack_ids?.join(', ')}</td>
                        <td className="py-2 pr-4">{new Date(s.generated_at).toLocaleString()}</td>
                      </tr>
                    ))}
                    {snaps.length === 0 && (
                      <tr><td className="py-2 text-slate-500" colSpan={4}>No snapshots found for tenant</td></tr>
                    )}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader><CardTitle>Snapshot Misses (Aggregated)</CardTitle></CardHeader>
            <CardContent>
              <div className="overflow-auto">
                <table className="min-w-full text-sm">
                  <thead>
                    <tr className="text-left border-b">
                      <th className="py-2 pr-4">Method</th>
                      <th className="py-2 pr-4">Template</th>
                      <th className="py-2 pr-4">Misses</th>
                      <th className="py-2 pr-4">First</th>
                      <th className="py-2 pr-4">Last</th>
                    </tr>
                  </thead>
                  <tbody>
                    {misses.map((m, i) => (
                      <tr key={i} className="border-b hover:bg-muted/40">
                        <td className="py-2 pr-4 font-mono">{m.method}</td>
                        <td className="py-2 pr-4 font-mono">{m.template}</td>
                        <td className="py-2 pr-4">{m.misses}</td>
                        <td className="py-2 pr-4">{new Date(m.first_seen).toLocaleString()}</td>
                        <td className="py-2 pr-4">{new Date(m.last_seen).toLocaleString()}</td>
                      </tr>
                    ))}
                    {misses.length === 0 && (
                      <tr><td className="py-2 text-slate-500" colSpan={5}>No misses in selected time window</td></tr>
                    )}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </Layout>
  );
}

