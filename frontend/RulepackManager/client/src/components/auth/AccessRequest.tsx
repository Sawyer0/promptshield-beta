import { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';

interface OrgOption { id: string; name: string; autoJoin?: boolean }

interface Props {
  orgs: OrgOption[];
  onSubmitted?: () => void;
}

export function AccessRequest({ orgs, onSubmitted }: Props) {
  const [desiredRole, setDesiredRole] = useState<string>('member');
  const [note, setNote] = useState<string>('');
  const [orgId, setOrgId] = useState<string>(orgs?.[0]?.id || '');
  const [submitting, setSubmitting] = useState(false);
  const hasOrgs = Array.isArray(orgs) && orgs.length > 0;

  const submit = async () => {
    try {
      setSubmitting(true);
      const body: any = { desiredRole, note };
      if (orgId) body.orgId = orgId;
      const r = await fetch('/api/onboarding/request-access', {
        method: 'POST', credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      });
      if (r.ok) {
        alert('Request sent. A tenant admin will invite you by email.');
        onSubmitted?.();
      } else {
        alert('Failed to send request. Please contact your administrator.');
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Card className="w-full max-w-md">
      <CardHeader>
        <CardTitle>Request Access</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {hasOrgs ? (
          <div>
            <div className="text-sm text-muted-foreground mb-2">Organizations matching your email domain</div>
            <Select value={orgId} onValueChange={setOrgId}>
              <SelectTrigger>
                <SelectValue placeholder="Select an organization" />
              </SelectTrigger>
              <SelectContent>
                {orgs.map(o => (
                  <SelectItem key={o.id} value={o.id}>{o.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        ) : (
          <div className="text-sm text-muted-foreground">No matching organizations found. Submit a request and an administrator will review it.</div>
        )}

        <div>
          <div className="text-sm font-medium mb-1">Desired role</div>
          <Select value={desiredRole} onValueChange={setDesiredRole}>
            <SelectTrigger>
              <SelectValue placeholder="Select role" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="member">Member</SelectItem>
              <SelectItem value="developer">Developer</SelectItem>
              <SelectItem value="security_engineer">Security Engineer</SelectItem>
              <SelectItem value="tenant_admin">Tenant Admin</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div>
          <div className="text-sm font-medium mb-1">Note (optional)</div>
          <Textarea value={note} onChange={(e) => setNote(e.target.value)} placeholder="Add context for the admin (e.g., team, project)" />
        </div>

        <div className="flex justify-end">
          <Button onClick={submit} disabled={submitting}>{submitting ? 'Submitting…' : 'Request access'}</Button>
        </div>
      </CardContent>
    </Card>
  );
}
