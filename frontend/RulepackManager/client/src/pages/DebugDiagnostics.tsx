import { useEffect, useState } from 'react';
import { Layout } from '@/components/Layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

export default function DebugDiagnostics() {
  const [auth, setAuth] = useState<any | null>(null);
  const [jwt, setJwt] = useState<any | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let aborted = false;
    (async () => {
      try {
        const r1 = await fetch('/api/debug/auth', { credentials: 'include' });
        const j1 = await r1.json();
        if (!aborted) setAuth(j1);
      } catch (e: any) { setError(String(e?.message || e)); }
      try {
        const r2 = await fetch('/api/debug/jwt-config', { credentials: 'include' });
        const j2 = await r2.json();
        if (!aborted) setJwt(j2);
      } catch (e: any) { setError(String(e?.message || e)); }
    })();
    return () => { aborted = true; };
  }, []);

  return (
    <Layout title="Debug & Diagnostics" description="Authentication, JWT, and environment debug info">
      <div className="space-y-6">
        {error && <div className="text-destructive">{error}</div>}
        <Card>
          <CardHeader><CardTitle>Auth Context</CardTitle></CardHeader>
          <CardContent>
            <pre className="bg-slate-950 text-green-300 p-3 rounded text-xs overflow-auto min-h-[60px]">{auth ? JSON.stringify(auth, null, 2) : '—'}</pre>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>JWT Configuration</CardTitle></CardHeader>
          <CardContent>
            <pre className="bg-slate-950 text-green-300 p-3 rounded text-xs overflow-auto min-h-[60px]">{jwt ? JSON.stringify(jwt, null, 2) : '—'}</pre>
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}

