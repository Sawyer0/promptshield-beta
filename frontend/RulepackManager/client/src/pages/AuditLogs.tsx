import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Layout } from '@/components/Layout';
import { PageHeader } from '@/components/PageHeader';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { SearchInput } from '@/components/common/SearchInput';
import { SelectControl } from '@/components/common/SelectControl';
import { FilterToolbar } from '@/components/common/FilterToolbar';
import { Skeleton } from '@/components/ui/skeleton';
import { useToast } from '@/hooks/use-toast';
import { auditApi, handleApiError } from '@/lib/api';
import { 
  Shield, 
  AlertTriangle, 
  CheckCircle, 
  XCircle, 
  Activity,
  Search,
  Filter,
  Download,
  Calendar,
  User,
  Globe,
  Clock,
  AlertCircle,
  Eye,
  RefreshCw,
  Plus,
  Edit,
  Trash2
} from 'lucide-react';
import { DataTable, type Column } from '@/components/common/DataTable';
import { AuditEventModal } from '@/components/AuditEventModal';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Pagination } from '@/components/common/Pagination';

export default function AuditLogs() {
  const [selectedEvent, setSelectedEvent] = useState<any | null>(null);
  const [searchTerm, setSearchTerm] = useState("");
  const [actionFilter, setActionFilter] = useState<string>("all");
  const [decisionFilter, setDecisionFilter] = useState<string>("all");
  const [dateRange, setDateRange] = useState<string>("7");
  const [page, setPage] = useState(0);
  const [limit] = useState(50);
  const { toast } = useToast();

  const { data: auditResponse, isLoading, isFetching, refetch } = useQuery({
    queryKey: ["/api/v1/audit", { searchTerm, actionFilter, decisionFilter, dateRange, page, limit }],
    queryFn: () => auditApi.getAll({
      search: searchTerm || undefined,
      actions: actionFilter !== "all" ? [actionFilter] : undefined,
      decision: decisionFilter !== "all" ? decisionFilter : undefined,
      timeRange: `${parseInt(dateRange)}d`,
      limit,
      offset: page * limit,
    }),
    staleTime: 30 * 1000, // 30 seconds
  });

  const handleExportLogs = async () => {
    try {
      const exportData = await auditApi.getAll({
        search: searchTerm || undefined,
        actions: actionFilter !== "all" ? [actionFilter] : undefined,
        decision: decisionFilter !== "all" ? decisionFilter : undefined,
        timeRange: `${parseInt(dateRange)}d`,
        limit: 10_000,
        offset: 0,
        format: 'json',
      });
      
      const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `audit-logs-${new Date().toISOString().split('T')[0]}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      
      toast({
        title: "Export Complete",
        description: "Audit logs exported successfully!",
      });
    } catch (error) {
      handleApiError(error as Error, toast);
    }
  };

  const auditEvents = auditResponse?.data || [];
  const totalEvents = auditResponse?.total || 0;
  const totalPages = Math.ceil(totalEvents / limit);

  const filteredEvents = useMemo(() => {
    return auditEvents.filter((event: any) => {
      const meta = event.metadata || {};
      const endpoint = meta.path || meta.endpoint || '';
      const user = event.actor_email || event.actor_id || '';
      const decision = meta.decision || '';
      const resource = `${event.object_type || ''} ${event.object_id || ''}`.trim();

      const matchesSearch = !searchTerm ||
        user.toLowerCase().includes(searchTerm.toLowerCase()) ||
        String(event.action || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
        endpoint.toLowerCase().includes(searchTerm.toLowerCase()) ||
        resource.toLowerCase().includes(searchTerm.toLowerCase()) ||
        String(meta.reason || '').toLowerCase().includes(searchTerm.toLowerCase());

      const matchesAction = actionFilter === "all" || String(event.action) === actionFilter;
      const matchesDecision = decisionFilter === "all" || String(decision) === decisionFilter;

      return matchesSearch && matchesAction && matchesDecision;
    });
  }, [auditEvents, searchTerm, actionFilter, decisionFilter]);

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      hour12: true
    });
  };

  const formatTimeAgo = (timestamp: string) => {
    const date = new Date(timestamp);
    const now = new Date();
    const diffInMinutes = Math.floor((now.getTime() - date.getTime()) / (1000 * 60));
    
    if (diffInMinutes < 1) return 'Just now';
    if (diffInMinutes < 60) return `${diffInMinutes}m ago`;
    if (diffInMinutes < 1440) return `${Math.floor(diffInMinutes / 60)}h ago`;
    return `${Math.floor(diffInMinutes / 1440)}d ago`;
  };

  const getActionIcon = (action: string) => {
    switch (action) {
      case 'create':
        return <Plus className="h-4 w-4 text-green-600" />;
      case 'update':
        return <Edit className="h-4 w-4 text-blue-600" />;
      case 'delete':
        return <Trash2 className="h-4 w-4 text-red-600" />;
      case 'access':
        return <Eye className="h-4 w-4 text-purple-600" />;
      case 'violation':
        return <AlertTriangle className="h-4 w-4 text-orange-600" />;
      case 'risk_detected':
        return <Shield className="h-4 w-4 text-red-600" />;
      case 'rule_triggered':
        return <Activity className="h-4 w-4 text-amber-600" />;
      case 'policy_applied':
        return <CheckCircle className="h-4 w-4 text-green-600" />;
      default:
        return <AlertCircle className="h-4 w-4 text-gray-600" />;
    }
  };

  const getDecisionBadge = (decision?: string) => {
    switch (decision) {
      case 'deny':
        return 'destructive';
      case 'quarantine':
        return 'secondary';
      case 'allow':
        return 'default';
      default:
        return 'outline';
    }
  };

  const getStatusColor = (status?: number) => {
    if (status == null) return 'text-gray-600';
    if (status >= 200 && status < 300) return 'text-green-600';
    if (status >= 300 && status < 400) return 'text-blue-600';
    if (status >= 400 && status < 500) return 'text-orange-600';
    if (status >= 500) return 'text-red-600';
    return 'text-gray-600';
  };

  return (
    <>
    <Layout title="Audit Logs" description="Complete audit trail of security events and violations">
      <div className="container mx-auto px-4 py-6 sm:py-8">
        <PageHeader
          title="Audit Logs"
          subtitle="Complete audit trail of security events and violations"
          actions={
            <>
              <Button 
                onClick={() => refetch()} 
                variant="outline" 
                disabled={isLoading || isFetching}
                data-testid="button-refresh"
                className="w-full sm:w-auto"
              >
                <RefreshCw className={`h-4 w-4 mr-2 ${isLoading || isFetching ? 'animate-spin' : ''}`} />
                Refresh
              </Button>
              <Button onClick={handleExportLogs} variant="outline" data-testid="button-export" className="w-full sm:w-auto">
                <Download className="h-4 w-4 mr-2" />
                Export Logs
              </Button>
            </>
          }
        />

        {/* Filters */}
        <Card className="mb-4 sm:mb-6">
          <FilterToolbar
            search={<SearchInput value={searchTerm} onChange={setSearchTerm} placeholder="Search events..." data-testid="input-search-events" />}
            actions={
              <Button
                variant="outline"
                onClick={() => { setSearchTerm(''); setActionFilter('all'); setDecisionFilter('all'); setDateRange('7'); setPage(0); }}
                data-testid="button-clear-filters"
              >
                <Filter className="h-4 w-4 mr-2" />
                Clear
              </Button>
            }
          >
            <SelectControl
              value={actionFilter}
              onChange={setActionFilter}
              options={[
                { value: 'all', label: 'All Actions' },
                { value: 'create', label: 'Create' },
                { value: 'update', label: 'Update' },
                { value: 'delete', label: 'Delete' },
                { value: 'access', label: 'Access' },
                { value: 'violation', label: 'Violation' },
                { value: 'risk_detected', label: 'Risk Detected' },
                { value: 'rule_triggered', label: 'Rule Triggered' },
                { value: 'tool.exec', label: 'Tool Exec' },
                { value: 'policy_applied', label: 'Policy Applied' },
              ]}
              data-testid="select-action-filter"
            />
            <SelectControl
              value={decisionFilter}
              onChange={setDecisionFilter}
              options={[
                { value: 'all', label: 'All Decisions' },
                { value: 'allow', label: 'Allow' },
                { value: 'deny', label: 'Deny' },
                { value: 'quarantine', label: 'Quarantine' },
              ]}
              data-testid="select-decision-filter"
            />
            <SelectControl
              value={dateRange}
              onChange={setDateRange}
              options={[
                { value: '1', label: 'Last 24 hours' },
                { value: '7', label: 'Last 7 days' },
                { value: '30', label: 'Last 30 days' },
                { value: '90', label: 'Last 90 days' },
              ]}
              data-testid="select-date-range"
            />
          </FilterToolbar>
        </Card>

        {/* Results Summary */}
        <div className="mb-4 sm:mb-6">
          <p className="text-sm text-muted-foreground">
            Showing {filteredEvents.length} of {totalEvents} events
            {searchTerm && ` matching "${searchTerm}"`}
          </p>
        </div>

        {/* Summary Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-6 mb-6">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Events</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{totalEvents}</div>
              <p className="text-xs text-muted-foreground">Last {dateRange} days</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Violations</CardTitle>
              <AlertTriangle className="h-4 w-4 text-orange-600" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-orange-600">
                {auditEvents.filter((e: any) => e.action === 'violation').length}
              </div>
              <p className="text-xs text-muted-foreground">Security violations</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Policy Denials</CardTitle>
              <XCircle className="h-4 w-4 text-red-600" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-red-600">
                {auditEvents.filter((e: any) => (e.metadata?.decision === 'deny')).length}
              </div>
              <p className="text-xs text-muted-foreground">Blocked requests</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Risk Events</CardTitle>
              <Shield className="h-4 w-4 text-red-600" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-red-600">
                {auditEvents.filter((e: any) => e.action === 'risk_detected').length}
              </div>
              <p className="text-xs text-muted-foreground">High-risk detections</p>
            </CardContent>
          </Card>
        </div>

        {/* Audit Events Table */}
        <Card>
          <CardContent className="p-0">
            {isLoading ? (
              <div className="space-y-4 p-4 sm:p-6">
                {Array.from({ length: 10 }).map((_, i) => (
                  <div key={i} className="flex items-center space-x-4">
                    <Skeleton className="h-8 w-8 rounded-full" />
                    <div className="space-y-2 flex-1">
                      <Skeleton className="h-4 w-3/4" />
                      <Skeleton className="h-3 w-1/2" />
                    </div>
                    <Skeleton className="h-6 w-16 rounded-full" />
                    <Skeleton className="h-4 w-16" />
                  </div>
                ))}
              </div>
            ) : filteredEvents.length === 0 ? (
              <div className="text-center py-8 sm:py-12">
                <Activity className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                <h3 className="text-lg font-semibold text-foreground mb-2">No audit events found</h3>
                <p className="text-muted-foreground">
                  {searchTerm || actionFilter !== "all" || decisionFilter !== "all" 
                    ? "Try adjusting your search criteria or filters"
                    : "Audit events will appear here as system activity occurs"
                  }
                </p>
              </div>
            ) : (
              (() => {
                type AuditEventRow = any;
                const columns: Column<AuditEventRow>[] = [
                  { key: 'icon', header: '', className: 'w-12', cell: (e) => getActionIcon(e.action) },
                  {
                    key: 'event', header: 'Event', className: 'min-w-32',
                    cell: (e) => (
                      <div className="space-y-1">
                        <div className="font-medium text-sm text-foreground" data-testid={`event-action-${e.id}`}>
                          {String(e.action || '').replace('_', ' ').toUpperCase()}
                        </div>
                        <div className="text-xs text-muted-foreground">
                          {e.metadata?.reason || 'System Event'}
                        </div>
                      </div>
                    ),
                  },
                  {
                    key: 'user', header: 'User', className: 'min-w-24',
                    cell: (e) => (
                      <div className="flex items-center space-x-2">
                        <User className="h-3 w-3 text-muted-foreground" />
                        <span className="text-sm text-foreground truncate max-w-24" data-testid={`event-user-${e.id}`}>
                          {e.actor_email || (e.actor_id ? String(e.actor_id).substring(0, 8) + '...' : 'System')}
                        </span>
                      </div>
                    ),
                  },
                  {
                    key: 'resource', header: 'Resource', className: 'min-w-32',
                    cell: (e) => (
                      <div className="space-y-1 max-w-32">
                        <div className="text-sm font-medium text-foreground truncate" data-testid={`event-resource-${e.id}`}>
                          {e.metadata?.path || `${e.object_type || ''} ${e.object_id || ''}`.trim() || 'N/A'}
                        </div>
                        <div className="text-xs text-muted-foreground truncate">
                          {e.metadata?.reason || ''}
                        </div>
                      </div>
                    ),
                  },
                  {
                    key: 'decision', header: 'Decision', className: 'min-w-20',
                    cell: (e) => (e.metadata?.decision ? <Badge variant={getDecisionBadge(e.metadata.decision)} data-testid={`event-decision-${e.id}`}>{String(e.metadata.decision).toUpperCase()}</Badge> : null),
                  },
                  {
                    key: 'status', header: 'Status', className: 'min-w-20',
                    cell: (e) => (<span className={"text-sm font-medium " + getStatusColor(e.metadata?.status)} data-testid={`event-status-${e.id}`}>{e.metadata?.status ?? 'N/A'}</span>),
                  },
                  {
                    key: 'time', header: 'Timestamp', className: 'min-w-24',
                    cell: (e) => (
                      <div className="space-y-1">
                        <div className="text-sm text-foreground">{formatTimestamp(e.created_at)}</div>
                        <div className="text-xs text-muted-foreground">{formatTimeAgo(e.created_at)}</div>
                      </div>
                    ),
                  },
                  {
                    key: 'view', header: '', className: 'w-12',
                    cell: (e) => (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setSelectedEvent(e)}
                        data-testid={`button-view-event-${e.id}`}
                      >
                        <Eye className="h-3 w-3" />
                      </Button>
                    ),
                  },
                ];
                return (
                  <DataTable
                    columns={columns}
                    data={filteredEvents}
                    getRowKey={(row) => row.id}
                    isLoading={false}
                  />
                );
              })()
            )}
          </CardContent>
        </Card>

        <Pagination
          page={page}
          totalPages={totalPages}
          totalCount={totalEvents}
          pageSize={limit}
          onPageChange={(p) => setPage(p)}
          onPageSizeChange={() => { /* limit is fixed for now */ }}
          disabled={isLoading}
        />

      </div>
    </Layout>
    <AuditEventModal isOpen={!!selectedEvent} onClose={() => setSelectedEvent(null)} event={selectedEvent} />
    </>
  );
}
