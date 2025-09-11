import { useEffect, useState } from 'react';
import { Layout } from '@/components/Layout';
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useToast } from '@/hooks/use-toast';

export default function Organization() {
  const { toast } = useToast();
  const [orgId, setOrgId] = useState<string | null>(null);
  const [role, setRole] = useState<string | null>(null);
  const [members, setMembers] = useState<Array<any>>([]);
  const [invites, setInvites] = useState<Array<any>>([]);
  const [loading, setLoading] = useState(true);
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState('basic_member');
  const [captcha, setCaptcha] = useState('');

  useEffect(() => {
    (async () => {
      setLoading(true);
      try {
        const s = await fetch('/api/session/tenant', { credentials: 'include' });
        const sj = await s.json();
        setOrgId(sj?.orgId || null);
        setRole(sj?.role || null);
        if (sj?.orgId) {
          const buildAuthHeaders = async (): Promise<HeadersInit> => {
            const h: Record<string,string> = {};
            try { const tok = await (window as any)?.Clerk?.session?.getToken?.(); if (tok) h['Authorization'] = `Bearer ${tok}`; } catch {}
            return h;
          };
          const headers = await buildAuthHeaders();
          const m = await fetch('/api/orgs/members', { credentials: 'include', headers });
          const mj = await m.json();
          setMembers(mj?.data || []);
          if ((sj?.role || '').toLowerCase() === 'admin') {
            const i = await fetch('/api/orgs/invitations', { credentials: 'include', headers });
            const ij = await i.json();
            setInvites(ij?.data || []);
          }
        }
      } catch (e) {
        setMembers([]); setInvites([]);
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const sendInvite = async () => {
    try {
      if (!orgId) { toast({ title: 'Select org', description: 'Choose an organization', variant: 'destructive' }); return; }
      const headers: Record<string,string> = { 'Content-Type': 'application/json', 'x-ps-captcha': captcha };
      try { const tok = await (window as any)?.Clerk?.session?.getToken?.(); if (tok) headers['Authorization'] = `Bearer ${tok}`; } catch {}
      const res = await fetch('/api/orgs/invite', {
        method: 'POST', credentials: 'include', headers,
        body: JSON.stringify({ orgId, email: inviteEmail.trim(), role: inviteRole, captchaToken: captcha })
      });
      if (!res.ok) throw new Error(`${res.status}`);
      toast({ title: 'Invitation sent', description: `${inviteEmail}` });
      setInviteEmail(''); setCaptcha('');
      // reload invites
      const i = await fetch('/api/orgs/invitations', { credentials: 'include', headers });
      const ij = await i.json();
      setInvites(ij?.data || []);
    } catch (e: any) {
      toast({ title: 'Invite failed', description: String(e?.message || e), variant: 'destructive' });
    }
  };

  return (
    <Layout title="Organization" description="Manage members and invitations for your organization">
      {loading ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : !orgId ? (
        <Card><CardContent className="p-6 text-muted-foreground">Select an organization first.</CardContent></Card>
      ) : (
        <div className="grid gap-6 md:grid-cols-2">
          <Card>
            <CardHeader><CardTitle>Members</CardTitle></CardHeader>
            <CardContent>
              {members.length === 0 ? (
                <p className="text-sm text-muted-foreground">No members</p>
              ) : (
                <ul className="space-y-1 text-sm">
                  {members.map((m) => (
                    <li key={m.id} className="flex justify-between"><span>{m.email || m.id}</span><span className="text-muted-foreground">{m.role}</span></li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader><CardTitle>Pending Invitations</CardTitle></CardHeader>
            <CardContent>
              {((role || '').toLowerCase() === 'admin') ? (
                <>
                  {invites.length === 0 ? (
                    <p className="text-sm text-muted-foreground">No pending invites</p>
                  ) : (
                    <ul className="space-y-1 text-sm">
                      {invites.map((i) => (
                        <li key={i.id} className="flex justify-between"><span>{i.email}</span><span className="text-muted-foreground">{i.role}</span></li>
                      ))}
                    </ul>
                  )}
                  <div className="mt-4 space-y-2">
                    <div>
                      <Input type="email" placeholder="teammate@example.com" value={inviteEmail} onChange={e => setInviteEmail(e.target.value)} />
                    </div>
                    <div>
                      <Input placeholder="basic_member|admin" value={inviteRole} onChange={e => setInviteRole(e.target.value)} />
                    </div>
                    <div>
                      <Input placeholder="captcha token" value={captcha} onChange={e => setCaptcha(e.target.value)} />
                    </div>
                    <Button disabled={!inviteEmail.trim()} onClick={sendInvite}>Send Invite</Button>
                    <p className="text-xs text-muted-foreground">Note: Admin-only. Captcha token required; set PS_CAPTCHA_TOKEN on the server. Dev accepts "approved".</p>
                  </div>
                </>
              ) : (
                <p className="text-sm text-muted-foreground">Only admins can view or send invitations.</p>
              )}
            </CardContent>
          </Card>
        </div>
      )}
    </Layout>
  );
}
