import { useEffect, useState } from 'react';
import { Layout } from '@/components/Layout';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Textarea } from '@/components/ui/textarea';
import { toolsApi } from '@/lib/api';

export default function Tools() {
  const [tools, setTools] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showEdit, setShowEdit] = useState(false);
  const [editing, setEditing] = useState<any | null>(null);
  const [saving, setSaving] = useState(false);

  const load = async () => {
    try {
      setLoading(true);
      const res = await toolsApi.list({ limit: 100 });
      setTools(res.data || []);
    } catch (e: any) {
      setError(String(e?.message || e));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { load(); }, []);

  return (
    <Layout title="Tools & Actions" description="Register tools with capability tags and traits">
      <div className="flex justify-between items-center mb-4">
        <div>
          <h2 className="text-xl font-semibold">Tools Registry</h2>
          <p className="text-sm text-muted-foreground">Capability-tagged tools power presets and Action-Selector.</p>
        </div>
        <div className="space-x-2">
          <Button variant="outline" onClick={() => { setEditing({ tool_id: '', name: '', description: '', capability_tags: [], data_domains: [], risk_score: 0, side_effect: 'none', auth_scope: 'user-delegated' }); setShowEdit(true); }}>New Tool</Button>
          <Button onClick={load} disabled={loading}>Refresh</Button>
        </div>
      </div>
      {loading ? (
        <p className="text-muted-foreground">Loading…</p>
      ) : error ? (
        <p className="text-destructive">{error}</p>
      ) : tools.length === 0 ? (
        <Card>
          <CardContent className="p-8 text-center text-muted-foreground">No tools registered yet.</CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {tools.map((t) => (
            <Card key={t.id} className="hover:shadow-md transition-shadow">
              <CardContent className="p-4">
                <div className="flex items-start justify-between">
                  <div>
                    <div className="font-semibold">{t.name}</div>
                    <div className="text-xs text-muted-foreground break-all">{t.tool_id}</div>
                  </div>
                  <div className="space-x-2">
                    <Button size="sm" variant="ghost" onClick={() => { setEditing(t); setShowEdit(true); }}>Edit</Button>
                    <Button size="sm" variant="ghost" onClick={async () => { if (!confirm(`Delete ${t.name}?`)) return; try { await toolsApi.remove(t.id); await load(); } catch (e:any) { alert(String(e?.message||e)); } }}>Delete</Button>
                  </div>
                </div>
                <div className="mt-2 text-sm text-muted-foreground line-clamp-2">{t.description || '—'}</div>
                <div className="mt-3 flex flex-wrap gap-1">
                  {(t.capability_tags || []).map((tag: string) => (
                    <Badge key={tag} variant="secondary" className="text-xs">{tag}</Badge>
                  ))}
                </div>
                <div className="mt-2 flex flex-wrap gap-1">
                  {(t.data_domains || []).map((d: string) => (
                    <Badge key={d} className="text-xs">{d}</Badge>
                  ))}
                </div>
                <div className="mt-3 text-xs text-muted-foreground">
                  Risk Score: <span className="font-medium">{t.risk_score ?? 0}</span> • Side Effect: <span className="font-medium">{t.side_effect || 'none'}</span> • Auth: <span className="font-medium">{t.auth_scope || 'user-delegated'}</span>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Edit / Create Modal */}
      {showEdit && (
        <Dialog open={showEdit} onOpenChange={setShowEdit}>
          <DialogContent className="max-w-2xl">
            <DialogHeader><DialogTitle>{editing?.id ? 'Edit Tool' : 'Create Tool'}</DialogTitle></DialogHeader>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              <div>
                <div className="text-sm font-medium mb-1">Tool ID</div>
                <Input value={editing?.tool_id || ''} onChange={e => setEditing((p: any) => ({ ...p, tool_id: e.target.value }))} placeholder="unique identifier" />
              </div>
              <div>
                <div className="text-sm font-medium mb-1">Name</div>
                <Input value={editing?.name || ''} onChange={e => setEditing((p: any) => ({ ...p, name: e.target.value }))} placeholder="Human name" />
              </div>
              <div className="md:col-span-2">
                <div className="text-sm font-medium mb-1">Description</div>
                <Textarea rows={3} value={editing?.description || ''} onChange={e => setEditing((p: any) => ({ ...p, description: e.target.value }))} />
              </div>
              <div>
                <div className="text-sm font-medium mb-1">Capability Tags (comma or newline)</div>
                <Textarea rows={3} value={(editing?.capability_tags || []).join('\n')} onChange={e => setEditing((p: any) => ({ ...p, capability_tags: e.target.value.split(/\n|,/).map(s => s.trim()).filter(Boolean) }))} />
              </div>
              <div>
                <div className="text-sm font-medium mb-1">Data Domains (comma or newline)</div>
                <Textarea rows={3} value={(editing?.data_domains || []).join('\n')} onChange={e => setEditing((p: any) => ({ ...p, data_domains: e.target.value.split(/\n|,/).map(s => s.trim()).filter(Boolean) }))} />
              </div>
              <div>
                <div className="text-sm font-medium mb-1">Risk Score (0-100)</div>
                <Input type="number" value={editing?.risk_score ?? 0} onChange={e => setEditing((p: any) => ({ ...p, risk_score: Number(e.target.value||0) }))} />
              </div>
              <div>
                <div className="text-sm font-medium mb-1">Side Effect</div>
                <Input value={editing?.side_effect || 'none'} onChange={e => setEditing((p: any) => ({ ...p, side_effect: e.target.value }))} placeholder="none|reversible|irreversible" />
              </div>
              <div>
                <div className="text-sm font-medium mb-1">Auth Scope</div>
                <Input value={editing?.auth_scope || 'user-delegated'} onChange={e => setEditing((p: any) => ({ ...p, auth_scope: e.target.value }))} placeholder="user-delegated|service-account" />
              </div>
              <div className="md:col-span-2 flex justify-end gap-2 mt-2">
                <Button variant="outline" onClick={() => setShowEdit(false)}>Cancel</Button>
                <Button disabled={saving || !editing?.tool_id || !editing?.name} onClick={async () => {
                  try {
                    setSaving(true);
                    if (editing?.id) {
                      await toolsApi.update(editing.id, {
                        name: editing.name,
                        description: editing.description,
                        capability_tags: editing.capability_tags,
                        data_domains: editing.data_domains,
                        risk_score: editing.risk_score,
                        side_effect: editing.side_effect,
                        auth_scope: editing.auth_scope,
                      });
                    } else {
                      await toolsApi.create({
                        tool_id: editing.tool_id,
                        name: editing.name,
                        description: editing.description,
                        capability_tags: editing.capability_tags,
                        data_domains: editing.data_domains,
                        risk_score: editing.risk_score,
                        side_effect: editing.side_effect,
                        auth_scope: editing.auth_scope,
                      });
                    }
                    setShowEdit(false);
                    await load();
                  } catch (e: any) {
                    alert(String(e?.message || e));
                  } finally {
                    setSaving(false);
                  }
                }}>{saving ? 'Saving…' : 'Save'}</Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
      )}
    </Layout>
  );
}


