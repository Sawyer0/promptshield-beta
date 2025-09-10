import { useEffect, useState } from 'react';
import { Layout } from '@/components/Layout';
import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useToast } from '@/hooks/use-toast';
import { providerProfilesApi } from '@/lib/api';

export default function ProviderProfiles() {
  const { toast } = useToast();
  const [items, setItems] = useState<Array<any>>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState<{ provider: string; label: string; apiKey: string; baseUrl?: string }>({ provider: 'openai', label: '', apiKey: '' });

  const load = async () => {
    setLoading(true);
    try {
      const { data } = await providerProfilesApi.list();
      setItems(data || []);
    } catch (e: any) {
      toast({ title: 'Error', description: String(e?.message || e), variant: 'destructive' });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  const create = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.label || !form.apiKey) {
      toast({ title: 'Missing fields', description: 'Label and API key are required', variant: 'destructive' });
      return;
    }
    setSaving(true);
    try {
      await providerProfilesApi.create(form);
      setForm({ provider: 'openai', label: '', apiKey: '' });
      await load();
      toast({ title: 'Saved', description: 'Provider profile created' });
    } catch (e: any) {
      toast({ title: 'Error', description: String(e?.message || e), variant: 'destructive' });
    } finally {
      setSaving(false);
    }
  };

  const remove = async (id: string) => {
    if (!confirm('Delete this provider profile?')) return;
    try {
      await providerProfilesApi.remove(id);
      await load();
      toast({ title: 'Deleted' });
    } catch (e: any) {
      toast({ title: 'Error', description: String(e?.message || e), variant: 'destructive' });
    }
  };

  return (
    <Layout title="Provider Profiles" description="Manage BYOK provider API keys and endpoints">
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardContent className="p-6 space-y-4">
            <h3 className="font-semibold">Add Provider Profile (BYOK)</h3>
            <form onSubmit={create} className="space-y-3">
              <div>
                <Label>Provider</Label>
                <select className="w-full border rounded px-2 py-1" value={form.provider} onChange={e => setForm({ ...form, provider: e.target.value })}>
                  <option value="openai">OpenAI</option>
                  <option value="anthropic">Anthropic</option>
                  <option value="azure-openai">Azure OpenAI</option>
                  <option value="other">Other</option>
                </select>
              </div>
              <div>
                <Label>Label</Label>
                <Input value={form.label} onChange={e => setForm({ ...form, label: e.target.value })} placeholder="My OpenAI Key" />
              </div>
              <div>
                <Label>API Key</Label>
                <Input type="password" value={form.apiKey} onChange={e => setForm({ ...form, apiKey: e.target.value })} placeholder="sk-..." />
              </div>
              <div>
                <Label>Base URL (optional)</Label>
                <Input value={form.baseUrl || ''} onChange={e => setForm({ ...form, baseUrl: e.target.value })} placeholder="https://api.openai.com/v1" />
              </div>
              <div className="flex justify-end">
                <Button type="submit" disabled={saving}>{saving ? 'Saving…' : 'Save Profile'}</Button>
              </div>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="p-6 space-y-4">
            <h3 className="font-semibold">Profiles</h3>
            {loading ? (
              <div className="text-sm text-muted-foreground">Loading…</div>
            ) : items.length === 0 ? (
              <div className="text-sm text-muted-foreground">No profiles yet.</div>
            ) : (
              <div className="space-y-2">
                {items.map((p) => (
                  <div key={p.id} className="border rounded p-3 flex items-center justify-between">
                    <div>
                      <div className="font-medium">{p.label}</div>
                      <div className="text-xs text-muted-foreground">{p.provider}{p.base_url ? ` • ${p.base_url}` : ''}</div>
                    </div>
                    <Button variant="destructive" size="sm" onClick={() => remove(p.id)}>Delete</Button>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}
