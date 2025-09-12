import { useEffect, useMemo, useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Layout } from '@/components/Layout';
import { PageHeader } from '@/components/PageHeader';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Legend,
  Filler,
  TimeScale,
} from 'chart.js';
import { Line } from 'react-chartjs-2';

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler, TimeScale);

// Helper to transform Prometheus range-vector response to chartable series
function toSeries(result: any[]): { labels: string[]; datasets: { label: string; data: number[] }[] } {
  const labels: string[] = result[0]?.values?.map((v: [number,string]) => new Date(v[0]*1000).toLocaleTimeString()) || [];
  const datasets = result.map((s: any, i: number) => ({
    label: s.metric?.pod || s.metric?.instance || `series_${i+1}`,
    data: s.values.map((v: [number,string]) => parseFloat(v[1])),
    fill: false,
    borderWidth: 1,
  }));
  return { labels, datasets };
}

export default function EnforcerMonitoring() {
  const [data, setData] = useState<any | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let aborted = false;
    fetch('/api/metrics/enforcer?range=6h&step=60s', { credentials: 'include' })
      .then(r => r.ok ? r.json() : Promise.reject(new Error(`${r.status}: ${r.statusText}`)))
      .then(j => { if (!aborted) setData(j); })
      .catch(e => { if (!aborted) setError(String(e?.message || e)); });
    return () => { aborted = true; };
  }, []);

  const req = useMemo(() => toSeries(data?.requests?.data?.result || []), [data]);
  const red = useMemo(() => toSeries(data?.redactions?.data?.result || []), [data]);
  const cpu = useMemo(() => toSeries(data?.cpu?.data?.result || []), [data]);
  const heap = useMemo(() => toSeries(data?.heap?.data?.result || []), [data]);

  return (
    <Layout title="Enforcer Monitoring" description="Curated charts from Prometheus">
      <div className="container mx-auto px-4 py-6 sm:py-8">
        <PageHeader title="Enforcer Monitoring" subtitle="Usage & Health (curated)" />
        {error ? (
          <div className="text-red-600">{error}</div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <Card>
              <CardHeader><CardTitle>Request rate</CardTitle></CardHeader>
              <CardContent>
                <Line data={{ labels: req.labels, datasets: req.datasets }} options={{ responsive: true, plugins: { legend: { display: true } } }} />
              </CardContent>
            </Card>
            <Card>
              <CardHeader><CardTitle>Redactions (5m increase)</CardTitle></CardHeader>
              <CardContent>
                <Line data={{ labels: red.labels, datasets: red.datasets }} options={{ responsive: true }} />
              </CardContent>
            </Card>
            <Card>
              <CardHeader><CardTitle>CPU (rate per pod)</CardTitle></CardHeader>
              <CardContent>
                <Line data={{ labels: cpu.labels, datasets: cpu.datasets }} options={{ responsive: true }} />
              </CardContent>
            </Card>
            <Card>
              <CardHeader><CardTitle>Heap alloc bytes</CardTitle></CardHeader>
              <CardContent>
                <Line data={{ labels: heap.labels, datasets: heap.datasets }} options={{ responsive: true }} />
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </Layout>
  );
}

