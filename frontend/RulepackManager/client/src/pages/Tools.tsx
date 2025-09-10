import { useEffect, useState } from 'react';
import { Layout } from '@/components/Layout';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { toolsApi } from '@/lib/api';

export default function Tools() {
  const [tools, setTools] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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
        <Button onClick={load} disabled={loading}>Refresh</Button>
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
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </Layout>
  );
}


