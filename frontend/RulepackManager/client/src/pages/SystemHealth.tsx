import { useQuery } from "@tanstack/react-query";
import { AdminLayout } from "@/components/platform/AdminLayout";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { systemApi } from "@/lib/api";
import { Activity, Server, Database, Shield, Globe, Clock, Cpu, HardDrive, RefreshCw, CheckCircle, AlertCircle } from "lucide-react";

export default function SystemHealth() {
  const { data: healthData, isLoading: healthLoading, refetch: refetchHealth } = useQuery({
    queryKey: ["/api/healthz"],
    queryFn: () => systemApi.getHealth(),
    refetchInterval: 60000, // Reduced to 60 seconds for better performance
    staleTime: 30 * 1000, // 30 seconds cache
  });

  const { data: readinessData, isLoading: readinessLoading, refetch: refetchReadiness } = useQuery({
    queryKey: ["/api/readyz"],
    queryFn: () => systemApi.getReadiness(),
    refetchInterval: 60000, // Reduced to 60 seconds
    staleTime: 30 * 1000, // 30 seconds cache
  });

  const { data: systemInfoData, isLoading: systemInfoLoading } = useQuery({
    queryKey: ["/api/system/info"],
    queryFn: () => systemApi.getInfo(),
    staleTime: 5 * 60 * 1000, // 5 minutes cache for system info
  });

  const { data: statsData } = useQuery({
    queryKey: ["/api/system/stats"],
    queryFn: () => systemApi.getStats(),
    staleTime: 2 * 60 * 1000, // 2 minutes cache for stats
  });

  const handleRefresh = () => {
    refetchHealth();
    refetchReadiness();
  };

  const systemInfo = systemInfoData?.data;
  const readiness = readinessData as any;
  const health = healthData as any;
  const healthStatus = typeof health === 'string' ? health : (health?.status ?? 'unknown');
  const readinessChecks = typeof readiness === 'object' && readiness ? (readiness.checks ?? {}) : {} as any;
  const stats = statsData?.data;

  const getStatusColor = (status: string | boolean) => {
    if (status === 'ok' || status === true) return 'text-green-600';
    if (status === 'degraded') return 'text-amber-600';
    return 'text-red-600';
  };

  const getStatusBadge = (status: string | boolean) => {
    if (status === 'ok' || status === true) return 'default';
    if (status === 'degraded') return 'secondary';
    return 'destructive';
  };

  const getStatusIcon = (status: string | boolean) => {
    if (status === 'ok' || status === true) return <CheckCircle className="h-4 w-4" />;
    return <AlertCircle className="h-4 w-4" />;
  };

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    
    if (days > 0) return `${days}d ${hours}h ${minutes}m`;
    if (hours > 0) return `${hours}h ${minutes}m`;
    return `${minutes}m`;
  };

  const systemComponents = [
    { name: "API Server", status: healthStatus || 'unknown', icon: Server },
    { name: "Database", status: readinessChecks?.database ?? false, icon: Database },
    { name: "Authentication", status: readinessChecks?.auth ?? false, icon: Shield },
    { name: "Storage", status: readinessChecks?.storage ?? false, icon: HardDrive },
  ];

  return (
<AdminLayout 
      title="System Health" 
      description="Monitor system status and performance"
    >
      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-bold text-foreground">System Health</h2>
          <p className="text-muted-foreground">Monitor system status and performance</p>
        </div>
        <Button onClick={handleRefresh} variant="outline" data-testid="button-refresh">
          <RefreshCw className="mr-2 h-4 w-4" />
          Refresh
        </Button>
      </div>

      {/* Overall Status */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="flex items-center">
            <Activity className="mr-2 h-5 w-5" />
            Overall System Status
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-8">
            {healthLoading ? (
              <div className="text-center">
                <Skeleton className="h-12 w-12 rounded-full mx-auto mb-4" />
                <Skeleton className="h-6 w-24 mx-auto" />
              </div>
            ) : (
              <div className="text-center">
                <div className={`h-16 w-16 rounded-full mx-auto mb-4 flex items-center justify-center ${
                  health?.status === 'ok' ? 'bg-green-100' : 'bg-red-100'
                }`}>
                  {getStatusIcon(healthStatus || 'error')}
                </div>
                <h3 className={`text-2xl font-bold ${getStatusColor(healthStatus || 'error')}`} data-testid="overall-status">
                  {healthStatus === 'ok' ? 'Healthy' : 'Unhealthy'}
                </h3>
                <p className="text-muted-foreground">
                  Last checked: {new Date().toLocaleTimeString()}
                </p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* System Components */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-6">
        {systemComponents.map((component) => (
          <Card key={component.name}>
            <CardContent className="p-6">
              <div className="flex items-center justify-between mb-4">
                <div className="h-10 w-10 bg-primary/10 rounded-lg flex items-center justify-center">
                  <component.icon className="h-5 w-5 text-primary" />
                </div>
                {readinessLoading ? (
                  <Skeleton className="h-6 w-16" />
                ) : (
                  <Badge variant={getStatusBadge(component.status) as any} data-testid={`status-${component.name.toLowerCase().replace(' ', '-')}`}>
                    {typeof component.status === 'boolean' 
                      ? (component.status ? 'Healthy' : 'Error')
                      : component.status}
                  </Badge>
                )}
              </div>
              <h3 className="text-lg font-semibold text-foreground">{component.name}</h3>
              <p className="text-sm text-muted-foreground">
                {typeof component.status === 'boolean' 
                  ? (component.status ? 'Operating normally' : 'Service unavailable')
                  : 'Monitoring active'}
              </p>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* System Information */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <Server className="mr-2 h-5 w-5" />
              System Information
            </CardTitle>
          </CardHeader>
          <CardContent>
            {systemInfoLoading ? (
              <div className="space-y-4">
                {Array.from({ length: 5 }).map((_, i) => (
                  <div key={i} className="flex justify-between">
                    <Skeleton className="h-4 w-24" />
                    <Skeleton className="h-4 w-32" />
                  </div>
                ))}
              </div>
            ) : (
              <div className="space-y-4">
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Version</span>
                  <span className="font-medium" data-testid="system-version">
                    {systemInfo?.version || 'Unknown'}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Build Time</span>
                  <span className="font-medium">
                    {systemInfo?.build_time ? new Date(systemInfo.build_time).toLocaleDateString() : 'Unknown'}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Runtime</span>
                  <span className="font-medium">
                    {systemInfo?.go_version || 'Node.js'}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Uptime</span>
                  <span className="font-medium" data-testid="system-uptime">
                    {systemInfo?.uptime ? formatUptime(systemInfo.uptime) : 'Unknown'}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-muted-foreground">Enforcement Mode</span>
                  <Badge variant="default" data-testid="enforcement-mode">
                    {systemInfo?.enforcement_mode || 'Unknown'}
                  </Badge>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <Cpu className="mr-2 h-5 w-5" />
              Performance Metrics
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="flex justify-between">
                <span className="text-muted-foreground">Active RulePacks</span>
                <span className="font-medium text-primary" data-testid="metric-active-rulepacks">
                  {stats?.activeRulePacks || 0}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Active Tenants</span>
                <span className="font-medium text-blue-600" data-testid="metric-active-tenants">
                  {stats?.activeTenants || 0}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">Threats Blocked</span>
                <span className="font-medium text-red-600" data-testid="metric-threats-blocked">
                  {stats?.totalThreatsBlocked?.toLocaleString() || 0}
                </span>
              </div>
              <div className="flex justify-between">
                <span className="text-muted-foreground">System Health</span>
                <Badge variant="default" className="bg-green-100 text-green-800">
                  {stats?.systemHealth || 'Unknown'}
                </Badge>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
</AdminLayout>
  );
}
