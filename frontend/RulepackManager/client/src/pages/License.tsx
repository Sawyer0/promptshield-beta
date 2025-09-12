import { useEffect, useState } from 'react';
import { Layout } from '@/components/Layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { licenseApi } from '@/lib/api';

export default function License() {
  const [info, setInfo] = useState<any | null>(null);
  const [key, setKey] = useState('');
  const [working, setWorking] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = async () => {
    setError(null);
    try { setInfo(await licenseApi.getInfo()); } catch (e: any) { setError(String(e?.message || e)); }
  };
  useEffect(() => { load(); }, []);

  const update = async () => {
    setWorking(true); setError(null);
    try {
      await licenseApi.update(key);
      setKey('');
      await load();
    } catch (e: any) {
      setError(String(e?.message || e));
    } finally { setWorking(false); }
  };

  return (
    <Layout title="License" description="Manage license information and keys">
      <div className="space-y-6">
        <Card>
          <CardHeader><CardTitle>Current License</CardTitle></CardHeader>
          <CardContent>
            {error && <div className="text-destructive mb-2">{error}</div>}
            <pre className="bg-slate-950 text-green-300 p-3 rounded text-xs overflow-auto min-h-[60px]">{info ? JSON.stringify(info, null, 2) : '—'}</pre>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle>Update License Key</CardTitle></CardHeader>
          <CardContent className="flex gap-2">
            <Input value={key} onChange={e => setKey(e.target.value)} placeholder="paste license key" />
            <Button onClick={update} disabled={!key.trim() || working}>Update</Button>
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}

