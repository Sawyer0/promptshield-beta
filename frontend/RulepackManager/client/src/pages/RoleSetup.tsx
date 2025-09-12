import { useEffect, useState } from 'react';
import { Shield, Users, Wrench, Eye, ClipboardList, Activity, LineChart, Server, Building } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Checkbox } from '@/components/ui/checkbox';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import { Label } from '@/components/ui/label';

type RoleKey = 'platform_admin'|'tenant_admin'|'security_engineer'|'developer'|'auditor'|'compliance_officer';
type UseCaseKey = 'policy_enforcement'|'monitoring'|'compliance_reporting'|'billing_analytics';
type DeployKey = 'cloud'|'self_hosted'|'air_gapped';

export default function RoleSetup() {
  const [saving, setSaving] = useState(false);
  const [role, setRole] = useState<RoleKey | null>(null);
  const [useCase, setUseCase] = useState<UseCaseKey | null>(null);
  const [deploy, setDeploy] = useState<DeployKey>('cloud');
  const [industry, setIndustry] = useState<string>('');
  const [frameworks, setFrameworks] = useState<string[]>([]);
  const [environment, setEnvironment] = useState<'evaluation'|'production'>('evaluation');

  useEffect(() => {
    // If roles already set, skip (we still render so user can change)
    try { const raw = localStorage.getItem('ps_roles'); if (raw) {/*noop*/} } catch {}
  }, []);

  const cards = {
    roles: [
      { key: 'platform_admin', title: 'Platform Owner', desc: 'Operate PromptShield as a provider (multi-tenant, billing, ops).', icon: Shield },
      { key: 'tenant_admin', title: 'Security Admin', desc: 'Owns policies, org settings, and access for a tenant.', icon: Users },
      { key: 'security_engineer', title: 'Security Engineer', desc: 'Builds rulepacks, tools, and enforcement policies.', icon: Wrench },
      { key: 'developer', title: 'Developer', desc: 'Integrates SDKs and tests policies in applications.', icon: Server },
      { key: 'auditor', title: 'Auditor', desc: 'Read-only reporting and evidence review.', icon: Eye },
      { key: 'compliance_officer', title: 'Compliance Officer', desc: 'Manages SOC2/GDPR reporting and evidence.', icon: ClipboardList },
    ] as Array<{ key: RoleKey; title: string; desc: string; icon: any }>,
    useCases: [
      { key: 'policy_enforcement', title: 'Policy Enforcement', desc: 'Tool policies, approvals, and egress rules.', icon: Shield },
      { key: 'monitoring', title: 'Monitoring', desc: 'Live events, metrics, and service health.', icon: Activity },
      { key: 'compliance_reporting', title: 'Compliance Reporting', desc: 'Controls, mapping, and exports.', icon: ClipboardList },
      { key: 'billing_analytics', title: 'Usage Analytics', desc: 'Usage, cost, and allocation insights.', icon: LineChart },
    ] as Array<{ key: UseCaseKey; title: string; desc: string; icon: any }>,
    deploy: [
      { key: 'cloud', title: 'Cloud', desc: 'Managed or self-managed in cloud', icon: Building },
      { key: 'self_hosted', title: 'Self-Hosted', desc: 'Your Kubernetes or VMs', icon: Server },
      { key: 'air_gapped', title: 'Air-gapped', desc: 'Offline and isolated environments', icon: Shield },
    ] as Array<{ key: DeployKey; title: string; desc: string; icon: any }>,
  };

  function roleToRoles(k: RoleKey): string[] {
    if (k === 'platform_admin') return ['platform_admin'];
    if (k === 'tenant_admin') return ['tenant_admin'];
    if (k === 'security_engineer') return ['security_engineer'];
    if (k === 'developer') return ['developer'];
    if (k === 'auditor') return ['auditor'];
    if (k === 'compliance_officer') return ['auditor']; // maps to read-only in UI
    return [];
  }

  function computeDestination(r: RoleKey, u?: UseCaseKey | null): string {
    if (r === 'platform_admin') return '/platform';
    // tenant-scoped views
    if (u === 'policy_enforcement') return '/tool-policies';
    if (u === 'monitoring') return '/monitoring/enforcer';
    if (u === 'compliance_reporting') return '/compliance';
    if (u === 'billing_analytics') return '/analytics/usage';
    if (r === 'tenant_admin') return '/organization';
    if (r === 'security_engineer') return '/rulepacks';
    if (r === 'developer') return '/dashboard';
    if (r === 'auditor' || r === 'compliance_officer') return '/compliance';
    return '/dashboard';
  }

  async function onContinue() {
    if (!role) return;
    setSaving(true);
    try {
      const roles = roleToRoles(role);
      localStorage.setItem('ps_roles', JSON.stringify(roles));
      if (roles.includes('platform_admin')) localStorage.setItem('user_system_role', 'admin');
      // Log extended onboarding context (best-effort)
      try {
        await fetch('/api/onboarding/role', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ selection: role, roles, useCase, deploy, industry, frameworks, environment }) });
      } catch {}
      // Set a one-time onboarding nudge for first landing page
      try {
        const variant = Math.random() < 0.5 ? 'A' : 'B';
        const dest = computeDestination(role, useCase);
        localStorage.setItem('ps_onboarding_nudge', JSON.stringify({ forPath: dest, variant, dismissed: false, context: { role, useCase } }));
      } catch {}
      const hasTenant = !!localStorage.getItem('promptshield_tenant_id');
      const dest = computeDestination(role, useCase);
      if (role !== 'platform_admin' && !hasTenant) {
        window.location.href = '/choose-tenant';
      } else {
        window.location.href = dest;
      }
    } finally {
      setSaving(false);
    }
  }

  const Section = ({ title, children }: { title: string; children: any }) => (
    <div>
      <div className="text-xs font-bold text-muted-foreground uppercase tracking-wider mb-2">{title}</div>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">{children}</div>
    </div>
  );

  const SelectCard = ({ selected, onClick, title, desc, Icon }: { selected: boolean; onClick: () => void; title: string; desc: string; Icon: any }) => (
    <button
      type="button"
      onClick={onClick}
      className={
        'text-left border rounded-xl p-4 hover:shadow-md transition-all bg-white dark:bg-slate-900 ' +
        (selected ? 'border-emerald-500 ring-2 ring-emerald-200 dark:ring-emerald-900' : 'border-slate-200 dark:border-slate-800')
      }
    >
      <div className="flex items-start gap-3">
        <div className={'p-2 rounded-lg ' + (selected ? 'bg-emerald-500/20' : 'bg-slate-100 dark:bg-slate-800')}>
          <Icon className="h-5 w-5" />
        </div>
        <div>
          <div className="font-medium">{title}</div>
          <div className="text-xs text-muted-foreground mt-1">{desc}</div>
        </div>
      </div>
    </button>
  );

  return (
    <div className="min-h-screen bg-background flex items-center justify-center p-6">
      <div className="w-full max-w-4xl mx-auto">
        <div className="flex justify-center mb-6">
          <Shield className="h-10 w-10 text-primary" />
        </div>
        <div className="text-center mb-6">
          <h1 className="text-2xl font-bold">Welcome. Let’s tailor your experience</h1>
          <p className="text-sm text-muted-foreground">Select your role and focus so we can route you to the right workspace.</p>
        </div>
        <div className="space-y-6">
          <Section title="Who are you?">
            {cards.roles.map((r) => (
              <SelectCard
                key={r.key}
                selected={role === (r.key as RoleKey)}
                onClick={() => setRole(r.key as RoleKey)}
                title={r.title}
                desc={r.desc}
                Icon={r.icon}
              />
            ))}
          </Section>
          <Section title="Primary focus">
            {cards.useCases.map((u) => (
              <SelectCard
                key={u.key}
                selected={useCase === (u.key as UseCaseKey)}
                onClick={() => setUseCase(u.key as UseCaseKey)}
                title={u.title}
                desc={u.desc}
                Icon={u.icon}
              />
            ))}
          </Section>
          <Section title="Deployment preference">
            {cards.deploy.map((d) => (
              <SelectCard
                key={d.key}
                selected={deploy === (d.key as DeployKey)}
                onClick={() => setDeploy(d.key as DeployKey)}
                title={d.title}
                desc={d.desc}
                Icon={d.icon}
              />
            ))}
          </Section>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <Card className="p-4">
              <div className="text-xs font-bold text-muted-foreground uppercase tracking-wider mb-2">Organization context</div>
              <div className="space-y-3">
                <div>
                  <Label className="text-xs">Industry</Label>
                  <Input placeholder="e.g., Finance" value={industry} onChange={(e) => setIndustry(e.target.value)} />
                </div>
                <div>
                  <Label className="text-xs">Operating environment</Label>
                  <RadioGroup value={environment} onValueChange={(v: any) => setEnvironment(v)} className="mt-1 flex gap-4">
                    <div className="flex items-center space-x-2">
                      <RadioGroupItem id="env-eval" value="evaluation" />
                      <Label htmlFor="env-eval">Evaluation</Label>
                    </div>
                    <div className="flex items-center space-x-2">
                      <RadioGroupItem id="env-prod" value="production" />
                      <Label htmlFor="env-prod">Production</Label>
                    </div>
                  </RadioGroup>
                </div>
              </div>
            </Card>
            <Card className="p-4">
              <div className="text-xs font-bold text-muted-foreground uppercase tracking-wider mb-2">Compliance frameworks</div>
              <div className="grid grid-cols-2 gap-2">
                {['SOC2','GDPR','ISO 27001','HIPAA'].map((fw) => (
                  <label key={fw} className="flex items-center space-x-2">
                    <Checkbox
                      checked={frameworks.includes(fw)}
                      onCheckedChange={(checked: any) => {
                        setFrameworks((prev) => checked ? Array.from(new Set([...prev, fw])) : prev.filter(x => x !== fw));
                      }}
                    />
                    <span className="text-sm">{fw}</span>
                  </label>
                ))}
              </div>
            </Card>
          </div>
          <div className="flex items-center justify-between gap-2">
            <button
              type="button"
              className="text-xs text-muted-foreground hover:underline"
              onClick={async () => {
                try {
                  setSaving(true);
                  const desiredRole = 'member';
                  const note = 'Requesting access to any tenant';
                  await fetch('/api/onboarding/request-access', { method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ desiredRole, note }) });
                  alert('Request sent to tenant admins. You will receive an invitation when approved.');
                } catch {
                  alert('Failed to send request. Please contact your administrator.');
                } finally {
                  setSaving(false);
                }
              }}
            >Need tenant access? Request access</button>
            <Button variant="ghost" onClick={() => { window.location.href = '/landing'; }}>Back</Button>
            <Button disabled={!role || saving} onClick={onContinue}>{saving ? 'Saving…' : 'Continue'}</Button>
          </div>
          <div className="text-xs text-muted-foreground text-center">You can change this later in Preferences.</div>
        </div>
      </div>
    </div>
  );
}
