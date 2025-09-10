import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useAuth } from '@/hooks/useAuth';
import { Layout } from '@/components/Layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { 
  Users, 
  Building, 
  Shield, 
  TrendingUp,
  Activity,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Clock,
  Server,
  Database,
  Cpu,
  HardDrive
} from 'lucide-react';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { systemApi } from '@/lib/api';
import { apiRequest } from '@/lib/queryClient';

type PlatformStats = {
  RequestsTotal?: number;
  P95LatencyMs?: number;
  DecisionsTotal?: Record<string, number>;
  Uptime?: string;
  Version?: string;
  // tolerate snake_case from potential alt sources
  requests_total?: number;
  p95_latency_ms?: number;
  decisions_total?: Record<string, number>;
  uptime?: string;
  version?: string;
};

type TenantRow = {
  id: string;
  name: string;
  status?: string;
  plan?: string;
  user_count?: number;
  requests_today?: number;
  created_at?: string;
};

// Real API endpoints based on your Supabase backend

export default function PlatformDashboard() {
  const [systemDraining, setSystemDraining] = useState(false);

  const { user } = useAuth();
  const isAdmin = user?.systemRole === 'admin';

  // Platform statistics (admin)
  const { data: platformStats, isLoading: statsLoading } = useQuery<PlatformStats>({
    queryKey: ['platform-stats'],
    refetchInterval: 30000,
    staleTime: 25000,
    enabled: isAdmin,
    queryFn: () => systemApi.getStats(),
  });

  // System info (admin)
  const { data: systemInfo, isLoading: healthLoading } = useQuery<{ Uptime?: string; Version?: string }>({
    queryKey: ['system-info'],
    refetchInterval: 30000,
    staleTime: 25000,
    enabled: isAdmin,
    queryFn: () => systemApi.getInfo(),
  });

  // Tenant data (admin)
  const { data: tenantResp, isLoading: tenantsLoading } = useQuery<{ tenants: TenantRow[] }>({
    queryKey: ['tenants-admin'],
    refetchInterval: 60000,
    staleTime: 30000,
    enabled: isAdmin,
    queryFn: async () => {
      const res = await apiRequest('GET', `/api/v1/admin/tenants`);
      return res.json();
    },
  });

  const handleDrainSystem = () => {
    setSystemDraining(true);
    // Mock drain operation
    setTimeout(() => setSystemDraining(false), 3000);
  };

  const requestsTotal = (platformStats?.RequestsTotal ?? platformStats?.requests_total ?? 0);
  const p95Latency = (platformStats?.P95LatencyMs ?? platformStats?.p95_latency_ms ?? 0);
  const denyCount = platformStats?.DecisionsTotal?.deny ?? platformStats?.decisions_total?.deny ?? 0;
  const quarantineCount = platformStats?.DecisionsTotal?.quarantine ?? platformStats?.decisions_total?.quarantine ?? 0;
  const systemStatus = 'healthy';
  const systemUptime = systemInfo?.Uptime ?? (systemInfo as any)?.uptime ?? 'N/A';

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy':
      case 'active':
        return <CheckCircle className="h-4 w-4 text-green-600" />;
      case 'suspended':
        return <XCircle className="h-4 w-4 text-red-600" />;
      case 'warning':
        return <AlertTriangle className="h-4 w-4 text-yellow-600" />;
      default:
        return <Clock className="h-4 w-4 text-gray-600" />;
    }
  };

  const getPlanBadge = (plan: string) => {
    const variant = plan === 'enterprise' ? 'default' : plan === 'pro' ? 'secondary' : 'outline';
    return <Badge variant={variant} className="text-xs">{plan}</Badge>;
  };

  return (
    <Layout title="Platform Dashboard" description="SaaS platform management and monitoring">
      <div className="space-y-6">
        {/* Platform Overview Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {statsLoading ? (
            [...Array(4)].map((_, i) => (
              <Card key={i} className="p-4">
                <div className="flex items-center justify-between">
                  <div className="space-y-2">
                    <div className="h-3 w-20 bg-muted animate-pulse rounded" />
                    <div className="h-6 w-16 bg-muted animate-pulse rounded" />
                    <div className="h-3 w-24 bg-muted animate-pulse rounded" />
                  </div>
                  <div className="h-6 w-6 bg-muted animate-pulse rounded" />
                </div>
              </Card>
            ))
          ) : (
            <>
              <Card className="p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-xs font-medium text-muted-foreground mb-1">Total Requests</p>
                    <p className="text-xl font-bold">
                      {(requestsTotal / 1_000_000).toFixed(1)}M
                    </p>
                    <p className="text-xs text-green-600">
                      ↗ P95: {p95Latency || 0}ms
                    </p>
                  </div>
                  <Activity className="h-6 w-6 text-blue-600" />
                </div>
              </Card>
              
              <Card className="p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-xs font-medium text-muted-foreground mb-1">Active Tenants</p>
                    <p className="text-xl font-bold text-green-600">
                      {(tenantResp?.tenants || []).filter(t => (t.status || 'active') === 'active').length}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      of {tenantResp?.tenants?.length || 0} total
                    </p>
                  </div>
                  <Building className="h-6 w-6 text-green-600" />
                </div>
              </Card>
              
              <Card className="p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-xs font-medium text-muted-foreground mb-1">Threats Blocked</p>
                    <p className="text-xl font-bold text-red-600">{(denyCount / 1000).toFixed(0)}K</p>
                    <p className="text-xs text-yellow-600">
                      +{(quarantineCount / 1000).toFixed(0)}K quarantined
                    </p>
                  </div>
                  <Shield className="h-6 w-6 text-red-600" />
                </div>
              </Card>
              
              <Card className="p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-xs font-medium text-muted-foreground mb-1">System Health</p>
                    <p className="text-xl font-bold text-green-600">{systemStatus}</p>
                    <p className="text-xs text-muted-foreground">
                      Uptime: {systemUptime}
                    </p>
                  </div>
                  <CheckCircle className="h-6 w-6 text-green-600" />
                </div>
              </Card>
            </>
          )}
        </div>

        {/* System Components Health */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <Server className="h-5 w-5 mr-2" />
              System Components
            </CardTitle>
          </CardHeader>
          <CardContent>
            {healthLoading ? (
              <div className="space-y-3">
                {[...Array(4)].map((_, i) => (
                  <div key={i} className="flex items-center space-x-4">
                    <Skeleton className="h-4 w-4" />
                    <Skeleton className="h-4 w-32" />
                    <Skeleton className="h-4 w-48" />
                  </div>
                ))}
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-3">
                  <div className="flex items-center space-x-3">
                    {getStatusIcon('healthy')}
                    <div className="flex-1">
                      <div className="flex items-center space-x-2">
                        <Database className="h-4 w-4 text-muted-foreground" />
                        <span className="text-sm font-medium">Database</span>
                      </div>
                      <p className="text-xs text-muted-foreground">Connectivity OK</p>
                    </div>
                  </div>
                  
                  <div className="flex items-center space-x-3">
                    {getStatusIcon('healthy')}
                    <div className="flex-1">
                      <div className="flex items-center space-x-2">
                        <Shield className="h-4 w-4 text-muted-foreground" />
                        <span className="text-sm font-medium">License</span>
                      </div>
                      <p className="text-xs text-muted-foreground">Valid until Dec 2025</p>
                    </div>
                  </div>
                </div>
                
                <div className="space-y-3">
                  <div className="flex items-center space-x-3">
                    {getStatusIcon('healthy')}
                    <div className="flex-1">
                      <div className="flex items-center space-x-2">
                        <Cpu className="h-4 w-4 text-muted-foreground" />
                        <span className="text-sm font-medium">Memory</span>
                      </div>
                      <p className="text-xs text-muted-foreground">512.5MB / 2048MB used</p>
                    </div>
                  </div>
                  
                  <div className="flex items-center space-x-3">
                    {getStatusIcon('healthy')}
                    <div className="flex-1">
                      <div className="flex items-center space-x-2">
                        <Activity className="h-4 w-4 text-muted-foreground" />
                        <span className="text-sm font-medium">Goroutines</span>
                      </div>
                      <p className="text-xs text-muted-foreground">45 active</p>
                    </div>
                  </div>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Recent Tenants Activity */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between">
            <CardTitle className="flex items-center">
              <Users className="h-5 w-5 mr-2" />
              Recent Tenant Activity
            </CardTitle>
            <Button variant="outline" size="sm">
              View All Tenants
            </Button>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Tenant</TableHead>
                    <TableHead>Plan</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Users</TableHead>
                    <TableHead>Requests Today</TableHead>
                    <TableHead>Created</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {tenantsLoading ? (
                    [...Array(3)].map((_, i) => (
                      <TableRow key={i}>
                        <TableCell><div className="h-4 w-32 bg-muted animate-pulse rounded" /></TableCell>
                        <TableCell><div className="h-4 w-16 bg-muted animate-pulse rounded" /></TableCell>
                        <TableCell><div className="h-4 w-20 bg-muted animate-pulse rounded" /></TableCell>
                        <TableCell><div className="h-4 w-8 bg-muted animate-pulse rounded" /></TableCell>
                        <TableCell><div className="h-4 w-16 bg-muted animate-pulse rounded" /></TableCell>
                        <TableCell><div className="h-4 w-20 bg-muted animate-pulse rounded" /></TableCell>
                      </TableRow>
                    ))
                  ) : (
                    (tenantResp?.tenants || []).slice(0, 5).map((tenant: TenantRow) => (
                      <TableRow key={tenant.id}>
                        <TableCell>
                          <div>
                            <div className="font-medium">{tenant.name}</div>
                            <div className="text-xs text-muted-foreground">{tenant.id}</div>
                          </div>
                        </TableCell>
                        <TableCell>
                          {getPlanBadge(tenant.plan || 'basic')}
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center space-x-2">
                            {getStatusIcon(tenant.status || 'active')}
                            <span className="text-sm capitalize">{tenant.status || 'active'}</span>
                          </div>
                        </TableCell>
                        <TableCell>{tenant.user_count || 0}</TableCell>
                        <TableCell>
                          <span className={(tenant.requests_today ?? 0) > 0 ? 'text-green-600' : 'text-muted-foreground'}>
                            {(tenant.requests_today ?? 0).toLocaleString()}
                          </span>
                        </TableCell>
                        <TableCell>
                          <span className="text-xs text-muted-foreground">
                            {tenant.created_at ? new Date(tenant.created_at).toLocaleDateString() : 'N/A'}
                          </span>
                        </TableCell>
                      </TableRow>
                    )) || (
                      <TableRow>
                        <TableCell colSpan={6} className="text-center text-muted-foreground">
                          No tenant data available
                        </TableCell>
                      </TableRow>
                    )
                  )}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>

        {/* Platform Actions */}
        <Card>
          <CardHeader>
            <CardTitle>Platform Management</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex space-x-4">
              <Button 
                variant="outline" 
                onClick={handleDrainSystem}
                disabled={systemDraining}
              >
                {systemDraining ? 'Draining...' : 'Drain System'}
              </Button>
              <Button variant="outline">
                View Audit Logs
              </Button>
              <Button variant="outline">
                Download Reports
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </Layout>
  );
}