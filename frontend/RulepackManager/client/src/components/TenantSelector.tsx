import { useState, useEffect, useCallback } from 'react';
import { useTenant } from '@/contexts/TenantContext';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Building, Plus } from 'lucide-react';
import { tenantApi } from '@/lib/api';
// import type { Tenant } from '@shared/apiTypes';

export function TenantSelector() {
  const { tenantId, tenantName, setTenant } = useTenant();
  const [availableTenants, setAvailableTenants] = useState<Array<{id: string; name: string}>>([]);
  const [loading, setLoading] = useState(true);
  const [createOpen, setCreateOpen] = useState(false);
  const [redirecting, setRedirecting] = useState(false);
  const [newTenantName, setNewTenantName] = useState('');

  useEffect(() => {
    const load = async () => {
      try {
        let headers: Record<string,string> = {};
        try { const tok = await (window as any)?.Clerk?.session?.getToken?.(); if (tok) headers['Authorization'] = `Bearer ${tok}`; } catch {}
        const resp = await fetch('/api/orgs', { credentials: 'include', headers });
        const data = await resp.json();
        const list = (data?.data || []) as Array<{id: string; name: string}>;
        setAvailableTenants(list);
        if (list.length === 0) {
          setCreateOpen(true);
        } else if (list.length === 1) {
          // Auto-select the only tenant and redirect
          setRedirecting(true);
          try {
            const only = list[0];
            const headers: Record<string,string> = { 'Content-Type': 'application/json' };
            try { const tok = await (window as any)?.Clerk?.session?.getToken?.(); if (tok) headers['Authorization'] = `Bearer ${tok}`; } catch {}
            await fetch('/api/orgs/select', {
              method: 'POST', headers, credentials: 'include',
              body: JSON.stringify({ orgId: only.id })
            });
            setTenant(only.id, only.name || '');
            // Stay on current route; Router will render content once tenant is set
            return;
          } catch (_) {
            setRedirecting(false);
          }
        }
      } catch (_) {
        setAvailableTenants([]);
      } finally {
        setLoading(false);
      }
    };
    load();
  }, []);

  const handleTenantChange = useCallback(async (selectedOrgId: string) => {
    const selected = availableTenants.find(t => t.id === selectedOrgId);
    if (selected) {
      const headers: Record<string,string> = { 'Content-Type': 'application/json' };
      try { const tok = await (window as any)?.Clerk?.session?.getToken?.(); if (tok) headers['Authorization'] = `Bearer ${tok}`; } catch {}
      await fetch('/api/orgs/select', {
        method: 'POST', headers, credentials: 'include',
        body: JSON.stringify({ orgId: selected.id })
      });
      setTenant(selected.id, selected.name);
      // Stay on current route; Router will re-render
    }
  }, [availableTenants, setTenant]);

  if (loading || redirecting) {
    return (
      <div className="flex items-center space-x-2 text-sm">
        <Building className="h-4 w-4 text-muted-foreground animate-pulse" />
        <span className="text-muted-foreground">
          {redirecting ? 'Taking you to your organization…' : 'Loading tenants...'}
        </span>
      </div>
    );
  }

  if (tenantId && tenantName) {
    // Regular users see their tenant without ability to switch
    return (
      <div className="flex items-center space-x-2 text-sm">
        <Building className="h-4 w-4 text-muted-foreground" />
        <span className="font-medium">{tenantName}</span>
      </div>
    );
  }

  return (
    <Card className="w-full max-w-md mx-auto">
      <CardHeader>
        <CardTitle className="flex items-center">
          <Building className="mr-2 h-5 w-5" />
          Select Tenant
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground">
          Please select a tenant to continue with PromptShield. All requests will include X-Tenant-ID header for proper data isolation.
        </p>
        
        <Select onValueChange={handleTenantChange}>
          <SelectTrigger>
            <SelectValue placeholder="Choose your tenant..." />
          </SelectTrigger>
          <SelectContent>
            {availableTenants.map(tenant => (
              <SelectItem key={tenant.id} value={tenant.id}>
                {tenant.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        
        <div className="text-center">
          <Button variant="ghost" size="sm" className="text-xs" onClick={() => setCreateOpen(true)}>
            <Plus className="mr-1 h-3 w-3" />
            Request New Tenant
          </Button>
        </div>
      </CardContent>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create New Tenant</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <Input
              placeholder="Organization name"
              value={newTenantName}
              onChange={(e) => setNewTenantName(e.target.value)}
            />
            <div className="flex justify-end gap-2">
              <Button variant="ghost" onClick={() => setCreateOpen(false)}>Cancel</Button>
              <Button
                onClick={async () => {
                  if (!newTenantName.trim()) return;
              try {
                const resp = await fetch('/api/orgs/create', {
                  method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'include',
                  body: JSON.stringify({ name: newTenantName.trim() })
                });
                if (resp.ok) {
                  const org = await resp.json();
                  await fetch('/api/orgs/select', {
                    method: 'POST', headers: { 'Content-Type': 'application/json' }, credentials: 'include',
                    body: JSON.stringify({ orgId: org.id })
                  });
                  setTenant(org.id, org.name);
                  // Stay on current route
                }
              } catch (e) {
                // Optional: toast error
              }
                }}
              >
                Create and Continue
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </Card>
  );
}
