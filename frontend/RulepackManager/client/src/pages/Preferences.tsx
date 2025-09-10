import { useEffect, useState } from 'react';
import { Layout } from '@/components/Layout';
import { Card, CardContent } from '@/components/ui/card';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Button } from '@/components/ui/button';
import { useToast } from '@/hooks/use-toast';

type Prefs = {
  tools_allowlist: string[];
  tools_scopes: Record<string, string[]>;
  egress_allowlist: { schemes: string[]; hosts: string[]; paths: string[] };
  timeouts_ms: { default: number; high_risk: number };
  require_approval_tools: string[];
};

export default function Preferences() {
  const { toast } = useToast();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [readOnly, setReadOnly] = useState(false);
  const [orgId, setOrgId] = useState<string | null>(null);
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState('basic_member');
  const [prefs, setPrefs] = useState<Prefs>({
    tools_allowlist: [],
    tools_scopes: {},
    egress_allowlist: { schemes: ['http','https'], hosts: [], paths: [] },
    timeouts_ms: { default: 2000, high_risk: 5000 },
    require_approval_tools: [],
  });

  useEffect(() => {
    const load = async () => {
      setLoading(true);
      try {
        // Load session tenant + org
        try {
          const r = await fetch('/api/session/tenant', { credentials: 'include' });
          const j = await r.json();
          setOrgId(j?.orgId || null);
        } catch {}
        const res = await fetch('/api/admin/settings', { credentials: 'include' });
        if (!res.ok) {
          setReadOnly(true);
          setLoading(false);
          return;
        }
        const data = await res.json();
        const stored = (data?.preferences || {}) as any;
        setPrefs({
          tools_allowlist: stored.tools_allowlist || [],
          tools_scopes: stored.tools_scopes || {},
          egress_allowlist: stored.egress_allowlist || { schemes: ['http','https'], hosts: [], paths: [] },
          timeouts_ms: stored.timeouts_ms || { default: 2000, high_risk: 5000 },
          require_approval_tools: stored.require_approval_tools || [],
        });
      } catch (_) {
        setReadOnly(true);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []);

  const save = async () => {
    setSaving(true);
    try {
      const res = await fetch('/api/admin/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ preferences: prefs }),
      });
      if (!res.ok) throw new Error(await res.text());
      toast({ title: 'Saved', description: 'Preferences updated' });
    } catch (e: any) {
      toast({ title: 'Error', description: String(e?.message || e), variant: 'destructive' });
    } finally {
      setSaving(false);
    }
  };

  const parseList = (s: string) => s.split(/\n|,/).map(x => x.trim()).filter(Boolean);

  const sendInvite = async () => {
    try {
      if (!orgId) {
        toast({ title: 'Select Organization', description: 'Choose an organization before inviting', variant: 'destructive' });
        return;
      }
      const res = await fetch('/api/orgs/invite', {
        method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ orgId, email: inviteEmail.trim(), role: inviteRole })
      });
      if (!res.ok) throw new Error(`${res.status}`);
      toast({ title: 'Invitation sent', description: `${inviteEmail} invited to organization` });
      setInviteEmail('');
    } catch (e: any) {
      toast({ title: 'Invite failed', description: String(e?.message || e), variant: 'destructive' });
    }
  };

  return (
    <Layout title="Preferences" description="Harden agent behavior and tool usage policies.">
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardContent className="p-6 space-y-3">
            <h3 className="font-semibold">Invite Teammates</h3>
            <Label>Email</Label>
            <Input type="email" value={inviteEmail} onChange={e => setInviteEmail(e.target.value)} placeholder="teammate@example.com" />
            <Label>Role</Label>
            <Input value={inviteRole} onChange={e => setInviteRole(e.target.value)} placeholder="basic_member|admin" />
            <div>
              <Button disabled={!orgId || !inviteEmail.trim()} onClick={sendInvite}>Send Invite</Button>
              {!orgId && <p className="text-xs text-muted-foreground mt-2">Select an organization first.</p>}
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardContent className="p-6 space-y-3">
            <h3 className="font-semibold">Tool Allowlist</h3>
            <Label>Allowed tool names (comma or newline separated)</Label>
            <Textarea
              disabled={readOnly}
              rows={5}
              value={prefs.tools_allowlist.join('\n')}
              onChange={e => setPrefs({ ...prefs, tools_allowlist: parseList(e.target.value) })}
            />
            <p className="text-xs text-muted-foreground">Only these tools can be invoked by the agent.</p>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6 space-y-3">
            <h3 className="font-semibold">Network Egress Allowlist</h3>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label>Schemes</Label>
                <Input disabled={readOnly} value={prefs.egress_allowlist.schemes.join(',')} onChange={e => setPrefs({ ...prefs, egress_allowlist: { ...prefs.egress_allowlist, schemes: parseList(e.target.value) } })} />
              </div>
              <div>
                <Label>Hosts</Label>
                <Input disabled={readOnly} value={prefs.egress_allowlist.hosts.join(',')} onChange={e => setPrefs({ ...prefs, egress_allowlist: { ...prefs.egress_allowlist, hosts: parseList(e.target.value) } })} />
              </div>
            </div>
            <Label>Paths (glob patterns)</Label>
            <Input disabled={readOnly} value={prefs.egress_allowlist.paths.join(',')} onChange={e => setPrefs({ ...prefs, egress_allowlist: { ...prefs.egress_allowlist, paths: parseList(e.target.value) } })} />
            <p className="text-xs text-muted-foreground">Restrict LLM tool network calls to known-safe endpoints.</p>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6 space-y-3">
            <h3 className="font-semibold">Timeouts & Budgets</h3>
            <div className="grid grid-cols-2 gap-3">
              <div>
                <Label>Default timeout (ms)</Label>
                <Input type="number" disabled={readOnly} value={prefs.timeouts_ms.default}
                  onChange={e => setPrefs({ ...prefs, timeouts_ms: { ...prefs.timeouts_ms, default: Number(e.target.value) } })} />
              </div>
              <div>
                <Label>High-risk timeout (ms)</Label>
                <Input type="number" disabled={readOnly} value={prefs.timeouts_ms.high_risk}
                  onChange={e => setPrefs({ ...prefs, timeouts_ms: { ...prefs.timeouts_ms, high_risk: Number(e.target.value) } })} />
              </div>
            </div>
            <p className="text-xs text-muted-foreground">Curb long-running tool calls to contain blast radius.</p>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6 space-y-3">
            <h3 className="font-semibold">Approvals (HITL)</h3>
            <Label>Require approval for tools (comma separated)</Label>
            <Input disabled={readOnly} value={prefs.require_approval_tools.join(',')}
              onChange={e => setPrefs({ ...prefs, require_approval_tools: parseList(e.target.value) })} />
            <p className="text-xs text-muted-foreground">Examples: delete_file, transfer_funds, export_data</p>
          </CardContent>
        </Card>
      </div>

      <div className="mt-6 flex justify-end">
        <Button disabled={readOnly || saving} onClick={save}>{saving ? 'Saving…' : 'Save Preferences'}</Button>
      </div>

      {readOnly && (
        <p className="mt-3 text-sm text-muted-foreground">Read‑only: you don’t have permission to change settings.</p>
      )}
    </Layout>
  );
}
