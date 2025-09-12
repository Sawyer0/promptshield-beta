import { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { Layout } from "@/components/Layout";
import { PageHeader } from "@/components/PageHeader";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { useToast } from "@/hooks/use-toast";
import { policyAssignmentApi, rulePackApi, handleApiError } from "@/lib/api";
import { queryClient } from "@/lib/queryClient";
import { AssignmentModal } from "@/components/AssignmentModal";
import { 
  Plus, 
  Trash2, 
  Edit3, 
  MoreVertical,
  ArrowUpDown
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { PolicyAssignment } from "@shared/apiTypes";
import { FilterToolbar } from "@/components/common/FilterToolbar";
import { SearchInput } from "@/components/common/SearchInput";
import { SelectControl } from "@/components/common/SelectControl";

export default function PolicyAssignments() {
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState("");
  const [enabledFilter, setEnabledFilter] = useState("all");
  const [methodFilter, setMethodFilter] = useState("all");
  const [editingAssignment, setEditingAssignment] = useState<PolicyAssignment | null>(null);
  const { toast } = useToast();

  const { data: assignmentsResponse, isLoading } = useQuery({
    queryKey: ["/api/v1/policy-assignments"],
    queryFn: () => policyAssignmentApi.getAll(),
    staleTime: 30 * 1000, // 30 seconds
  });

  const { data: rulePacksResponse } = useQuery({
    queryKey: ["/api/v1/rulepacks"],
    queryFn: () => rulePackApi.getAll(),
    staleTime: 5 * 60 * 1000, // 5 minutes cache
  });

  const toggleAssignmentMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => 
      policyAssignmentApi.update(id, { enabled }),
    onSuccess: () => {
      toast({
        title: "Success",
        description: "Assignment updated successfully!",
      });
      queryClient.invalidateQueries({ queryKey: ["/api/v1/policy-assignments"] });
    },
    onError: (error) => handleApiError(error, toast),
  });

  const deleteAssignmentMutation = useMutation({
    mutationFn: policyAssignmentApi.delete,
    onSuccess: () => {
      toast({
        title: "Success",
        description: "Assignment deleted successfully!",
      });
      queryClient.invalidateQueries({ queryKey: ["/api/v1/policy-assignments"] });
    },
    onError: (error) => handleApiError(error, toast),
  });

  const createAssignmentMutation = useMutation({
    mutationFn: async (payload: any) => {
      if (Array.isArray(payload)) {
        return await policyAssignmentApi.batchCreate(payload);
      }
      return await policyAssignmentApi.create(payload);
    },
    onSuccess: () => {
      toast({
        title: "Success",
        description: "Assignment created successfully!",
      });
      setIsCreateModalOpen(false);
      setEditingAssignment(null);
      queryClient.invalidateQueries({ queryKey: ["/api/v1/policy-assignments"] });
    },
    onError: (error) => handleApiError(error, toast),
  });

  const updateAssignmentMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: any }) => 
      policyAssignmentApi.update(id, data),
    onSuccess: () => {
      toast({
        title: "Success",
        description: "Assignment updated successfully!",
      });
      setEditingAssignment(null);
      queryClient.invalidateQueries({ queryKey: ["/api/v1/policy-assignments"] });
    },
    onError: (error) => handleApiError(error, toast),
  });

  const assignments: PolicyAssignment[] = assignmentsResponse?.data || [];
  const rulePacks = rulePacksResponse?.data || [];

  // Enhanced assignment data with rulepack names
  const enrichedAssignments = assignments.map(assignment => {
    const rulePack = rulePacks.find(rp => rp.id === assignment.rulepackId);
    return {
      ...assignment,
      rulepackName: rulePack?.name || 'Unknown RulePack',
      rulepackVersion: rulePack?.currentVersionId || '1.0',
    };
  });

  const filteredAssignments = enrichedAssignments.filter(assignment => {
    const matchesSearch = !searchTerm || 
      assignment.targetScope.toLowerCase().includes(searchTerm.toLowerCase()) ||
      assignment.rulepackName.toLowerCase().includes(searchTerm.toLowerCase());
    
    const matchesEnabled = enabledFilter === "all" || 
      (enabledFilter === "true" && assignment.enabled) ||
      (enabledFilter === "false" && !assignment.enabled);

    const methodVal = (assignment.method || '*').toUpperCase();
    const matchesMethod = methodFilter === 'all' || methodVal === methodFilter.toUpperCase();
    
    return matchesSearch && matchesEnabled && matchesMethod;
  });

  const formatTimeAgo = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffInHours = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60));
    
    if (diffInHours < 1) return 'Less than an hour ago';
    if (diffInHours < 24) return `${diffInHours} hours ago`;
    if (diffInHours < 168) return `${Math.floor(diffInHours / 24)} days ago`;
    return date.toLocaleDateString();
  };

  const getScopeType = (scope: string) => {
    if (scope === '/') return { type: 'Global Root', color: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200' };
    if (scope.endsWith('/*')) return { type: 'Wildcard', color: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200' };
    if (scope.includes('*')) return { type: 'Pattern', color: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' };
    return { type: 'Exact', color: 'bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200' };
  };

  const handleToggleAssignment = (assignment: PolicyAssignment) => {
    toggleAssignmentMutation.mutate({
      id: assignment.id,
      enabled: !assignment.enabled
    });
  };

  const handleDeleteAssignment = (assignmentId: string) => {
    if (confirm("Are you sure you want to delete this assignment? This action cannot be undone.")) {
      deleteAssignmentMutation.mutate(assignmentId);
    }
  };

  const handleEditAssignment = (assignment: PolicyAssignment) => {
    setEditingAssignment(assignment);
  };

  const handleModalSubmit = (data: any) => {
    if (editingAssignment) {
      const item = Array.isArray(data) ? data[0] : data;
      updateAssignmentMutation.mutate({
        id: editingAssignment.id,
        data: item,
      });
    } else {
      createAssignmentMutation.mutate(data);
    }
  };

  const stats = {
    total: assignments.length,
    active: assignments.filter(a => a.enabled).length,
    inactive: assignments.filter(a => !a.enabled).length,
    scopes: new Set(assignments.map(a => a.targetScope)).size,
  };

  return (
    <Layout title="RulePack Assignments" description="Assign RulePacks to endpoints and manage enforcement">
      <div className="container mx-auto px-4 py-6 sm:py-8">
        <PageHeader
          title="RulePack Assignments"
          subtitle="Assign RulePacks to endpoints and manage enforcement"
          actions={
            <Button onClick={() => setIsCreateModalOpen(true)} data-testid="button-create-assignment" className="w-full sm:w-auto">
              <Plus className="mr-2 h-4 w-4" />
              Assign RulePack
            </Button>
          }
        />
        {/* Filters */}
        <Card className="mb-4 sm:mb-6">
          <CardContent className="p-4 sm:p-6">
            <FilterToolbar
              search={
                <SearchInput
                  value={searchTerm}
                  onChange={setSearchTerm}
                  placeholder="Search scope or rulepack..."
                  data-testid="input-search-assignments"
                />
              }
            >
              <SelectControl
                value={enabledFilter}
                onChange={setEnabledFilter}
                options={[
                  { value: 'all', label: 'All Status' },
                  { value: 'true', label: 'Enabled' },
                  { value: 'false', label: 'Disabled' },
                ]}
                data-testid="select-enabled-filter"
              />
              <SelectControl
                value={methodFilter}
                onChange={setMethodFilter}
                options={[
                  { value: 'all', label: 'All Methods' },
                  { value: '*', label: 'Any (*)' },
                  { value: 'GET', label: 'GET' },
                  { value: 'POST', label: 'POST' },
                  { value: 'PUT', label: 'PUT' },
                  { value: 'PATCH', label: 'PATCH' },
                  { value: 'DELETE', label: 'DELETE' },
                  { value: 'HEAD', label: 'HEAD' },
                  { value: 'OPTIONS', label: 'OPTIONS' },
                ]}
                data-testid="select-method-filter"
              />
            </FilterToolbar>
          </CardContent>
        </Card>

        {/* Results Summary */}
        <div className="mb-4 sm:mb-6">
          <p className="text-sm text-muted-foreground">
            Showing {filteredAssignments.length} of {assignments.length} assignments
            {searchTerm && ` matching "${searchTerm}"`}
          </p>
        </div>

        {/* Assignments Table */}
        <Card>
          <CardContent className="p-0">
            {isLoading ? (
              <div className="space-y-4 p-4 sm:p-6">
                {Array.from({ length: 8 }).map((_, i) => (
                  <div key={i} className="flex items-center space-x-4">
                    <Skeleton className="h-6 w-6 rounded" />
                    <div className="space-y-2 flex-1">
                      <Skeleton className="h-4 w-3/4" />
                      <Skeleton className="h-3 w-1/2" />
                    </div>
                    <Skeleton className="h-6 w-16 rounded-full" />
                    <Skeleton className="h-6 w-12 rounded-full" />
                    <Skeleton className="h-8 w-8" />
                  </div>
                ))}
              </div>
            ) : filteredAssignments.length === 0 ? (
              <div className="text-center py-8 sm:py-12">
                <h3 className="text-lg font-semibold text-foreground mb-2">No RulePack assignments found</h3>
                <p className="text-muted-foreground mb-4">
                  {searchTerm || enabledFilter !== "all" 
                    ? "Try adjusting your search criteria or filters"
                    : "Assign a RulePack to start protecting your API endpoints"
                  }
                </p>
            <Button onClick={() => setIsCreateModalOpen(true)} disabled={!canEdit()} title={!canEdit() ? 'Read-only: you do not have permission to create assignments' : undefined}>
              <Plus className="mr-1 h-4 w-4" /> New Assignment
            </Button>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <Table>
<TableHeader>
                    <TableRow>
                      <TableHead className="min-w-20">Method</TableHead>
                      <TableHead className="min-w-48">Target Scope</TableHead>
                      <TableHead className="min-w-32">RulePack</TableHead>
                      <TableHead className="min-w-20">Precedence</TableHead>
                      <TableHead className="min-w-20">Status</TableHead>
                      <TableHead className="min-w-24">Created</TableHead>
                      <TableHead className="w-12"></TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
{filteredAssignments.map((assignment) => {
                      const scopeInfo = getScopeType(assignment.targetScope);
                      return (
                        <TableRow key={assignment.id} className="hover:bg-muted/50">
                          <TableCell>
                            <Badge variant={(assignment.method || '*') === '*' ? 'secondary' : 'default'} data-testid={`assignment-method-${assignment.id}`}>
                              {(assignment.method || '*') === '*' ? 'Any (*)' : (assignment.method || '*')}
                            </Badge>
                          </TableCell>
                          <TableCell>
                            <div className="space-y-1">
                              <div className="font-mono text-sm text-foreground max-w-48 truncate" data-testid={`assignment-scope-${assignment.id}`}>
                                {assignment.targetScope}
                              </div>
                              <div className="flex items-center space-x-2">
                                <Badge variant="outline" className={scopeInfo.color}>
                                  {scopeInfo.type}
                                </Badge>
                              </div>
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="space-y-1 max-w-32">
                              <div className="text-sm font-medium text-foreground truncate" data-testid={`assignment-rulepack-${assignment.id}`}>
                                {assignment.rulepackName}
                              </div>
                              <div className="text-xs text-muted-foreground">
                                v{assignment.rulepackVersion}
                              </div>
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="flex items-center space-x-2">
                              <ArrowUpDown className="h-3 w-3 text-muted-foreground" />
                              <span className="text-sm font-bold text-foreground" data-testid={`assignment-priority-${assignment.id}`}>
                                {assignment.priority}
                              </span>
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="flex items-center space-x-2">
                              <Switch
                                checked={assignment.enabled}
                                onCheckedChange={() => handleToggleAssignment(assignment)}
                                disabled={toggleAssignmentMutation.isPending}
                                data-testid={`switch-assignment-${assignment.id}`}
                              />
                              <span className={`text-xs font-medium ${assignment.enabled ? 'text-green-600' : 'text-gray-600'}`}>
                                {assignment.enabled ? 'Enabled' : 'Disabled'}
                              </span>
                            </div>
                          </TableCell>
                          <TableCell>
                            <div className="text-sm text-foreground">
                              {assignment.createdAt ? formatTimeAgo(assignment.createdAt.toString()) : 'Unknown'}
                            </div>
                          </TableCell>
                          <TableCell>
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button variant="ghost" size="sm" data-testid={`button-actions-${assignment.id}`}>
                                  <MoreVertical className="h-4 w-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuItem onClick={() => handleEditAssignment(assignment)}>
                                  <Edit3 className="mr-2 h-4 w-4" />
                                  Edit
                                </DropdownMenuItem>
                                <DropdownMenuItem 
                                  onClick={() => handleToggleAssignment(assignment)}
                                  disabled={toggleAssignmentMutation.isPending}
                                >
                                  {assignment.enabled ? 'Disable' : 'Enable'}
                                </DropdownMenuItem>
                                <DropdownMenuItem 
                                  onClick={() => handleDeleteAssignment(assignment.id)}
                                  className="text-red-600"
                                >
                                  <Trash2 className="mr-2 h-4 w-4" />
                                  Delete
                                </DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>

        {/* Assignment Creation/Edit Modal */}
        {(isCreateModalOpen || editingAssignment) && (
          <AssignmentModal
            isOpen={isCreateModalOpen || !!editingAssignment}
            onClose={() => {
              setIsCreateModalOpen(false);
              setEditingAssignment(null);
            }}
            onSubmit={handleModalSubmit}
            rulePacks={rulePacks}
            isLoading={createAssignmentMutation.isPending || updateAssignmentMutation.isPending}
            assignment={editingAssignment}
          />
        )}
      </div>
    </Layout>
  );
}

function hasRole(name: string) {
  try { const raw = localStorage.getItem('ps_roles'); const r = raw ? JSON.parse(raw) : []; return Array.isArray(r) && r.includes(name); } catch { return false; }
}
function canEdit() {
  return hasRole('tenant_admin') || hasRole('security_engineer');
}
