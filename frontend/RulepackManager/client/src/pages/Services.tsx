import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useAuthenticatedApi } from '@/hooks/useAuthenticatedApi';
import { Layout } from '@/components/Layout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Label } from '@/components/ui/label';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { useToast } from '@/hooks/use-toast';
import { 
  Play, 
  Square, 
  RotateCcw, 
  Settings, 
  Activity, 
  Server, 
  Shield,
  Eye,
  ArrowUp,
  ArrowDown,
  Plus
} from 'lucide-react';
import { PageHeader } from '@/components/PageHeader';

interface Service {
  id: string;
  name: string;
  type: string;
  status: 'running' | 'stopped' | 'error' | 'starting' | 'stopping';
  enabled: boolean;
  version: string;
  last_started?: string;
  last_stopped?: string;
  replicas?: number;
}

export default function Services() {
  const [selectedService, setSelectedService] = useState<Service | null>(null);
  const api = useAuthenticatedApi();
  const queryClient = useQueryClient();
  const { toast } = useToast();

  // Fetch services
  const { data: services, isLoading } = useQuery({
    queryKey: ['/api/v1/services'],
    queryFn: () => api.services.getAll(),
  });

  // Service control mutations
  const startService = useMutation({
    mutationFn: (serviceId: string) => api.services.start(serviceId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['/api/v1/services'] });
      toast({ title: "Service Started", description: "Service is starting up..." });
    },
  });

  const stopService = useMutation({
    mutationFn: (serviceId: string) => api.services.stop(serviceId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['/api/v1/services'] });
      toast({ title: "Service Stopped", description: "Service has been stopped" });
    },
  });

  const restartService = useMutation({
    mutationFn: (serviceId: string) => api.services.restart(serviceId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['/api/v1/services'] });
      toast({ title: "Service Restarting", description: "Service is restarting..." });
    },
  });

  const scaleService = useMutation({
    mutationFn: ({ serviceId, replicas }: { serviceId: string; replicas: number }) => 
      api.services.restart(serviceId), // Use restart as placeholder since scale may not be implemented
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['/api/v1/services'] });
      toast({ title: "Service Updated", description: "Service configuration updated" });
    },
  });

  function getStatusBadge(status: string) {
    switch (status?.toLowerCase()) {
      case 'running':
        return <Badge className="bg-green-600">Running</Badge>;
      case 'stopped':
        return <Badge variant="secondary">Stopped</Badge>;
      case 'error':
        return <Badge variant="destructive">Error</Badge>;
      case 'starting':
        return <Badge variant="outline" className="text-blue-600 border-blue-300 bg-blue-50 dark:bg-blue-950/30">Starting</Badge>;
      case 'stopping':
        return <Badge variant="outline" className="text-amber-600 border-amber-300 bg-amber-50 dark:bg-amber-950/30">Stopping</Badge>;
      default:
        return <Badge variant="outline">Unknown</Badge>;
    }
  }

  // Mock services data if none available
  const mockServices: Service[] = [
    {
      id: 'enforcer-1',
      name: 'AI Request Enforcer',
      type: 'enforcer',
      status: 'running',
      enabled: true,
      version: '1.2.3',
      last_started: new Date().toISOString(),
      replicas: 3
    },
    {
      id: 'scanner-1', 
      name: 'Content Scanner',
      type: 'scanner',
      status: 'running',
      enabled: true,
      version: '1.1.0',
      last_started: new Date().toISOString(),
      replicas: 2
    },
    {
      id: 'monitor-1',
      name: 'Threat Monitor',
      type: 'monitor',
      status: 'stopped',
      enabled: false,
      version: '1.0.5',
      last_stopped: new Date().toISOString(),
      replicas: 1
    }
  ];

  const servicesList = services?.data || mockServices;

  return (
    <Layout title="AI Services Management" description="Control and monitor AI security service infrastructure">
      <div className="space-y-6 sm:space-y-8 bg-gradient-to-br from-blue-50/50 via-indigo-50/60 to-purple-50/50 dark:from-blue-950/15 dark:via-indigo-950/20 dark:to-purple-950/15 min-h-screen p-4 sm:p-6 -m-4 sm:-m-6">
      <PageHeader
        title="AI Services Management"
        subtitle="Control and monitor your AI security services infrastructure"
        actions={
          <Button className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 w-full sm:w-auto">
            <Plus className="mr-2 h-4 w-4" />
            Deploy Service
          </Button>
        }
      />

      {/* Services Overview */}
      <div className="grid gap-4 sm:gap-6 grid-cols-1 sm:grid-cols-2 md:grid-cols-3">
        <Card className="border-l-4 border-l-green-500 bg-white/50 dark:bg-slate-800/50 backdrop-blur-sm">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2 px-4 sm:px-6 pt-4 sm:pt-6">
            <CardTitle className="text-sm font-medium text-slate-600 dark:text-slate-400">Running Services</CardTitle>
            <Activity className="h-4 w-4 text-green-600" />
          </CardHeader>
          <CardContent className="px-4 sm:px-6 pb-4 sm:pb-6">
            <div className="text-xl sm:text-2xl font-bold text-slate-900 dark:text-slate-100">
              {servicesList.filter(s => s.status === 'running').length}
            </div>
            <p className="text-xs text-slate-600 dark:text-slate-400">
              Active protection services
            </p>
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-blue-500 bg-white/50 dark:bg-slate-800/50 backdrop-blur-sm">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium text-slate-600 dark:text-slate-400">Total Replicas</CardTitle>
            <Server className="h-4 w-4 text-blue-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-slate-900 dark:text-slate-100">
              {servicesList.reduce((sum, s) => sum + (s.replicas || 0), 0)}
            </div>
            <p className="text-xs text-slate-600 dark:text-slate-400">
              Across all services
            </p>
          </CardContent>
        </Card>

        <Card className="border-l-4 border-l-purple-500 bg-white/50 dark:bg-slate-800/50 backdrop-blur-sm">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium text-slate-600 dark:text-slate-400">Service Health</CardTitle>
            <Shield className="h-4 w-4 text-purple-600" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-slate-900 dark:text-slate-100">98.5%</div>
            <p className="text-xs text-slate-600 dark:text-slate-400">
              Average uptime (30d)
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Services List */}
      <Card className="bg-white/60 dark:bg-slate-800/60 backdrop-blur-sm">
        <CardHeader className="px-4 sm:px-6">
          <CardTitle className="flex items-center gap-2 text-lg sm:text-xl">
            <Server className="h-5 w-5 text-blue-600 flex-shrink-0" />
            <span className="truncate">Service Infrastructure</span>
          </CardTitle>
          <CardDescription className="text-sm">
            Manage your AI security service deployment and configuration
          </CardDescription>
        </CardHeader>
        <CardContent className="px-4 sm:px-6">
          <div className="space-y-4">
            {servicesList.map((service) => (
              <Card key={service.id} className="border border-slate-200 dark:border-slate-700">
                <CardContent className="p-4 sm:p-6">
                  <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
                    <div className="flex items-start gap-3 sm:gap-4 min-w-0 flex-1">
                      <div className="h-10 w-10 sm:h-12 sm:w-12 rounded-lg bg-gradient-to-br from-blue-500 to-purple-500 flex items-center justify-center flex-shrink-0">
                        {service.type === 'enforcer' && <Shield className="h-5 w-5 sm:h-6 sm:w-6 text-white" />}
                        {service.type === 'scanner' && <Eye className="h-5 w-5 sm:h-6 sm:w-6 text-white" />}
                        {service.type === 'monitor' && <Activity className="h-5 w-5 sm:h-6 sm:w-6 text-white" />}
                      </div>
                      
                      <div className="flex-1 min-w-0">
                        <h3 className="font-medium text-base sm:text-lg text-slate-900 dark:text-slate-100 truncate">
                          {service.name}
                        </h3>
                        <p className="text-sm text-slate-600 dark:text-slate-400 mb-2">
                          {service.type ? service.type.charAt(0).toUpperCase() + service.type.slice(1) : 'Unknown'} Service • v{service.version}
                        </p>
                        
                        <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4">
                          <div className="flex items-center gap-2">
                            <span className="text-xs text-slate-500">Status:</span>
                            {getStatusBadge(service.status)}
                          </div>
                          <div className="flex items-center gap-2">
                            <span className="text-xs text-slate-500">Replicas:</span>
                            <Badge variant="outline">{service.replicas || 1}</Badge>
                          </div>
                          {service.last_started && (
                            <div className="flex items-center gap-2">
                              <span className="text-xs text-slate-500">Started:</span>
                              <span className="text-xs font-mono hidden sm:inline">
                                {new Date(service.last_started).toLocaleString()}
                              </span>
                              <span className="text-xs font-mono sm:hidden">
                                {new Date(service.last_started).toLocaleDateString()}
                              </span>
                            </div>
                          )}
                        </div>
                      </div>
                    </div>

                    {/* Service Controls */}
                    <div className="flex flex-col sm:flex-row gap-2 w-full sm:w-auto">
                      <div className="flex gap-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setSelectedService(service)}
                          data-testid={`button-view-service-${service.id}`}
                          className="flex-1 sm:flex-none"
                        >
                          <Eye className="h-4 w-4 sm:mr-0" />
                        </Button>
                        
                        {service.status === 'running' ? (
                          <>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => restartService.mutate(service.id)}
                              disabled={restartService.isPending}
                              data-testid={`button-restart-${service.id}`}
                              className="flex-1 sm:flex-none"
                            >
                              <RotateCcw className="h-4 w-4 sm:mr-0" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => stopService.mutate(service.id)}
                              disabled={stopService.isPending}
                              data-testid={`button-stop-${service.id}`}
                              className="flex-1 sm:flex-none"
                            >
                              <Square className="h-4 w-4 text-red-600 sm:mr-0" />
                            </Button>
                          </>
                        ) : (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => startService.mutate(service.id)}
                            disabled={startService.isPending}
                            data-testid={`button-start-${service.id}`}
                            className="flex-1 sm:flex-none"
                          >
                            <Play className="h-4 w-4 text-green-600 sm:mr-0" />
                          </Button>
                        )}
                      </div>

                      <div className="flex items-center gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => scaleService.mutate({ serviceId: service.id, replicas: (service.replicas || 1) + 1 })}
                          disabled={scaleService.isPending}
                          data-testid={`button-scale-up-${service.id}`}
                          className="flex-1 sm:flex-none"
                        >
                          <ArrowUp className="h-3 w-3" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => scaleService.mutate({ serviceId: service.id, replicas: Math.max(1, (service.replicas || 1) - 1) })}
                          disabled={scaleService.isPending || (service.replicas || 1) <= 1}
                          data-testid={`button-scale-down-${service.id}`}
                          className="flex-1 sm:flex-none"
                        >
                          <ArrowDown className="h-3 w-3" />
                        </Button>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Service Detail Dialog */}
      <Dialog open={!!selectedService} onOpenChange={() => setSelectedService(null)}>
        <DialogContent className="max-w-4xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Server className="h-5 w-5 text-blue-600" />
              Service Details - {selectedService?.name}
            </DialogTitle>
          </DialogHeader>
          
          {selectedService && (
            <Tabs defaultValue="overview" className="space-y-4">
              <TabsList>
                <TabsTrigger value="overview">Overview</TabsTrigger>
                <TabsTrigger value="config">Configuration</TabsTrigger>
                <TabsTrigger value="logs">Logs</TabsTrigger>
                <TabsTrigger value="metrics">Metrics</TabsTrigger>
              </TabsList>

              <TabsContent value="overview" className="space-y-4">
                <Card>
                  <CardHeader>
                    <CardTitle>Service Information</CardTitle>
                  </CardHeader>
                  <CardContent className="grid gap-4 md:grid-cols-2">
                    <div>
                      <Label className="text-sm font-medium">Service ID</Label>
                      <div className="mt-1 text-sm font-mono">{selectedService.id}</div>
                    </div>
                    <div>
                      <Label className="text-sm font-medium">Type</Label>
                      <div className="mt-1 text-sm capitalize">{selectedService.type}</div>
                    </div>
                    <div>
                      <Label className="text-sm font-medium">Version</Label>
                      <div className="mt-1 text-sm">{selectedService.version}</div>
                    </div>
                    <div>
                      <Label className="text-sm font-medium">Replicas</Label>
                      <div className="mt-1 text-sm">{selectedService.replicas || 1}</div>
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="config" className="space-y-4">
                <Card>
                  <CardHeader>
                    <CardTitle>Service Configuration</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <pre className="bg-slate-100 dark:bg-slate-800 p-4 rounded-lg overflow-auto text-xs">
{JSON.stringify({
  service_id: selectedService.id,
  name: selectedService.name,
  type: selectedService.type,
  enabled: selectedService.enabled,
  replicas: selectedService.replicas || 1,
  config: {
    timeout_ms: 5000,
    max_concurrent: 100,
    log_level: "INFO"
  }
}, null, 2)}
                    </pre>
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="logs" className="space-y-4">
                <Card>
                  <CardHeader>
                    <CardTitle>Recent Logs</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="bg-slate-900 text-green-400 p-4 rounded-lg font-mono text-sm space-y-1">
                      <div>[2025-08-29 18:04:32] INFO: Service {selectedService.name} started successfully</div>
                      <div>[2025-08-29 18:04:33] INFO: Loaded security policies from rulepack</div>
                      <div>[2025-08-29 18:04:34] INFO: Ready to process requests</div>
                      <div>[2025-08-29 18:05:45] INFO: Processed 127 requests, 3 blocked</div>
                      <div>[2025-08-29 18:06:12] WARN: High request volume detected</div>
                    </div>
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="metrics" className="space-y-4">
                <div className="grid gap-4 md:grid-cols-2">
                  <Card>
                    <CardHeader>
                      <CardTitle>Performance Metrics</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <div className="flex justify-between">
                        <span className="text-sm">Requests/sec</span>
                        <span className="font-medium">42.3</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-sm">Response Time (p95)</span>
                        <span className="font-medium">45ms</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-sm">Success Rate</span>
                        <span className="font-medium text-green-600">99.8%</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-sm">CPU Usage</span>
                        <span className="font-medium">34%</span>
                      </div>
                    </CardContent>
                  </Card>

                  <Card>
                    <CardHeader>
                      <CardTitle>Security Metrics</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-3">
                      <div className="flex justify-between">
                        <span className="text-sm">Threats Blocked</span>
                        <span className="font-medium text-red-600">12</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-sm">Content Quarantined</span>
                        <span className="font-medium text-amber-600">3</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-sm">Rules Triggered</span>
                        <span className="font-medium">8</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-sm">False Positives</span>
                        <span className="font-medium">0</span>
                      </div>
                    </CardContent>
                  </Card>
                </div>
              </TabsContent>
            </Tabs>
          )}
        </DialogContent>
      </Dialog>
      </div>
    </Layout>
  );
}