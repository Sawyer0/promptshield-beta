import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Layout } from '@/components/Layout';
import { PageHeader } from '@/components/PageHeader';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
// import { Input } from '@/components/ui/input';
import { SearchInput } from '@/components/common/SearchInput';
import { SelectControl } from '@/components/common/SelectControl';
import { Skeleton } from '@/components/ui/skeleton';
import { useToast } from '@/hooks/use-toast';
import { auditApi, handleApiError } from '@/lib/api';
import { DataTable, type Column } from '@/components/common/DataTable';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Pagination } from '@/components/common/Pagination';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { Separator } from '@/components/ui/separator';
import { FilterToolbar } from '@/components/common/FilterToolbar';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { JSONViewer } from '@/components/common/JSONViewer';
import { 
  AlertTriangle, 
  Shield, 
  Search, 
  Filter, 
  Download, 
  Eye, 
  XCircle,
  Copy,
  ExternalLink,
  Settings,
  Globe,
  User,
  Calendar,
  Info,
  RefreshCw,
  Bookmark,
  Link2,
  Activity
} from 'lucide-react';

interface ViolationEvent {
  id: string;
  timestamp: string;
  action: string;
  userId?: string;
  userName?: string;
  metadata: {
    decision?: 'allow' | 'deny' | 'quarantine';
    reason?: string;
    rule_name?: string;
    path?: string;
    status?: number;
    method?: string;
    violations?: number;
    client_ip?: string;
    request_id?: string;
    rulepack_id?: string;
    tenant_id?: string;
    [key: string]: any;
  };
}

export default function Violations() {
  const [searchTerm, setSearchTerm] = useState("");
  const [decisionFilter, setDecisionFilter] = useState("all");
  // Endpoint/reason inputs removed for minimal UX; use global search instead
  const [endpointFilter] = useState("");
  const [reasonFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [timeRange, setTimeRange] = useState("24h");
  const [pageSize, setPageSize] = useState(50);
  const [page, setPage] = useState(0);
  const [selectedViolation, setSelectedViolation] = useState<ViolationEvent | null>(null);
  const [savedFilters, setSavedFilters] = useState<string | null>(
    localStorage.getItem('violations-filter-preset')
  );
  const { toast } = useToast();

  // Quick tags removed; decision dropdown covers common cases

  const { data: violationsResponse, isLoading, isFetching, refetch } = useQuery({
    queryKey: [
      "/api/v1/audit/violations", 
      { searchTerm, decisionFilter, statusFilter, timeRange, page, pageSize }
    ],
    queryFn: () => auditApi.getAll({
      actions: ["request.decision", "scan.decision", "violation", "risk_detected"],
      search: searchTerm || undefined,
      decision: decisionFilter !== "all" ? decisionFilter : undefined,
      endpoint: endpointFilter || undefined,
      reason: reasonFilter || undefined,
      status: statusFilter !== "all" ? statusFilter : undefined,
      timeRange,
      limit: pageSize,
      offset: page * pageSize,
    }),
    staleTime: 30 * 1000, // 30 seconds
  });

  const violations: ViolationEvent[] = (violationsResponse?.data as any[]) || [];
  const totalViolations = (violationsResponse?.total as number) || violations.length;
  const totalPages = Math.ceil(totalViolations / pageSize);

  const filteredViolations = useMemo(() => {
    return violations.filter(violation => {
      // Apply search and filters
      const matchesSearch = !searchTerm || 
        violation.metadata?.path?.toLowerCase().includes(searchTerm.toLowerCase()) ||
        violation.metadata?.reason?.toLowerCase().includes(searchTerm.toLowerCase()) ||
        violation.metadata?.rule_name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
        violation.action.toLowerCase().includes(searchTerm.toLowerCase());
      
      const matchesDecision = decisionFilter === "all" || violation.metadata?.decision === decisionFilter;
      const matchesEndpoint = !endpointFilter || violation.metadata?.path?.includes(endpointFilter);
      const matchesReason = !reasonFilter || 
        violation.metadata?.reason?.toLowerCase().includes(reasonFilter.toLowerCase()) ||
        violation.metadata?.rule_name?.toLowerCase().includes(reasonFilter.toLowerCase());
      const s = violation.metadata?.status;
      const matchesStatus = statusFilter === "all" || (
        s != null && (
          (statusFilter === "2xx" && s >= 200 && s < 300) ||
          (statusFilter === "4xx" && s >= 400 && s < 500) ||
          (statusFilter === "5xx" && s >= 500)
        )
      );
      
      return matchesSearch && matchesDecision && matchesEndpoint && matchesStatus;
    });
  }, [violations, searchTerm, decisionFilter, endpointFilter, statusFilter]);

  const stats = useMemo(() => {
    if (!Array.isArray(violations)) {
      return { deniedCount: 0, quarantinedCount: 0, blockedCount: 0, avgResponseTime: 0 };
    }
    
    const deniedCount = violations.filter(v => v.metadata?.decision === 'deny').length;
    const quarantinedCount = violations.filter(v => v.metadata?.decision === 'quarantine').length;
    const blockedCount = violations.filter(v => v.action === 'violation' || v.action === 'risk_detected').length;
    const avgResponseTime = violations.reduce((acc, v) => acc + (v.metadata?.duration || 0), 0) / violations.length || 0;
    
    return { deniedCount, quarantinedCount, blockedCount, avgResponseTime };
  }, [violations]);

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false
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

  const getDecisionBadge = (decision?: string) => {
    switch (decision?.toLowerCase()) {
      case 'deny':
        return <Badge variant="destructive" className="bg-red-600 text-white">Deny</Badge>;
      case 'quarantine':
        return <Badge variant="secondary" className="bg-amber-600 text-white">Quarantine</Badge>;
      case 'allow':
        return <Badge variant="default" className="bg-green-600 text-white">Allow</Badge>;
      default:
        return <Badge variant="outline">Unknown</Badge>;
    }
  };

  const getStatusColor = (status?: number) => {
    if (!status) return 'text-gray-600';
    if (status >= 200 && status < 300) return 'text-green-600';
    if (status >= 300 && status < 400) return 'text-blue-600';
    if (status >= 400 && status < 500) return 'text-orange-600';
    if (status >= 500) return 'text-red-600';
    return 'text-gray-600';
  };

  const handleExportViolations = async (format: 'json' | 'csv') => {
    try {
      const exportData = await auditApi.getAll({
        actions: ["request.decision", "scan.decision", "violation", "risk_detected"],
        search: searchTerm || undefined,
        decision: decisionFilter !== "all" ? decisionFilter : undefined,
        // endpoint/reason removed from explicit filters; captured by search
        timeRange,
        format,
      });
      
      const blob = new Blob([
        format === 'json' 
          ? JSON.stringify(exportData, null, 2)
          : convertToCSV((exportData.data as any[]) || [])
      ], { 
        type: format === 'json' ? 'application/json' : 'text/csv' 
      });
      
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `violations-${new Date().toISOString().split('T')[0]}.${format}`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
      
      toast({
        title: "Export Complete",
        description: `Violations exported as ${format.toUpperCase()} successfully!`,
      });
    } catch (error) {
      handleApiError(error as Error, toast);
    }
  };

  const convertToCSV = (data: ViolationEvent[]) => {
    const headers = ['Timestamp', 'Decision', 'Endpoint', 'Reason', 'Status', 'User', 'Violations'];
    const rows = data.map(v => [
      v.timestamp,
      v.metadata?.decision || '',
      v.metadata?.path || '',
      v.metadata?.reason || v.metadata?.rule_name || '',
      v.metadata?.status || '',
      v.userName || v.metadata?.user_email || '',
      v.metadata?.violations || 0
    ]);
    
    return [headers, ...rows].map(row => 
      row.map(cell => `"${String(cell).replace(/"/g, '""')}"`).join(',')
    ).join('\n');
  };

  const saveFilterPreset = () => {
    const preset = { decisionFilter, statusFilter, timeRange };
    localStorage.setItem('violations-filter-preset', JSON.stringify(preset));
    setSavedFilters(JSON.stringify(preset));
    toast({
      title: "Filter Preset Saved",
      description: "Your current filters have been saved for future sessions.",
    });
  };

  const loadFilterPreset = () => {
    if (savedFilters) {
      const preset = JSON.parse(savedFilters);
      setDecisionFilter(preset.decisionFilter || "all");
      // endpoint/reason removed
      setStatusFilter(preset.statusFilter || "all");
      setTimeRange(preset.timeRange || "24h");
      toast({
        title: "Filter Preset Loaded",
        description: "Your saved filters have been applied.",
      });
    }
  };

  const clearAllFilters = () => {
    setSearchTerm("");
    setDecisionFilter("all");
    // endpoint/reason removed
    setStatusFilter("all");
    setTimeRange("24h");
    // quick filter removed
    setPage(0);
  };

  const copyEventLink = (violation: ViolationEvent) => {
    const url = new URL(window.location.href);
    url.searchParams.set('event', violation.id);
    url.searchParams.set('decision', decisionFilter);
    url.searchParams.set('timeRange', timeRange);
    
    navigator.clipboard.writeText(url.toString());
    toast({
      title: "Link Copied",
      description: "Event permalink copied to clipboard.",
    });
  };

  return (
    <Layout title="Security Violations" description="Clear visibility into blocked, denied, and quarantined traffic">
      <div className="container mx-auto px-4 py-6 sm:py-8">
        <PageHeader
          title="Security Violations"
          subtitle="Clear visibility into blocked, denied, and quarantined traffic with triage tools"
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
              <Button 
                onClick={() => handleExportViolations('json')} 
                variant="outline" 
                data-testid="button-export"
                className="w-full sm:w-auto"
              >
                <Download className="h-4 w-4 mr-2" />
                Export
              </Button>
            </>
          }
        />

        {/* Summary Cards (same style as Audit Logs) */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-6 mb-6">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Total Events</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{totalViolations}</div>
              <p className="text-xs text-muted-foreground">Last {timeRange}</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">Violations</CardTitle>
              <AlertTriangle className="h-4 w-4 text-orange-600" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-orange-600">
                {violations.filter(v => v.action === 'violation').length}
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
                {violations.filter(v => v.metadata?.decision === 'deny').length}
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
                {violations.filter(v => v.action === 'risk_detected').length}
              </div>
              <p className="text-xs text-muted-foreground">High-risk detections</p>
            </CardContent>
          </Card>
        </div>

        {/* Consolidated Filters */}
        <Card className="mb-4 sm:mb-6">
          <FilterToolbar
            search={
              <SearchInput
                value={searchTerm}
                onChange={setSearchTerm}
                placeholder="Search violations by endpoint, rule, reason, or action..."
                data-testid="input-search-violations"
              />
            }
            actions={
              <div className="flex items-center gap-2 ml-2">
                <Button variant="outline" onClick={saveFilterPreset} data-testid="button-save-view" className="ml-2"><Bookmark className="h-3 w-3 mr-1" />Save View</Button>
                {savedFilters && <Button variant="outline" onClick={loadFilterPreset} data-testid="button-load-view"><Settings className="h-3 w-3 mr-1" />Load Saved</Button>}
                <Button variant="outline" onClick={clearAllFilters} data-testid="button-clear-filters"><Filter className="h-4 w-4 mr-2" />Clear</Button>
              </div>
            }
          >
            <SelectControl
              value={timeRange}
              onChange={setTimeRange}
              options={[
                { value: '1h', label: 'Last 1 Hour' },
                { value: '24h', label: 'Last 24 Hours' },
                { value: '7d', label: 'Last 7 Days' },
                { value: '30d', label: 'Last 30 Days' },
              ]}
              data-testid="select-time-range"
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
            {/* Removed extra endpoint/reason inputs for minimal UX */}
            <SelectControl
              value={statusFilter}
              onChange={setStatusFilter}
              options={[
                { value: 'all', label: 'All Status' },
                { value: '2xx', label: '2xx Success' },
                { value: '4xx', label: '4xx Client Error' },
                { value: '5xx', label: '5xx Server Error' },
              ]}
              data-testid="select-status-filter"
            />
          </FilterToolbar>
        </Card>

        {/* Results Summary */}
        <div className="mb-4 sm:mb-6">
          <p className="text-sm text-muted-foreground">
            Showing {filteredViolations.length} of {totalViolations} violations
            {searchTerm && ` matching "${searchTerm}"`}
          </p>
        </div>

        {/* Violations Table */}
        <Card>
          <CardContent className="p-0">
            {isLoading ? (
              <div className="space-y-4 p-4 sm:p-6">
                {Array.from({ length: pageSize }).map((_, i) => (
                  <div key={i} className="flex items-center space-x-4">
                    <Skeleton className="h-8 w-24" />
                    <Skeleton className="h-6 w-20" />
                    <Skeleton className="h-4 w-32" />
                    <Skeleton className="h-4 w-24" />
                    <Skeleton className="h-4 w-16" />
                    <Skeleton className="h-4 w-20" />
                    <Skeleton className="h-8 w-8" />
                  </div>
                ))}
              </div>
            ) : filteredViolations.length === 0 ? (
              <div className="text-center py-8 sm:py-12">
                <Shield className="h-12 w-12 text-green-500 mx-auto mb-4" />
                <h3 className="text-lg font-semibold text-foreground mb-2">All Clear</h3>
                <p className="text-muted-foreground mb-4">
                  {searchTerm || decisionFilter !== "all" || endpointFilter || reasonFilter || statusFilter !== "all" 
                    ? "No violations match your current filters - try adjusting the time range or criteria"
                    : "No security violations found in the selected time range. Your systems are secure!"
                  }
                </p>
                <Button variant="outline" onClick={clearAllFilters}>
                  <Filter className="h-4 w-4 mr-2" />
                  Clear Filters
                </Button>
              </div>
            ) : (
              (() => {
                const columns: Column<ViolationEvent>[] = [
                  {
                    key: 'time',
                    header: 'Timestamp',
                    className: 'min-w-32',
                    cell: (v) => (
                      <div className="space-y-1">
                        <div className="text-sm font-mono text-foreground">{formatTimestamp(v.timestamp)}</div>
                        <div className="text-xs text-muted-foreground">{formatTimeAgo(v.timestamp)}</div>
                      </div>
                    ),
                  },
                  {
                    key: 'decision',
                    header: 'Decision',
                    className: 'min-w-24',
                    cell: (v) => getDecisionBadge(v.metadata?.decision),
                  },
                  {
                    key: 'endpoint',
                    header: 'Endpoint Path',
                    className: 'min-w-48',
                    cell: (v) => (
                      <div className="space-y-1">
                        <div className="font-mono text-sm text-foreground max-w-48 truncate" data-testid={`violation-path-${v.id}`}>
                          {v.metadata?.path || 'Unknown endpoint'}
                        </div>
                        <div className="text-xs text-muted-foreground">{v.metadata?.method || 'N/A'}</div>
                      </div>
                    ),
                  },
                  {
                    key: 'reason',
                    header: 'Rule/Reason',
                    className: 'min-w-32',
                    cell: (v) => (
                      <div className="space-y-1 max-w-32">
                        <div className="text-sm text-foreground truncate" data-testid={`violation-reason-${v.id}`}>
                          {v.metadata?.rule_name || v.metadata?.reason || 'Security policy'}
                        </div>
                        <div className="text-xs text-muted-foreground truncate">{v.action.replace('_', ' ')}</div>
                      </div>
                    ),
                  },
                  {
                    key: 'status',
                    header: 'Status',
                    className: 'min-w-16',
                    cell: (v) => (
                      <span className={"text-sm font-medium " + getStatusColor(v.metadata?.status)} data-testid={`violation-status-${v.id}`}>
                        {v.metadata?.status || 'N/A'}
                      </span>
                    ),
                  },
                  {
                    key: 'user',
                    header: 'User',
                    className: 'min-w-24',
                    cell: (v) => (
                      <div className="flex items-center space-x-2 max-w-24">
                        <User className="h-3 w-3 text-muted-foreground" />
                        <span className="text-sm text-foreground truncate" data-testid={`violation-user-${v.id}`}>
                          {v.userName || v.userId || 'System'}
                        </span>
                      </div>
                    ),
                  },
                  {
                    key: 'req',
                    header: 'Req ID',
                    className: 'min-w-24',
                    cell: (v) => (
                      <div className="flex items-center gap-2">
                        <span className="text-xs font-mono truncate max-w-24">{v.metadata?.request_id || '—'}</span>
                        {v.metadata?.request_id && (
                          <Button variant="ghost" size="sm" onClick={() => navigator.clipboard.writeText(String(v.metadata?.request_id || ''))}>
                            <Copy className="h-3 w-3" />
                          </Button>
                        )}
                      </div>
                    ),
                  },
                  {
                    key: 'count',
                    header: 'Violations',
                    className: 'min-w-16',
                    cell: (v) => (
                      <div className="text-center">
                        <span className="text-sm font-bold text-red-600" data-testid={`violation-count-${v.id}`}>
                          {v.metadata?.violations || 1}
                        </span>
                      </div>
                    ),
                  },
                  {
                    key: 'action',
                    header: '',
                    className: 'w-12',
                    cell: (v) => (
                      <Button variant="ghost" size="sm" onClick={() => setSelectedViolation(v)} data-testid={`button-view-violation-${v.id}`}>
                        <Eye className="h-3 w-3" />
                      </Button>
                    ),
                  },
                ];
                return (
                  <DataTable
                    columns={columns}
                    data={filteredViolations}
                    getRowKey={(row) => row.id}
                    isLoading={false}
                    emptyMessage={
                      <div className="text-center py-8 sm:py-12">
                        <Shield className="h-12 w-12 text-green-500 mx-auto mb-4" />
                        <h3 className="text-lg font-semibold text-foreground mb-2">All Clear</h3>
                        <p className="text-muted-foreground mb-4">
                          {searchTerm || decisionFilter !== 'all' || endpointFilter || reasonFilter || statusFilter !== 'all' 
                            ? 'No violations match your current filters - try adjusting the time range or criteria'
                            : 'No security violations found in the selected time range. Your systems are secure!'}
                        </p>
                        <Button variant="outline" onClick={clearAllFilters}>
                          <Filter className="h-4 w-4 mr-2" />
                          Clear Filters
                        </Button>
                      </div>
                    }
                  />
                );
              })()
            )}
          </CardContent>
        </Card>

        <Pagination
          page={page}
          totalPages={totalPages}
          totalCount={totalViolations}
          pageSize={pageSize}
          onPageChange={(p) => setPage(p)}
          onPageSizeChange={(s) => setPageSize(s)}
          disabled={isLoading}
        />

        {/* Violation Details Drawer */}
        <Sheet open={!!selectedViolation} onOpenChange={() => setSelectedViolation(null)}>
          <SheetContent className="w-full sm:max-w-2xl overflow-y-auto">
            <SheetHeader>
              <SheetTitle className="flex items-center gap-2">
                <AlertTriangle className="h-5 w-5 text-red-600" />
                Security Event Details
              </SheetTitle>
              <SheetDescription>
                Complete information about this security violation and context
              </SheetDescription>
            </SheetHeader>

            {selectedViolation && (
              <div className="space-y-6 mt-6">
                {/* Summary */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Event Summary</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <span className="text-sm font-medium text-muted-foreground">Decision</span>
                        <div className="mt-1">{getDecisionBadge(selectedViolation.metadata?.decision)}</div>
                      </div>
                      <div>
                        <span className="text-sm font-medium text-muted-foreground">Timestamp</span>
                        <div className="mt-1 text-sm font-mono">{formatTimestamp(selectedViolation.timestamp)}</div>
                      </div>
                      <div>
                        <span className="text-sm font-medium text-muted-foreground">Endpoint</span>
                        <div className="mt-1 text-sm font-mono bg-muted px-2 py-1 rounded">
                          {selectedViolation.metadata?.path || 'Unknown'}
                        </div>
                      </div>
                      <div>
                        <span className="text-sm font-medium text-muted-foreground">HTTP Status</span>
                        <div className={`mt-1 text-sm font-medium ${getStatusColor(selectedViolation.metadata?.status)}`}>
                          {selectedViolation.metadata?.status || 'N/A'}
                        </div>
                      </div>
                    </div>
                    
                    <Separator />
                    
                    <div>
                      <span className="text-sm font-medium text-muted-foreground">Rule/Reason</span>
                      <div className="mt-1 text-sm">
                        {selectedViolation.metadata?.rule_name || selectedViolation.metadata?.reason || 'Security policy violation'}
                      </div>
                    </div>
                  </CardContent>
                </Card>

                {/* Request Context */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Request Context</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <span className="text-sm font-medium text-muted-foreground">Method</span>
                        <div className="mt-1 text-sm font-mono">{selectedViolation.metadata?.method || 'N/A'}</div>
                      </div>
                      <div>
                        <span className="text-sm font-medium text-muted-foreground">Client IP</span>
                        <div className="mt-1 text-sm font-mono">{selectedViolation.metadata?.client_ip || 'Unknown'}</div>
                      </div>
                    </div>
                    
                    {selectedViolation.metadata?.request_id && (
                      <div>
                        <span className="text-sm font-medium text-muted-foreground">Request ID</span>
                        <div className="mt-1 text-sm font-mono bg-muted px-2 py-1 rounded">
                          {selectedViolation.metadata.request_id}
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>

                {/* Matched Signals */}
                {(((selectedViolation.metadata?.violations || 0) > 0) || !!selectedViolation.metadata?.rule_name) && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-base">Matched Signals</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="space-y-2">
                        {selectedViolation.metadata?.rule_name && (
                          <div className="flex items-center space-x-2">
                            <Badge variant="destructive" className="text-xs">
                              {selectedViolation.metadata.rule_name}
                            </Badge>
                            <span className="text-sm text-muted-foreground">
                              {selectedViolation.metadata?.reason || 'Security rule triggered'}
                            </span>
                          </div>
                        )}
                        <div className="text-sm text-muted-foreground">
                          {selectedViolation.metadata?.violations || 1} violation(s) detected
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                )}

                {/* Raw Event JSON */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Raw Event Data</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <JSONViewer data={selectedViolation} />
                  </CardContent>
                </Card>

                {/* Action Links */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Related Actions</CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <Button 
                      variant="outline" 
                      className="w-full justify-start" 
                      onClick={() => copyEventLink(selectedViolation)}
                      data-testid="button-copy-permalink"
                    >
                      <Copy className="h-4 w-4 mr-2" />
                      Copy Event Permalink
                    </Button>
                    
                    <Button 
                      variant="outline" 
                      className="w-full justify-start"
                      onClick={() => {
                        // Navigate to audit logs with this event
                        window.open(`/audit?event=${selectedViolation.id}`, '_blank');
                      }}
                      data-testid="button-open-audits"
                    >
                      <ExternalLink className="h-4 w-4 mr-2" />
                      Open in Audit Logs
                    </Button>
                    
                    {selectedViolation.metadata?.rulepack_id && (
                      <Button 
                        variant="outline" 
                        className="w-full justify-start"
                        onClick={() => {
                          window.open(`/rulepacks/${selectedViolation.metadata.rulepack_id}`, '_blank');
                        }}
                        data-testid="button-related-rulepack"
                      >
                        <Link2 className="h-4 w-4 mr-2" />
                        Related RulePack
                      </Button>
                    )}
                    
                    {selectedViolation.metadata?.path && (
                      <Button 
                        variant="outline" 
                        className="w-full justify-start"
                        onClick={() => {
                          const pathPrefix = String(selectedViolation.metadata?.path || '').split('/').slice(0, -1).join('/') + '/*';
                          window.open(`/policies?scope=${encodeURIComponent(pathPrefix)}`, '_blank');
                        }}
                        data-testid="button-scope-assignments"
                      >
                        <Globe className="h-4 w-4 mr-2" />
                        Scope Assignments
                      </Button>
                    )}
                  </CardContent>
                </Card>
              </div>
            )}
          </SheetContent>
        </Sheet>
      </div>
    </Layout>
  );
}
