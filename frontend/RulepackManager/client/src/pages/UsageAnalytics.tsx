import { useEffect, useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Layout } from '@/components/Layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Button } from '@/components/ui/button';
import { usageApi, auditApi } from '@/lib/api';

type Point = { ts: string; value: number };

export default function UsageAnalytics() {
  const [range, setRange] = useState<'24h'|'7d'|'30d'>('7d');
  const [groupBy, setGroupBy] = useState<'day'|'hour'|'month'>('day');

  const { data: summary, refetch: refetchSummary, isLoading: loadingSummary } = useQuery({
    queryKey: ['usage-summary', range, groupBy],
    queryFn: async () => {
      const now = new Date();
      const from = new Date(now);
      if (range === '24h') from.setHours(now.getHours() - 24);
      else if (range === '7d') from.setDate(now.getDate() - 7);
      else if (range === '30d') from.setDate(now.getDate() - 30);
      const res = await usageApi.getSummary({ from: from.toISOString(), to: now.toISOString(), by: groupBy });
      return res;
    },
    staleTime: 60_000,
  });

  const { data: byEndpoint, isLoading: loadingEndpoint } = useQuery({
    queryKey: ['usage-by-endpoint', range],
    queryFn: async () => {
      const now = new Date();
      const from = new Date(now);
      if (range === '24h') from.setHours(now.getHours() - 24);
      else if (range === '7d') from.setDate(now.getDate() - 7);
      else if (range === '30d') from.setDate(now.getDate() - 30);
      const res = await usageApi.getByEndpoint({ from: from.toISOString(), to: now.toISOString() });
      return res;
    },
    staleTime: 60_000,
  });

  const totalRequests = useMemo(() => {
    const series: Point[] = (summary?.series || []) as any;
    return Array.isArray(series) ? series.reduce((a, p) => a + (p?.value || 0), 0) : 0;
  }, [summary]);

  const exportCsv = () => {
    const series: Point[] = (summary?.series || []) as any;
    const rows = [['timestamp','value'], ...series.map(p => [p.ts, String(p.value)])];
    const csv = rows.map(r => r.join(',')).join('\n');
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url; a.download = `usage_${range}_${groupBy}.csv`; a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <Layout title="Usage Analytics" description="Usage metrics for billing and planning">
      <div className="space-y-6">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle>Summary</CardTitle>
            <div className="flex gap-2">
              <Select value={range} onValueChange={(v: any) => setRange(v)}>
                <SelectTrigger className="w-28"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="24h">Last 24h</SelectItem>
                  <SelectItem value="7d">Last 7d</SelectItem>
                  <SelectItem value="30d">Last 30d</SelectItem>
                </SelectContent>
              </Select>
              <Select value={groupBy} onValueChange={(v: any) => setGroupBy(v)}>
                <SelectTrigger className="w-28"><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="hour">Hourly</SelectItem>
                  <SelectItem value="day">Daily</SelectItem>
                  <SelectItem value="month">Monthly</SelectItem>
                </SelectContent>
              </Select>
              <Button variant="outline" onClick={() => refetchSummary()}>Refresh</Button>
              <Button variant="outline" onClick={exportCsv}>Export CSV</Button>
            </div>
          </CardHeader>
          <CardContent>
            {loadingSummary ? (
              <div className="text-muted-foreground">Loading summary…</div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                  <div className="text-sm text-muted-foreground">Total Requests</div>
                  <div className="text-2xl font-semibold">{totalRequests.toLocaleString()}</div>
                </div>
                <div>
                  <div className="text-sm text-muted-foreground">Peak</div>
                  <div className="text-2xl font-semibold">{summary?.peak?.value ?? '-'} <span className="text-xs text-muted-foreground">@ {summary?.peak?.ts ?? ''}</span></div>
                </div>
                <div>
                  <div className="text-sm text-muted-foreground">Avg per {groupBy}</div>
                  <div className="text-2xl font-semibold">{summary?.avg ?? '-'}</div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>By Endpoint</CardTitle>
          </CardHeader>
          <CardContent>
            {loadingEndpoint ? (
              <div className="text-muted-foreground">Loading…</div>
            ) : !byEndpoint || !Array.isArray(byEndpoint?.endpoints) ? (
              <div className="text-muted-foreground">No data</div>
            ) : (
              <div className="border rounded overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="bg-muted">
                      <th className="p-2 text-left">Endpoint</th>
                      <th className="p-2 text-left">Requests</th>
                      <th className="p-2 text-left">Avg Latency (ms)</th>
                    </tr>
                  </thead>
                  <tbody>
                    {byEndpoint.endpoints.map((e: any) => (
                      <tr key={e.path} className="hover:bg-muted/50">
                        <td className="p-2 font-mono text-xs">{e.path}</td>
                        <td className="p-2">{(e.count ?? 0).toLocaleString()}</td>
                        <td className="p-2">{Math.round(e.p95_ms ?? e.avg_ms ?? 0)}</td>
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

