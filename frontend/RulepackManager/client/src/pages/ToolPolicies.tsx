import { useEffect, useState } from 'react';
import { Layout } from '@/components/Layout';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { useToast } from '@/hooks/use-toast';
import { toolPoliciesApi } from '@/lib/api';

type Policy = {
  scope: string;
  methods?: string[];
  allowed_tools?: string[];
  require_approval?: string[];
  timeout_ms?: number;
  egress_allowlist?: { schemes?: string[]; hosts?: string[]; paths?: string[] };
  require_roles?: string[];
  require_headers?: Record<string, string | string[]>;
};

export default function ToolPolicies() {
  const { toast } = useToast();
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const res = await toolPoliciesApi.get();
        setPolicies((res?.policies as any[]) || []);
      } catch (e: any) {
        toast({ title: 'Error', description: String(e?.message || e), variant: 'destructive' });
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const addRow = () => setPolicies((p) => ([...p, { scope: '', methods: ['POST'], allowed_tools: [], require_approval: [], egress_allowlist: { schemes: ['https'] } }]));
  const removeRow = (i: number) => setPolicies((p) => p.filter((_, idx) => idx !== i));

  const save = async () => {
    setSaving(true);
    try {
      await toolPoliciesApi.save({ policies });
      toast({ title: 'Saved', description: 'Tool policies updated' });
    } catch (e: any) {
      toast({ title: 'Error', description: String(e?.message || e), variant: 'destructive' });
    } finally {
      setSaving(false);
    }
  };

  const flush = async () => {
    try {
      await toolPoliciesApi.flushCaches();
      toast({ title: 'Flushed', description: 'Cluster caches invalidated' });
    } catch (e: any) {
      toast({ title: 'Error', description: String(e?.message || e), variant: 'destructive' });
    }
  };
  const flushTenant = async (tenantId: string) => {
    try {
      const res = await fetch('/api/admin/tool-policies/flush', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ tenant_id: tenantId }) });
      if (!res.ok) throw new Error(`${res.status}`);
      toast({ title: 'Flushed', description: `Tenant ${tenantId} cache invalidated` });
    } catch (e: any) {
      toast({ title: 'Error', description: String(e?.message || e), variant: 'destructive' });
    }
  };

  const parseList = (s: string) => s.split(/\n|,/).map(x => x.trim()).filter(Boolean);
  const parseHeaders = (s: string): Record<string, string> => {
    const out: Record<string, string> = {};
    s.split(/\n/).forEach(line => {
      const t = line.trim(); if (!t) return;
      const idx = t.indexOf(':');
      if (idx === -1) { out[t] = ''; return; }
      const k = t.slice(0, idx).trim(); const v = t.slice(idx+1).trim();
      if (k) out[k] = v;
    });
    return out;
  };
  const headersToString = (h?: Record<string, any>) => {
    if (!h) return '';
    const lines: string[] = [];
    Object.entries(h).forEach(([k, v]) => {
      if (Array.isArray(v)) {
        if (v.length === 0) lines.push(`${k}:`);
        else v.forEach(val => lines.push(`${k}: ${val}`));
      } else {
        lines.push(`${k}:${v ? ' ' + v : ''}`);
      }
    });
    return lines.join('\n');
  };

  if (loading) {
    return <Layout title="Tool Policies" description="Define endpoint-scoped tool permissions"><p className="text-muted-foreground">Loading…</p></Layout>;
  }

  return (
    <Layout title="Tool Policies" description="Define which tools are allowed per endpoint, with optional approvals and egress rules.">
      <div className="flex justify-between items-center mb-4">
        <div>
          <h2 className="text-xl font-semibold">Endpoint-Scoped Policies</h2>
          <p className="text-sm text-muted-foreground">Policies change infrequently; manage them centrally here.</p>
        </div>
        <div className="space-x-2">
          <Button variant="outline" onClick={addRow}>Add Policy</Button>
          <Button onClick={save} disabled={saving}>{saving ? 'Saving…' : 'Save'}</Button>
          <Button variant="outline" onClick={flush}>Flush Caches</Button>
        </div>
      </div>
      <div className="mb-4 text-xs text-muted-foreground">
        Tip: To flush a single tenant, use the API with body {`{"tenant_id":"<uuid>"}`}.
      </div>

      {policies.length === 0 ? (
        <Card><CardContent className="p-6 text-muted-foreground">No policies defined. Click “Add Policy”.</CardContent></Card>
      ) : (
        <div className="space-y-4">
          {policies.map((p, i) => (
            <Card key={i} className="border-primary/20">
              <CardContent className="p-4 grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium">Scope (path or prefix*)</label>
                  <Input value={p.scope} onChange={e => update(i, { scope: e.target.value })} placeholder="/v1/chat/*" />
                </div>
                <div>
                  <label className="text-sm font-medium">Methods (comma)</label>
                  <Input value={(p.methods || []).join(',')} onChange={e => update(i, { methods: parseList(e.target.value) })} placeholder="POST" />
                </div>
                <div>
                  <label className="text-sm font-medium">Allowed Tools (comma or newline)</label>
                  <Textarea rows={3} value={(p.allowed_tools || []).join('\n')} onChange={e => update(i, { allowed_tools: parseList(e.target.value) })} />
                </div>
                <div>
                  <label className="text-sm font-medium">Require Approval (tools)</label>
                  <Input value={(p.require_approval || []).join(',')} onChange={e => update(i, { require_approval: parseList(e.target.value) })} placeholder="delete_file, transfer_funds" />
                </div>
                <div>
                  <label className="text-sm font-medium">Timeout (ms)</label>
                  <Input type="number" value={p.timeout_ms || 0} onChange={e => update(i, { timeout_ms: Number(e.target.value) })} />
                </div>
                <div>
                  <label className="text-sm font-medium">Require Roles (comma)</label>
                  <Input value={(p.require_roles || []).join(',')} onChange={e => update(i, { require_roles: parseList(e.target.value) })} placeholder="analyst,admin" />
                </div>
                <div className="md:col-span-2">
                  <label className="text-sm font-medium">Required Headers (one per line as Name:Value; empty value means presence-only)</label>
                  <Textarea rows={3}
                    value={headersToString(p.require_headers as any)}
                    onChange={e => {
                      const obj = parseHeaders(e.target.value);
                      // store as map of name -> value (string) for simplicity
                      update(i, { require_headers: obj as any });
                    }}
                  />
                </div>
                <div>
                  <label className="text-sm font-medium">Egress Schemes</label>
                  <Input value={(p.egress_allowlist?.schemes || []).join(',')} onChange={e => update(i, { egress_allowlist: { ...p.egress_allowlist, schemes: parseList(e.target.value) } })} placeholder="https" />
                </div>
                <div>
                  <label className="text-sm font-medium">Egress Hosts</label>
                  <Input value={(p.egress_allowlist?.hosts || []).join(',')} onChange={e => update(i, { egress_allowlist: { ...p.egress_allowlist, hosts: parseList(e.target.value) } })} placeholder="api.company.com, *.corp.net" />
                </div>
                <div className="md:col-span-2">
                  <label className="text-sm font-medium">Egress Paths (comma)</label>
                  <Input value={(p.egress_allowlist?.paths || []).join(',')} onChange={e => update(i, { egress_allowlist: { ...p.egress_allowlist, paths: parseList(e.target.value) } })} placeholder="/docs/*, /api/public/*" />
                </div>
                <div className="md:col-span-2 flex justify-end">
                  <Button variant="ghost" onClick={() => removeRow(i)}>Remove</Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </Layout>
  );

  function update(idx: number, patch: Partial<Policy>) {
    setPolicies(prev => prev.map((row, i) => i === idx ? { ...row, ...patch } : row));
  }
}

function hasRole(name: string) {
  try { const raw = localStorage.getItem('ps_roles'); const r = raw ? JSON.parse(raw) : []; return Array.isArray(r) && r.includes(name); } catch { return false; }
}
function canEdit() {
  return hasRole('tenant_admin') || hasRole('security_engineer');
}
