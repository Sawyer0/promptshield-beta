import { useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { Layout } from '@/components/Layout';
import { PageHeader } from '@/components/PageHeader';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Activity, AlertTriangle, XCircle, Shield, FileText, Users, ArrowRight, Settings, Eye, Plus, Edit, Trash2 } from 'lucide-react';
import { useLocation } from 'wouter';
import { auditApi, policyAssignmentApi, rulePackApi } from '@/lib/api';
import { RulePackModal } from '@/components/RulePackModal';
import { AssignmentModal } from '@/components/AssignmentModal';
import { useToast } from '@/hooks/use-toast';
import { queryClient } from '@/lib/queryClient';
import { handleApiError } from '@/lib/api';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

export default function Dashboard() {
  const [_, setLocation] = useLocation();
  const { toast } = useToast();
  const [isCreateRulePackModalOpen, setIsCreateRulePackModalOpen] = useState(false);
  const [isAssignRulePackModalOpen, setIsAssignRulePackModalOpen] = useState(false);

  // Fetch last 24h audit to power stat cards (same pattern as Audit Logs)
  const { data: auditResponse } = useQuery({
    queryKey: ['/api/v1/audit/violations', { timeRange: '24h', limit: 500 }],
    queryFn: () => auditApi.getAll({
      timeRange: '24h',
      limit: 500,
      offset: 0,
    }),
    staleTime: 30 * 1000,
  });

  // Fetch policy assignments for protected endpoints table
  const { data: assignmentsResponse } = useQuery({
    queryKey: ['/api/v1/policy-assignments'],
    queryFn: async () => {
      try {
        return await policyAssignmentApi.getAll();
      } catch (e: any) {
        const msg = String((e?.message || e) as string);
        if (msg.startsWith('404')) {
          return { success: true, data: [] } as any;
        }
        throw e;
      }
    },
    staleTime: 30 * 1000,
  });

  // Fetch rulepacks for assignment modal
  const { data: rulePacksResponse } = useQuery({
    queryKey: ['/api/rulepacks'],
    queryFn: () => rulePackApi.getAll(),
    staleTime: 5 * 60 * 1000,
  });

  const auditEvents = auditResponse?.data || [];
  const totalEvents = auditResponse?.total || 0;
  
  // Security-focused stats (matching AuditLogs page)
  const violationsCount = auditEvents.filter((e: any) => e.action === 'violation').length;
  const denialsCount = auditEvents.filter((e: any) => e.metadata?.decision === 'deny').length;
  const riskEventsCount = auditEvents.filter((e: any) => e.action === 'risk_detected').length;
  const allowedRequests = auditEvents.filter((e: any) => e.metadata?.decision === 'allow').length;

  const assignments = assignmentsResponse?.data || [];
  const rulePacks = rulePacksResponse?.data || [];

  // Create RulePack mutation
  const createRulePackMutation = useMutation({
    mutationFn: rulePackApi.create,
    onSuccess: () => {
      toast({
        title: "Success",
        description: "RulePack created successfully!",
      });
      setIsCreateRulePackModalOpen(false);
      queryClient.invalidateQueries({ queryKey: ['/api/rulepacks'] });
      setLocation('/rulepacks');
    },
    onError: (error) => handleApiError(error as unknown as Error, toast),
  });

  // Create Assignment mutation
  const createAssignmentMutation = useMutation({
    mutationFn: policyAssignmentApi.create,
    onSuccess: () => {
      toast({
        title: "Success",
        description: "Assignment created successfully!",
      });
      setIsAssignRulePackModalOpen(false);
      queryClient.invalidateQueries({ queryKey: ['/api/v1/policy-assignments'] });
      setLocation('/policies');
    },
    onError: (error) => handleApiError(error, toast),
  });

  const handleCreateRulePack = async (data: any) => {
    createRulePackMutation.mutate(data);
  };

  const handleCreateAssignment = async (data: any) => {
    createAssignmentMutation.mutate(data);
  };

  // Enhanced assignment data with rulepack names for the table
  const enrichedAssignments = assignments.map((assignment: any) => {
    const rulePack = rulePacks.find(rp => rp.id === assignment.rulepackId);
    return {
      ...assignment,
      rulepackName: rulePack?.name || 'Unknown RulePack',
      rulepackVersion: rulePack?.currentVersionId || '1.0',
    };
  });

  return (
    <Layout title="Dashboard" description="Overview and quick actions">
      <div className="container mx-auto px-4 py-6 sm:py-8">
        <PageHeader
          title="Dashboard"
          subtitle="Overview and quick actions"
          actions={
            <>
              <Button 
                variant="outline" 
                onClick={() => setIsAssignRulePackModalOpen(true)} 
                className="w-full sm:w-auto"
              >
                Assign RulePack
              </Button>
              <Button 
                onClick={() => setIsCreateRulePackModalOpen(true)} 
                className="w-full sm:w-auto"
              >
                Create RulePack
              </Button>
            </>
          }
        />

        {/* Audit Log Stat Cards (moved to top like violations page) */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-6 mb-8">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Protected Requests</CardTitle>
              <Shield className="h-4 w-4 text-blue-600" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-blue-600">{allowedRequests + denialsCount}</div>
              <p className="text-xs text-muted-foreground">Requests processed</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Violations</CardTitle>
              <AlertTriangle className="h-4 w-4 text-red-600" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-red-600">{violationsCount}</div>
              <p className="text-xs text-muted-foreground">Security incidents</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Policy Denials</CardTitle>
              <XCircle className="h-4 w-4 text-orange-600" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-orange-600">{denialsCount}</div>
              <p className="text-xs text-muted-foreground">Blocked requests</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Risk Events</CardTitle>
              <Shield className="h-4 w-4 text-yellow-600" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-yellow-600">{riskEventsCount}</div>
              <p className="text-xs text-muted-foreground">Security risks</p>
            </CardContent>
          </Card>
        </div>

        {/* Enterprise-grade Quick Actions */}
        <div className="space-y-8">
          {/* Primary Actions */}
          <div>
            <h2 className="text-xl font-semibold text-foreground mb-6">Quick Actions</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="group p-6 bg-white dark:bg-slate-800 rounded-lg border hover:shadow-md transition-all duration-200 cursor-pointer" onClick={() => setLocation('/rulepacks')}>
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center space-x-3">
                    <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                      <FileText className="h-6 w-6 text-blue-600" />
                    </div>
                    <div>
                      <h3 className="text-lg font-semibold text-foreground">Manage RulePacks</h3>
                      <p className="text-sm text-muted-foreground">Create and configure security policies</p>
                    </div>
                  </div>
                  <ArrowRight className="h-5 w-5 text-muted-foreground group-hover:text-blue-600 transition-colors" />
                </div>
                <div className="space-y-2">
                  <div className="flex items-center text-sm text-muted-foreground">
                    <span className="w-2 h-2 bg-blue-500 rounded-full mr-2"></span>
                    Create new security rules
                  </div>
                  <div className="flex items-center text-sm text-muted-foreground">
                    <span className="w-2 h-2 bg-blue-500 rounded-full mr-2"></span>
                    Edit existing policies
                  </div>
                  <div className="flex items-center text-sm text-muted-foreground">
                    <span className="w-2 h-2 bg-blue-500 rounded-full mr-2"></span>
                    Activate rule versions
                  </div>
                </div>
              </div>

              <div className="group p-6 bg-white dark:bg-slate-800 rounded-lg border hover:shadow-md transition-all duration-200 cursor-pointer" onClick={() => setLocation('/policies')}>
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center space-x-3">
                    <div className="p-2 bg-green-100 dark:bg-green-900/30 rounded-lg">
                      <Users className="h-6 w-6 text-green-600" />
                    </div>
                    <div>
                      <h3 className="text-lg font-semibold text-foreground">Assign RulePacks</h3>
                      <p className="text-sm text-muted-foreground">Deploy policies to endpoints</p>
                    </div>
                  </div>
                  <ArrowRight className="h-5 w-5 text-muted-foreground group-hover:text-green-600 transition-colors" />
                </div>
                <div className="space-y-2">
                  <div className="flex items-center text-sm text-muted-foreground">
                    <span className="w-2 h-2 bg-green-500 rounded-full mr-2"></span>
                    Target specific endpoints
                  </div>
                  <div className="flex items-center text-sm text-muted-foreground">
                    <span className="w-2 h-2 bg-green-500 rounded-full mr-2"></span>
                    Set enforcement priorities
                  </div>
                  <div className="flex items-center text-sm text-muted-foreground">
                    <span className="w-2 h-2 bg-green-500 rounded-full mr-2"></span>
                    Enable/disable assignments
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Secondary Actions */}
          <div>
            <h2 className="text-xl font-semibold text-foreground mb-6">Security Operations</h2>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              <div className="group p-6 bg-white dark:bg-slate-800 rounded-lg border hover:shadow-md transition-all duration-200 cursor-pointer" onClick={() => setLocation('/violations')}>
                <div className="flex items-center space-x-3 mb-4">
                  <div className="p-2 bg-orange-100 dark:bg-orange-900/30 rounded-lg">
                    <AlertTriangle className="h-5 w-5 text-orange-600" />
                  </div>
                  <div>
                    <h3 className="text-base font-semibold text-foreground">Review Violations</h3>
                    <p className="text-sm text-muted-foreground">Triage security events</p>
                  </div>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Blocked requests & alerts</span>
                  <ArrowRight className="h-4 w-4 text-muted-foreground group-hover:text-orange-600 transition-colors" />
                </div>
              </div>

              <div className="group p-6 bg-white dark:bg-slate-800 rounded-lg border hover:shadow-md transition-all duration-200 cursor-pointer" onClick={() => setLocation('/audit')}>
                <div className="flex items-center space-x-3 mb-4">
                  <div className="p-2 bg-purple-100 dark:bg-purple-900/30 rounded-lg">
                    <Eye className="h-5 w-5 text-purple-600" />
                  </div>
                  <div>
                    <h3 className="text-base font-semibold text-foreground">Audit Trail</h3>
                    <p className="text-sm text-muted-foreground">Complete activity log</p>
                  </div>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Search & export logs</span>
                  <ArrowRight className="h-4 w-4 text-muted-foreground group-hover:text-purple-600 transition-colors" />
                </div>
              </div>

              <div className="group p-6 bg-white dark:bg-slate-800 rounded-lg border hover:shadow-md transition-all duration-200 cursor-pointer" onClick={() => setLocation('/account/security')}>
                <div className="flex items-center space-x-3 mb-4">
                  <div className="p-2 bg-slate-100 dark:bg-slate-700 rounded-lg">
                    <Settings className="h-5 w-5 text-slate-600" />
                  </div>
                  <div>
                    <h3 className="text-base font-semibold text-foreground">Settings</h3>
                    <p className="text-sm text-muted-foreground">Account & preferences</p>
                  </div>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-muted-foreground">Security & configuration</span>
                  <ArrowRight className="h-4 w-4 text-muted-foreground group-hover:text-slate-600 transition-colors" />
                </div>
              </div>
            </div>
          </div>

          {/* Protected Endpoints Table */}
          <div>
            <h2 className="text-xl font-semibold text-foreground mb-6">Protected Endpoints</h2>
            <Card>
              <CardContent className="p-0">
                {enrichedAssignments.length === 0 ? (
                  <div className="text-center py-8">
                    <Shield className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                    <h3 className="text-lg font-semibold text-foreground mb-2">No protected endpoints</h3>
                    <p className="text-muted-foreground mb-4">
                      Assign RulePacks to start protecting your API endpoints
                    </p>
                    <Button onClick={() => setIsAssignRulePackModalOpen(true)}>
                      <Shield className="mr-2 h-4 w-4" />
                      Assign RulePack
                    </Button>
                  </div>
                ) : (
                  <div className="overflow-x-auto">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Endpoint</TableHead>
                          <TableHead>RulePack</TableHead>
                          <TableHead>Priority</TableHead>
                          <TableHead>Status</TableHead>
                          <TableHead>Created</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {enrichedAssignments.slice(0, 5).map((assignment: any) => (
                          <TableRow key={assignment.id} className="hover:bg-muted/50">
                            <TableCell>
                              <div className="font-mono text-sm text-foreground">
                                {assignment.targetScope}
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="space-y-1">
                                <div className="text-sm font-medium text-foreground">
                                  {assignment.rulepackName}
                                </div>
                                <div className="text-xs text-muted-foreground">
                                  v{assignment.rulepackVersion}
                                </div>
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center space-x-2">
                                <Badge variant="outline" className="text-xs">
                                  {assignment.priority}
                                </Badge>
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center space-x-2">
                                <div className={`w-2 h-2 rounded-full ${assignment.enabled ? 'bg-green-500' : 'bg-gray-400'}`}></div>
                                <span className="text-sm text-foreground">
                                  {assignment.enabled ? 'Active' : 'Inactive'}
                                </span>
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="text-sm text-muted-foreground">
                                {assignment.createdAt ? new Date(assignment.createdAt).toLocaleDateString() : 'Unknown'}
                              </div>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                    {enrichedAssignments.length > 5 && (
                      <div className="p-4 border-t">
                        <Button 
                          variant="outline" 
                          onClick={() => setLocation('/policies')}
                          className="w-full"
                        >
                          View All {enrichedAssignments.length} Protected Endpoints
                        </Button>
                      </div>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      </div>

      {/* Modals */}
      <RulePackModal
        isOpen={isCreateRulePackModalOpen}
        onClose={() => setIsCreateRulePackModalOpen(false)}
        onSubmit={handleCreateRulePack}
        isLoading={createRulePackMutation.isPending}
      />
      
      <AssignmentModal
        isOpen={isAssignRulePackModalOpen}
        onClose={() => setIsAssignRulePackModalOpen(false)}
        onSubmit={handleCreateAssignment}
        rulePacks={rulePacks}
        isLoading={createAssignmentMutation.isPending}
      />
    </Layout>
  );
}
