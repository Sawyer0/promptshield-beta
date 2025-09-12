import { useEffect, useRef, useState } from 'react';
import { Layout } from '@/components/Layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';

type EventItem = { id: string; type: string; ts: string; data: any };

export default function LiveEvents() {
  const [events, setEvents] = useState<EventItem[]>([]);
  const [connected, setConnected] = useState(false);
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    connect();
    return () => {
      try { esRef.current?.close(); } catch { /* noop */ }
    };
  }, []);

  const connect = () => {
    try { esRef.current?.close(); } catch {}
    const es = new EventSource('/api/events');
    es.onopen = () => setConnected(true);
    es.onerror = () => setConnected(false);
    es.onmessage = (ev) => {
      const payload = (() => { try { return JSON.parse(ev.data); } catch { return { raw: ev.data }; } })();
      const item: EventItem = { id: String(Date.now()) + Math.random(), type: payload?.type || 'message', ts: new Date().toISOString(), data: payload };
      setEvents(prev => [item, ...prev].slice(0, 200));
    };
    esRef.current = es;
  };

  const clear = () => setEvents([]);

  return (
    <Layout title="Live Events" description="Server-sent events stream for real-time monitoring">
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className={connected ? 'text-green-600' : 'text-muted-foreground'}>{connected ? 'Connected' : 'Disconnected'}</div>
          <div className="space-x-2">
            <Button variant="outline" onClick={connect}>Reconnect</Button>
            <Button variant="outline" onClick={clear}>Clear</Button>
          </div>
        </div>
        <Card>
          <CardHeader><CardTitle>Stream</CardTitle></CardHeader>
          <CardContent>
            {!events.length ? (
              <div className="text-muted-foreground">No events yet</div>
            ) : (
              <div className="space-y-2">
                {events.map(e => (
                  <div key={e.id} className="border rounded p-2">
                    <div className="text-xs text-muted-foreground">{e.ts} • {e.type}</div>
                    <pre className="bg-slate-950 text-green-300 p-2 rounded text-xs overflow-auto">{JSON.stringify(e.data, null, 2)}</pre>
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

