import { useEffect, useMemo, useState } from 'react';
import { useLocation } from 'wouter';
import { SignInModal } from '@/components/auth/SignInModal';
import { SignUpModal } from '@/components/auth/SignUpModal';
import { AccessRequest } from '@/components/auth/AccessRequest';
import { Button } from '@/components/ui/button';
import { Shield } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { isDevBypassClient } from '@/lib/dev';
import { useTenant } from '@/contexts/TenantContext';

export default function AuthModalPage() {
  const [, setLocation] = useLocation();
  const { setTenant } = useTenant();
  const qs = useMemo(() => new URLSearchParams(window.location.search), []);
  const mode = (qs.get('mode') || qs.get('m') || 'signin').toLowerCase();
  const nextPath = (qs.get('next') || '/').toString();
  const invitedOrg = (qs.get('org') || qs.get('orgId') || '').toString();
  // Do not auto-open modals on page load; buttons on this page will open them
  const [openSignIn, setOpenSignIn] = useState(false);
  const [openSignUp, setOpenSignUp] = useState(false);

  // Auto-open correct modal based on mode (brand-only auth)
  useEffect(() => {
    if (mode === 'signin') setOpenSignIn(true);
    if (mode === 'signup') setOpenSignUp(true);
  }, [mode]);

  const closeModal = () => {
    // Stay on this page when closing the modal for a consistent experience
    // We intentionally do not navigate away
  };

  const [phase, setPhase] = useState<'modal'|'select'|'request'>('modal');
  const [discovered, setDiscovered] = useState<Array<{id:string; name:string; autoJoin?: boolean}>>([]);
  const [memberships, setMemberships] = useState<Array<{id:string; name:string}>>([]);
  const [selectOrgId, setSelectOrgId] = useState<string>('');

  return (
    <div className="min-h-screen bg-gradient-to-br from-emerald-50 via-white to-sky-50 dark:from-gray-950 dark:via-gray-925 dark:to-gray-950">
      <div className="mx-auto max-w-6xl px-4 sm:px-6 py-14">
        <div className="grid lg:grid-cols-2 gap-10 items-start">
          <div>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Shield className="h-4 w-4 text-primary" />
              <span className="font-medium">PromptShield</span>
            </div>
            <div className="inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs text-muted-foreground mt-3">
              <span className="w-2 h-2 rounded-full" style={{ backgroundColor: "var(--brand-accent)" }} />
              Secure and compliant AI access
            </div>
            <h1 className="mt-4 text-4xl sm:text-5xl font-medium tracking-tight serif-display">
              Sign in or create your account
            </h1>
            <p className="mt-4 text-lg text-muted-foreground">
              Use your work email. Access is invite-only via your organization.
            </p>
            <div className="mt-6 flex flex-col sm:flex-row gap-3">
              <Button size="lg" className="gap-2" style={{ backgroundColor: "var(--brand-accent)", borderColor: "var(--brand-accent)" }} onClick={() => setOpenSignIn(true)}>
                Sign in
              </Button>
              <Button size="lg" variant="outline" className="gap-2" onClick={() => setOpenSignUp(true)}>
                Create account
              </Button>
            </div>
            <div className="mt-2 text-sm text-muted-foreground">
              <button className="underline" onClick={() => setLocation('/landing')}>Back to landing</button>
            </div>
          </div>

          <div>
            <div className="relative rounded-xl p-6 bg-card border shadow-sm">
              {phase === 'modal' ? (
                <div className="space-y-3 text-sm text-muted-foreground">
                  <div>Click one of the actions to continue. You can switch between Sign in and Create account.</div>
                  <div className="text-xs">After you sign in, we’ll check your organization membership and let you request access if needed.</div>
                </div>
              ) : phase === 'select' ? (
                <div className="space-y-4">
                  <Card>
                    <CardHeader><CardTitle>Select your organization</CardTitle></CardHeader>
                    <CardContent className="space-y-3">
                      <div className="text-sm text-muted-foreground">We found multiple organizations associated with your account. Choose one to continue.</div>
                      <Select value={selectOrgId} onValueChange={setSelectOrgId}>
                        <SelectTrigger><SelectValue placeholder="Select organization" /></SelectTrigger>
                        <SelectContent>
                          {memberships.map(m => (
                            <SelectItem key={m.id} value={m.id}>{m.name}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <div className="flex justify-end">
                        <Button
                          onClick={async () => {
                            if (!selectOrgId) return;
                            try {
                              await fetch('/api/orgs/select', {
                                method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' },
                                body: JSON.stringify({ orgId: selectOrgId })
                              });
                            } catch {}
                            setTenant(selectOrgId, memberships.find(m => m.id === selectOrgId)?.name || selectOrgId);
                            setLocation(nextPath || '/');
                          }}
                        >Continue</Button>
                      </div>
                    </CardContent>
                  </Card>
                </div>
              ) : (
                <div className="space-y-4">
                  <div className="text-sm text-muted-foreground">We found that you are not a member of any organizations yet. Request access to continue.</div>
                  <AccessRequest orgs={discovered} onSubmitted={() => setLocation('/landing')} />
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Modals overlay */}
      {phase === 'modal' && (
        <>
          <SignInModal
            open={openSignIn}
            onOpenChange={(v) => { setOpenSignIn(v); if (!v) closeModal(); }}
            onSuccess={async () => {
              // Ensure BFF recognizes session (include Clerk token if available)
              try {
                const tok = await (window as any)?.Clerk?.session?.getToken?.();
                const headers: Record<string, string> = {};
                if (tok) headers['Authorization'] = `Bearer ${tok}`;
                await fetch('/api/auth/user', { credentials: 'include', headers }).catch(() => {});
              } catch {}
              // After auth, check current memberships
              try {
                const memResp = await fetch('/api/orgs', { credentials: 'include' });
                const memData = await memResp.json();
                const memberships: any[] = memData?.data || [];
                if (Array.isArray(memberships) && memberships.length > 0) {
                  // Prefer invited org via query param
                  if (invitedOrg) {
                    const found = memberships.find(m => m.id === invitedOrg);
                    if (found) {
                      await fetch('/api/orgs/select', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ orgId: invitedOrg }) }).catch(() => {});
                      setTenant(invitedOrg, found.name || invitedOrg);
                      setLocation(nextPath || '/');
                      return;
                    }
                  }
                  if (memberships.length === 1) {
                    const orgId = memberships[0].id;
                    await fetch('/api/orgs/select', {
                      method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' },
                      body: JSON.stringify({ orgId })
                    }).catch(() => {});
                    setTenant(orgId, memberships[0].name || orgId);
                    setLocation(nextPath || '/');
                    return;
                  }
                  // Default select: invitedOrg > last selected > first
                  const mapped = memberships.map(m => ({ id: m.id, name: m.name || m.id }));
                  setMemberships(mapped);
                  const lastId = localStorage.getItem('promptshield_tenant_id') || '';
                  const preferred = (invitedOrg && mapped.find(m => m.id === invitedOrg)?.id)
                    || (lastId && mapped.find(m => m.id === lastId)?.id)
                    || mapped[0].id;
                  setSelectOrgId(preferred);
                  setPhase('select');
                  return;
                }
              } catch {}
              // Discover orgs by domain when user has no memberships
              try {
                const dResp = await fetch('/api/orgs/discover', { credentials: 'include' });
                const d = await dResp.json();
                const list = Array.isArray(d?.data) ? d.data : [];
                setDiscovered(list);
              } catch { setDiscovered([] as any); }
              setPhase('request');
            }}
          />

          <SignUpModal
            open={openSignUp}
            onOpenChange={(v) => { setOpenSignUp(v); if (!v) closeModal(); }}
            onSuccess={async (orgName?: string) => {
              // Ensure auth context
              try {
                const tok = await (window as any)?.Clerk?.session?.getToken?.();
                const headers: Record<string, string> = {};
                if (tok) headers['Authorization'] = `Bearer ${tok}`;
                await fetch('/api/auth/user', { credentials: 'include', headers }).catch(() => {});
              } catch {}

              // If self-serve org name was provided, attempt to create and select it
              try {
                if (orgName && orgName.trim()) {
                  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
                  try { const tok = await (window as any)?.Clerk?.session?.getToken?.(); if (tok) headers['Authorization'] = `Bearer ${tok}`; } catch {}
                  const resp = await fetch('/api/orgs/create', {
                    method: 'POST', headers, credentials: 'include',
                    body: JSON.stringify({ name: orgName.trim() })
                  });
                  if (resp.ok) {
                    const created = await resp.json();
                    const orgId = created?.id || created?.tenant_id;
                    const orgNameResp = created?.name || orgName.trim();
                    if (orgId) {
                      await fetch('/api/orgs/select', {
                        method: 'POST', headers, credentials: 'include',
                        body: JSON.stringify({ orgId })
                      }).catch(() => {});
                      setTenant(orgId, orgNameResp);
                      // After creation, route to app
                      setLocation(nextPath || '/');
                      return;
                    }
                  }
                }
              } catch {}

              // Check existing memberships
              try {
                const memResp = await fetch('/api/orgs', { credentials: 'include' });
                const memData = await memResp.json();
                const memberships: any[] = memData?.data || [];
                if (Array.isArray(memberships) && memberships.length > 0) {
                  if (invitedOrg) {
                    const found = memberships.find(m => m.id === invitedOrg);
                    if (found) {
                      await fetch('/api/orgs/select', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ orgId: invitedOrg }) }).catch(() => {});
                      setTenant(invitedOrg, found.name || invitedOrg);
                      setLocation(nextPath || '/');
                      return;
                    }
                  }
                  if (memberships.length === 1) {
                    const orgId = memberships[0].id;
                    await fetch('/api/orgs/select', {
                      method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' },
                      body: JSON.stringify({ orgId })
                    }).catch(() => {});
                    setTenant(orgId, memberships[0].name || orgId);
                    setLocation(nextPath || '/');
                    return;
                  }
                  const mapped = memberships.map(m => ({ id: m.id, name: m.name || m.id }));
                  setMemberships(mapped);
                  const lastId = localStorage.getItem('promptshield_tenant_id') || '';
                  const preferred = (lastId && mapped.find(m => m.id === lastId)?.id) || mapped[0].id;
                  setSelectOrgId(preferred);
                  setPhase('select');
                  return;
                }
              } catch {}

              // Discover orgs to request access
              try {
                const dResp = await fetch('/api/orgs/discover', { credentials: 'include' });
                const d = await dResp.json();
                const list = Array.isArray(d?.data) ? d.data : [];
                setDiscovered(list);
              } catch { setDiscovered([] as any); }
              setPhase('request');
            }}
          />
        </>
      )}
      {/* Access Request overlay */}
      {phase === 'request' && (
        <div className="w-full max-w-xl">
          <div className="mx-auto mb-4 text-center text-muted-foreground">
            We found that you are not a member of any organizations yet.
          </div>
          <AccessRequest orgs={discovered} onSubmitted={() => setLocation('/landing')} />
        </div>
      )}
    </div>
  );
}
