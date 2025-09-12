import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Layout } from '@/components/Layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { auditApi } from '@/lib/api';

export default function ViolationsSummary() {
  const [timeRange, setTimeRange] = useState<'24h'|'7d'|'30d'>('7d');

  const { data, isLoading } = useQuery({
    queryKey: ['violations-summary', timeRange],
    queryFn: async () => {
      const limit = 5000;
      const map: Record<typeof timeRange, string> = { '24h': '24h', '7d': '7d', '30d': '30d' };
      const res = await auditApi.getAll({ actions: ['violation', 'risk_detected', 'request.decision', 'scan.decision'], limit, timeRange: map[timeRange] });
      return res?.data || [];
    },
    staleTime: 30_000,
  });

  const stats = useMemo(() => auditApi.calculateStats(data || []), [data]);
  const topRules = useMemo(() => Object.entries(stats.topRules || {}).sort((a,b) => b[1]-a[1]).slice(0,10), [stats]);

  return (
    <Layout title="Violations Summary" description="Trends and rule statistics">
      <div className="space-y-6">
        <Card>
          <CardHeader className="flex items-center justify-between">
            <CardTitle>Overview</CardTitle>
            <Select value={timeRange} onValueChange={(v: any) => setTimeRange(v)}>
              <SelectTrigger className="w-28"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="24h">Last 24h</SelectItem>
                <SelectItem value="7d">Last 7d</SelectItem>
                <SelectItem value="30d">Last 30d</SelectItem>
              </SelectContent>
            </Select>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="text-muted-foreground">Loading…</div>
            ) : (
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div>
                  <div className="text-sm text-muted-foreground">Total Requests</div>
                  <div className="text-2xl font-semibold">{stats.totalRequests}</div>
                </div>
                <div>
                  <div className="text-sm text-muted-foreground">Denied</div>
                  <div className="text-2xl font-semibold text-red-600">{stats.denied}</div>
                </div>
                <div>
                  <div className="text-sm text-muted-foreground">Quarantined</div>
                  <div className="text-2xl font-semibold text-amber-600">{stats.quarantined}</div>
                </div>
                <div>
                  <div className="text-sm text-muted-foreground">Allowed</div>
                  <div className="text-2xl font-semibold text-green-600">{stats.allowed}</div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Top Rules Triggered</CardTitle>
          </CardHeader>
          <CardContent>
            {!topRules.length ? (
              <div className="text-muted-foreground">No rule triggers recorded</div>
            ) : (
              <div className="border rounded overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="bg-muted">
                      <th className="p-2 text-left">Rule / Reason</th>
                      <th className="p-2 text-left">Count</th>
                    </tr>
                  </thead>
                  <tbody>
                    {topRules.map(([name, count]) => (
                      <tr key={name} className="hover:bg-muted/50">
                        <td className="p-2">{name}</td>
                        <td className="p-2">{count as number}</td>
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

