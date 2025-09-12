import { useEffect, useMemo, useState } from "react";
import { Layout } from "@/components/Layout";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";

interface MappingControl { name: string; description?: string; status?: 'mapped'|'partial'|'not_mapped'; evidence?: any }
interface Mapping {
  version: string;
  frameworks: Record<string, Record<string, MappingControl>>;
}

interface ExportReport {
  framework: string;
  tenant_id: string | null;
  generated_at: string;
  range: { from: string | null; to: string | null };
  version: string;
  controls: Array<{
    control_id: string;
    name: string;
    description?: string;
    mapping: any;
    audit?: { total: number; sample: Array<any> };
    configs?: Record<string, any>;
    summary: {
      rules?: number;
      rule_tags?: number;
      audits_actions?: number;
      configs?: number;
      reports?: number;
      audit_events_found?: number;
      rules_with_controls?: number;
      rulepacks_with_controls?: number;
    };
  }>;
}

export default function Compliance() {
  const [mapping, setMapping] = useState<Mapping | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [framework, setFramework] = useState<string>("");
  const [from, setFrom] = useState<string>("");
  const [to, setTo] = useState<string>("");
  const [report, setReport] = useState<ExportReport | null>(null);
  const [loadingReport, setLoadingReport] = useState(false);

  useEffect(() => {
    (async () => {
      setLoading(true);
      setError(null);
      try {
        const r = await fetch('/api/compliance/mapping', { credentials: 'include' });
        const j = await r.json();
        if (!r.ok) throw new Error(j?.error?.message || 'Failed to load mapping');
        setMapping(j.data);
        const fw = Object.keys(j?.data?.frameworks || {})[0];
        setFramework(fw || "");
      } catch (e: any) {
        setError(String(e?.message || e));
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const coverage = useMemo(() => {
    if (!mapping || !framework) return { mapped: 0, partial: 0, not_mapped: 0, total: 0 };
    const entries = Object.values(mapping.frameworks[framework] || {});
    const res = { mapped: 0, partial: 0, not_mapped: 0, total: entries.length };
    for (const c of entries) {
      if (c.status === 'mapped') res.mapped++;
      else if (c.status === 'partial') res.partial++;
      else res.not_mapped++;
    }
    return res;
  }, [mapping, framework]);

  const loadEvidence = async () => {
    if (!framework) return;
    setLoadingReport(true);
    setError(null);
    try {
      const qs = new URLSearchParams({ framework, format: 'json' });
      if (from) qs.set('from', from);
      if (to) qs.set('to', to);
      const r = await fetch(`/api/compliance/export?${qs.toString()}`, { credentials: 'include' });
      const j = await r.json();
      if (!r.ok) throw new Error(j?.error?.message || 'Failed to load evidence');
      setReport(j.data);
    } catch (e: any) {
      setError(String(e?.message || e));
    } finally {
      setLoadingReport(false);
    }
  };

  const exportReport = async (fmt: 'json'|'csv') => {
    if (!framework) return;
    const qs = new URLSearchParams({ framework, format: fmt });
    if (from) qs.set('from', from);
    if (to) qs.set('to', to);
    const url = `/api/compliance/export?${qs.toString()}`;
    window.open(url, '_blank');
  };

  const frameworks = Object.keys(mapping?.frameworks || {});
  const controls = framework ? Object.entries(mapping?.frameworks?.[framework] || {}) : [];
  const countsByControl: Record<string, any> = useMemo(() => {
    const out: Record<string, any> = {};
    if (report && Array.isArray(report.controls)) {
      for (const c of report.controls) out[c.control_id] = c.summary || {};
    }
    return out;
  }, [report]);

  return (
    <Layout title="Compliance" description="Control mappings and evidence exports">
      <div className="space-y-6">
        <Card>
          <CardContent className="p-6 space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
              <div>
                <div className="text-sm text-muted-foreground">Mapping Version</div>
                <div className="text-2xl font-semibold">{mapping?.version || '-'}</div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Mapped</div>
                <div className="text-2xl font-semibold text-green-600">{coverage.mapped}</div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Partial</div>
                <div className="text-2xl font-semibold text-amber-600">{coverage.partial}</div>
              </div>
              <div>
                <div className="text-sm text-muted-foreground">Not Mapped</div>
                <div className="text-2xl font-semibold text-slate-600">{coverage.not_mapped}</div>
              </div>
            </div>
            <div className="flex items-end gap-4">
              <div>
                <div className="text-sm font-medium mb-1">Framework</div>
                <select className="border rounded px-3 py-2" value={framework} onChange={e => setFramework(e.target.value)}>
                  {frameworks.map(fw => (
                    <option key={fw} value={fw}>{fw}</option>
                  ))}
                </select>
              </div>
              <div>
                <div className="text-sm font-medium mb-1">From (ISO)</div>
                <input className="border rounded px-3 py-2" placeholder="2025-09-01T00:00:00Z" value={from} onChange={e => setFrom(e.target.value)} />
              </div>
              <div>
                <div className="text-sm font-medium mb-1">To (ISO)</div>
                <input className="border rounded px-3 py-2" placeholder="2025-09-12T00:00:00Z" value={to} onChange={e => setTo(e.target.value)} />
              </div>
              <div className="flex gap-2">
                <Button onClick={loadEvidence} disabled={!framework || loadingReport}>{loadingReport ? 'Loading…' : 'Load Evidence'}</Button>
                <Button variant="outline" onClick={() => exportReport('csv')} disabled={!framework}>Export CSV</Button>
                <Button variant="outline" onClick={() => exportReport('json')} disabled={!framework}>Export JSON</Button>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6 space-y-4">
            <div className="border rounded overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="bg-muted text-left">
                    <th className="p-2 border-b">Control</th>
                    <th className="p-2 border-b">Name</th>
                    <th className="p-2 border-b">Status</th>
                    <th className="p-2 border-b">Rules w/ Controls</th>
                    <th className="p-2 border-b">RulePacks w/ Controls</th>
                    <th className="p-2 border-b">Audit Events</th>
                    <th className="p-2 border-b">Evidence (summary)</th>
                  </tr>
                </thead>
                <tbody>
                  {controls.map(([cid, c]) => {
                    const mc = c as MappingControl;
                    const ev = mc?.evidence || {};
                    const summary = [
                      ev.rules?.length ? `${ev.rules.length} rules` : null,
                      ev.rule_tags?.length ? `${ev.rule_tags.length} tags` : null,
                      ev.audits?.actions?.length ? `${ev.audits.actions.length} audit actions` : null,
                      ev.configs?.length ? `${ev.configs.length} configs` : null,
                    ].filter(Boolean).join(' • ');
                    const counts = countsByControl[cid] || {};
                    const statusBadge = (
                      <span className={
                        "inline-flex items-center px-2 py-0.5 rounded text-xs font-medium " +
                        (mc.status === 'mapped' ? 'bg-green-100 text-green-800' : mc.status === 'partial' ? 'bg-amber-100 text-amber-800' : 'bg-slate-100 text-slate-800')
                      }>
                        {mc.status || 'not_mapped'}
                      </span>
                    );
                    return (
                      <tr key={cid as string} className="hover:bg-muted/50">
                        <td className="p-2 border-b font-mono text-xs">{cid}</td>
                        <td className="p-2 border-b">{mc.name}</td>
                        <td className="p-2 border-b">{statusBadge}</td>
                        <td className="p-2 border-b">{counts.rules_with_controls ?? '-'}</td>
                        <td className="p-2 border-b">{counts.rulepacks_with_controls ?? '-'}</td>
                        <td className="p-2 border-b">{counts.audit_events_found ?? '-'}</td>
                        <td className="p-2 border-b text-muted-foreground">{summary || '-'}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>

            {error && <div className="text-red-600 text-sm">{error}</div>}
            {loading && <div className="text-sm text-muted-foreground">Loading…</div>}
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}

