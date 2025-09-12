import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Layout } from '@/components/Layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { presetsApi } from '@/lib/api';

export default function Presets() {
  const { data, refetch, isLoading } = useQuery({
    queryKey: ['presets-list'],
    queryFn: () => presetsApi.list(),
    staleTime: 60_000,
  });
  const [selected, setSelected] = useState<string | null>(null);
  const [preview, setPreview] = useState<any | null>(null);
  const [working, setWorking] = useState(false);

  useEffect(() => {
    let aborted = false;
    (async () => {
      if (!selected) { setPreview(null); return; }
      try {
        const p = await presetsApi.preview(selected);
        if (!aborted) setPreview(p);
      } catch {
        if (!aborted) setPreview(null);
      }
    })();
    return () => { aborted = true; };
  }, [selected]);

  const importBaseline = async () => {
    setWorking(true);
    try {
      const r = await fetch('/api/presets/import/baseline', { method: 'POST', credentials: 'include' });
      if (!r.ok) throw new Error(`${r.status}`);
      await refetch();
    } catch (e) {
      console.error(e);
    } finally {
      setWorking(false);
    }
  };

  return (
    <Layout title="Security Presets" description="Create and apply pre-built security configurations">
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-semibold">Available Presets</h2>
          <Button onClick={importBaseline} disabled={working}>{working ? 'Importing…' : 'Import Baseline'}</Button>
        </div>

        <Card>
          <CardContent className="p-0">
            {isLoading ? (
              <div className="p-6 text-muted-foreground">Loading…</div>
            ) : !data?.data?.length ? (
              <div className="p-6 text-muted-foreground">No presets found</div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-0">
                <div className="border-r">
                  <ul>
                    {data.data.map((p: any) => (
                      <li key={p.id} className={`p-4 cursor-pointer hover:bg-muted ${selected===p.id?'bg-muted/60':''}`} onClick={() => setSelected(p.id)}>
                        <div className="font-medium">{p.name}</div>
                        <div className="text-sm text-muted-foreground">{p.description || '—'}</div>
                      </li>
                    ))}
                  </ul>
                </div>
                <div>
                  <div className="p-4 border-b font-medium">Preview</div>
                  <div className="p-4">
                    {!selected ? (
                      <div className="text-muted-foreground">Select a preset to preview</div>
                    ) : !preview ? (
                      <div className="text-muted-foreground">No preview available</div>
                    ) : (
                      <pre className="bg-slate-950 text-green-300 p-4 rounded overflow-auto text-xs">{JSON.stringify(preview, null, 2)}</pre>
                    )}
                  </div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}

