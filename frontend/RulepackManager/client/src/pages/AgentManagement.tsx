import { useState } from 'react';
import { Layout } from '@/components/Layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { agentApi } from '@/lib/api';

export default function AgentManagement() {
  const [agentId, setAgentId] = useState('');
  const [tools, setTools] = useState('');
  const [scopes, setScopes] = useState('');
  const [ttl, setTtl] = useState(3600);
  const [authRes, setAuthRes] = useState<any | null>(null);
  const [planJson, setPlanJson] = useState('');
  const [planRes, setPlanRes] = useState<any | null>(null);
  const [listing, setListing] = useState<any[] | null>(null);
  const [working, setWorking] = useState(false);

  const authorize = async () => {
    setWorking(true);
    try {
      const payload = {
        agent_id: agentId,
        tools: tools.split(/\s|,|\n/).map(s => s.trim()).filter(Boolean),
        scopes: scopes.split(/\s|,|\n/).map(s => s.trim()).filter(Boolean),
        ttl_seconds: ttl,
      };
      const res = await agentApi.authorize(payload);
      setAuthRes(res);
    } catch (e) {
      setAuthRes({ error: String((e as any)?.message || e) });
    } finally {
      setWorking(false);
    }
  };

  const validate = async () => {
    setWorking(true);
    try {
      const payload = { plan: JSON.parse(planJson || '{}'), agent_id: agentId || undefined };
      const res = await agentApi.validatePlan(payload);
      setPlanRes(res);
    } catch (e) {
      setPlanRes({ error: String((e as any)?.message || e) });
    } finally {
      setWorking(false);
    }
  };

  const refreshExecutions = async () => {
    setWorking(true);
    try {
      const res = await agentApi.listExecutions({ limit: 50 });
      setListing(Array.isArray(res?.executions) ? res.executions : (Array.isArray(res?.data) ? res.data : []));
    } catch (e) {
      setListing([]);
    } finally {
      setWorking(false);
    }
  };

  return (
    <Layout title="Agent Authorization" description="Authorize agents, validate plans, and monitor executions">
      <div className="space-y-6">
        <Card>
          <CardHeader><CardTitle>Authorize Agent</CardTitle></CardHeader>
          <CardContent className="grid grid-cols-1 md:grid-cols-4 gap-3">
            <div>
              <div className="text-sm font-medium mb-1">Agent ID</div>
              <Input value={agentId} onChange={e => setAgentId(e.target.value)} placeholder="agent-123" />
            </div>
            <div>
              <div className="text-sm font-medium mb-1">Tools (comma or newline)</div>
              <Textarea rows={3} value={tools} onChange={e => setTools(e.target.value)} placeholder="browse_web\nsearch_docs" />
            </div>
            <div>
              <div className="text-sm font-medium mb-1">Scopes (comma or newline)</div>
              <Textarea rows={3} value={scopes} onChange={e => setScopes(e.target.value)} placeholder="read,write" />
            </div>
            <div>
              <div className="text-sm font-medium mb-1">TTL (seconds)</div>
              <Input type="number" value={ttl} onChange={e => setTtl(Number(e.target.value || 0))} />
            </div>
            <div className="md:col-span-4 flex gap-2 justify-end">
              <Button onClick={authorize} disabled={working || !agentId}>Authorize</Button>
            </div>
            <div className="md:col-span-4">
              <div className="text-xs text-muted-foreground mb-1">Response</div>
              <pre className="bg-slate-950 text-green-300 p-3 rounded text-xs overflow-auto min-h-[60px]">{authRes ? JSON.stringify(authRes, null, 2) : '—'}</pre>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader><CardTitle>Validate Plan</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            <Textarea rows={10} value={planJson} onChange={e => setPlanJson(e.target.value)} placeholder='{"steps":[{"tool":"search_docs","args":{"q":"…"}}]}' />
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setPlanJson(JSON.stringify({ steps: [{ tool: 'search_docs', args: { q: 'policy leak' } }] }, null, 2))}>Example</Button>
              <Button onClick={validate} disabled={working || !planJson.trim()}>Validate</Button>
            </div>
            <div>
              <div className="text-xs text-muted-foreground mb-1">Validation Result</div>
              <pre className="bg-slate-950 text-green-300 p-3 rounded text-xs overflow-auto min-h-[60px]">{planRes ? JSON.stringify(planRes, null, 2) : '—'}</pre>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex items-center justify-between">
            <CardTitle>Recent Executions</CardTitle>
            <Button variant="outline" onClick={refreshExecutions} disabled={working}>Refresh</Button>
          </CardHeader>
          <CardContent>
            {!listing ? (
              <div className="text-muted-foreground">No data loaded</div>
            ) : listing.length === 0 ? (
              <div className="text-muted-foreground">No executions found</div>
            ) : (
              <div className="border rounded overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="bg-muted">
                      <th className="p-2 text-left">ID</th>
                      <th className="p-2 text-left">Agent</th>
                      <th className="p-2 text-left">Tool</th>
                      <th className="p-2 text-left">Status</th>
                      <th className="p-2 text-left">Time</th>
                    </tr>
                  </thead>
                  <tbody>
                    {listing.map((x: any) => (
                      <tr key={x.id} className="hover:bg-muted/50">
                        <td className="p-2 font-mono text-xs">{x.id}</td>
                        <td className="p-2">{x.agent_id || '—'}</td>
                        <td className="p-2">{x.tool || x.tool_id || '—'}</td>
                        <td className="p-2">{x.status || '—'}</td>
                        <td className="p-2 text-xs">{x.timestamp || x.created_at || '—'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}

