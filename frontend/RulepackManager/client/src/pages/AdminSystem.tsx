import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Layout } from '@/components/Layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { systemApi } from '@/lib/api';
import { apiRequest } from '@/lib/queryClient';

export default function AdminSystem() {
  const { data: info } = useQuery({ queryKey: ['system-info-admin'], queryFn: () => systemApi.getInfo(), staleTime: 30_000 });
  const { data: stats } = useQuery({ queryKey: ['system-stats-admin'], queryFn: () => systemApi.getStats(), staleTime: 30_000 });
  const [result, setResult] = useState<any | null>(null);
  const [working, setWorking] = useState(false);

  const callOp = async (op: 'drain'|'undrain'|'shutdown'|'features-refresh') => {
    setWorking(true);
    try {
      let url = '/api/admin/system/' + op;
      if (op === 'features-refresh') url = '/api/admin/system/features/refresh';
      const res = await apiRequest('POST', url, {});
      const j = await res.json();
      setResult({ ok: true, op, data: j });
    } catch (e: any) {
      setResult({ ok: false, error: String(e?.message || e) });
    } finally {
      setWorking(false);
    }
  };

  return (
    <Layout title="System Administration" description="Manage system features and lifecycle">
      <div className="space-y-6">
        <Card>
          <CardHeader><CardTitle>System Info</CardTitle></CardHeader>
          <CardContent>
            <pre className="bg-slate-950 text-green-300 p-3 rounded text-xs overflow-auto min-h-[60px]">{info ? JSON.stringify(info, null, 2) : '—'}</pre>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle>System Stats</CardTitle></CardHeader>
          <CardContent>
            <pre className="bg-slate-950 text-green-300 p-3 rounded text-xs overflow-auto min-h-[60px]">{stats ? JSON.stringify(stats, null, 2) : '—'}</pre>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle>Controls</CardTitle></CardHeader>
          <CardContent className="flex flex-wrap gap-2">
            <Button variant="outline" onClick={() => callOp('features-refresh')} disabled={working}>Refresh Features</Button>
            <Button variant="outline" onClick={() => callOp('drain')} disabled={working}>Drain</Button>
            <Button variant="outline" onClick={() => callOp('undrain')} disabled={working}>Undrain</Button>
            <Button variant="destructive" onClick={() => callOp('shutdown')} disabled={working}>Shutdown</Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle>Result</CardTitle></CardHeader>
          <CardContent>
            <pre className="bg-slate-950 text-green-300 p-3 rounded text-xs overflow-auto min-h-[60px]">{result ? JSON.stringify(result, null, 2) : '—'}</pre>
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}

