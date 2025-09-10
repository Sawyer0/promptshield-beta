import { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { Layout } from "@/components/Layout";
import { RulePackModal } from "@/components/RulePackModal";
import { RulePackDetailsModal } from "@/components/RulePackDetailsModal";
import { Button } from "@/components/ui/button";
import { SearchInput } from "@/components/common/SearchInput";
import { SelectControl } from "@/components/common/SelectControl";
import { FilterToolbar } from "@/components/common/FilterToolbar";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { useToast } from "@/hooks/use-toast";
import { rulePackApi, handleApiError } from "@/lib/api";
import { queryClient } from "@/lib/queryClient";
import { Plus, Search, Edit, Copy, Trash2, Power, PowerOff, Eye } from "lucide-react";
import type { RulePack } from "@shared/apiTypes";
import { PageHeader } from "@/components/PageHeader";

export default function RulePacks() {
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isDetailsModalOpen, setIsDetailsModalOpen] = useState(false);
  const [selectedRulePack, setSelectedRulePack] = useState<RulePack | null>(null);
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [severityFilter, setSeverityFilter] = useState("all");
  const { toast } = useToast();

  const { data: rulePacksResponse, isLoading } = useQuery({
    queryKey: ["/api/rulepacks"],
    queryFn: () => rulePackApi.getAll(),
    staleTime: 2 * 60 * 1000, // 2 minutes cache for better navigation
  });

  const createMutation = useMutation({
    mutationFn: rulePackApi.create,
    onSuccess: () => {
      toast({
        title: "Success",
        description: "RulePack created successfully!",
      });
      setIsCreateModalOpen(false);
      queryClient.invalidateQueries({ queryKey: ["/api/rulepacks"] });
    },
    onError: (error) => handleApiError(error, toast),
  });

  const deleteMutation = useMutation({
    mutationFn: rulePackApi.delete,
    onSuccess: () => {
      toast({
        title: "Success",
        description: "RulePack deleted successfully!",
      });
      queryClient.invalidateQueries({ queryKey: ["/api/rulepacks"] });
    },
    onError: (error) => handleApiError(error, toast),
  });

  const activateMutation = useMutation({
    mutationFn: rulePackApi.activateLatest,
    onSuccess: () => {
      toast({
        title: "Success",
        description: "RulePack activated successfully!",
      });
      queryClient.invalidateQueries({ queryKey: ["/api/rulepacks"] });
    },
    onError: (error) => handleApiError(error, toast),
  });

  const deactivateMutation = useMutation({
    mutationFn: rulePackApi.deactivate,
    onSuccess: () => {
      toast({
        title: "Success",
        description: "RulePack deactivated successfully!",
      });
      queryClient.invalidateQueries({ queryKey: ["/api/rulepacks"] });
    },
    onError: (error) => handleApiError(error, toast),
  });

  const rulePacks = rulePacksResponse?.data || [];

  const filteredRulePacks = rulePacks.filter((rp) => {
    const matchesSearch = rp.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         rp.description?.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus = statusFilter === "all" || 
                         (statusFilter === "active" && rp.isActive) ||
                         (statusFilter === "inactive" && !rp.isActive);
    
    return matchesSearch && matchesStatus;
  });

  const handleCreate = async (data: any) => {
    createMutation.mutate(data);
  };

  const handleDelete = (id: string) => {
    if (window.confirm("Are you sure you want to delete this RulePack?")) {
      deleteMutation.mutate(id);
    }
  };

  const handleToggleStatus = (rulePack: RulePack) => {
    if (rulePack.isActive) {
      deactivateMutation.mutate(rulePack.id);
    } else {
      activateMutation.mutate(rulePack.id);
    }
  };

  const handleViewDetails = (rulePack: RulePack) => {
    setSelectedRulePack(rulePack);
    setIsDetailsModalOpen(true);
  };

  const handleCloseDetails = () => {
    setIsDetailsModalOpen(false);
    setSelectedRulePack(null);
  };

  const formatTimeAgo = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffInHours = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60));
    
    if (diffInHours < 1) return 'Less than an hour ago';
    if (diffInHours < 24) return `${diffInHours} hours ago`;
    if (diffInHours < 168) return `${Math.floor(diffInHours / 24)} days ago`;
    return date.toLocaleDateString();
  };

  const getRuleCount = (rulePack: RulePack) => {
    if (Array.isArray(rulePack.rules)) {
      return rulePack.rules.length;
    }
    return 0;
  };

  return (
    <Layout 
      title="RulePacks" 
      description="Manage security rules and policies"
    >
      <PageHeader
        title="RulePacks"
        subtitle="Manage security rules and policies"
        actions={
          <Button 
            onClick={() => setIsCreateModalOpen(true)}
            data-testid="button-create-rulepack"
            className="flex-1 sm:flex-none"
          >
            <Plus className="mr-2 h-4 w-4" />
            <span className="sm:inline">Create RulePack</span>
          </Button>
        }
      />

      {/* Search and Filters */}
      <Card className="mb-4 sm:mb-6">
        <FilterToolbar
          search={
            <SearchInput
              value={searchTerm}
              onChange={setSearchTerm}
              placeholder="Search rulepacks..."
              data-testid="input-search-rulepacks"
            />
          }
        >
          <SelectControl
            value={statusFilter}
            onChange={setStatusFilter}
            options={[
              { value: 'all', label: 'All Status' },
              { value: 'active', label: 'Active' },
              { value: 'inactive', label: 'Inactive' },
            ]}
            data-testid="select-status-filter"
          />
          <SelectControl
            value={severityFilter}
            onChange={setSeverityFilter}
            options={[
              { value: 'all', label: 'All Severity' },
              { value: 'critical', label: 'Critical' },
              { value: 'high', label: 'High' },
              { value: 'medium', label: 'Medium' },
              { value: 'low', label: 'Low' },
            ]}
            data-testid="select-severity-filter"
          />
        </FilterToolbar>
      </Card>

      {/* RulePacks Grid */}
      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
          {Array.from({ length: 6 }).map((_, i) => (
            <Card key={i}>
              <CardContent className="p-4 sm:p-6">
                <Skeleton className="h-6 w-3/4 mb-2" />
                <Skeleton className="h-4 w-full mb-4" />
                <div className="flex justify-between items-center mb-4">
                  <Skeleton className="h-4 w-16" />
                  <Skeleton className="h-6 w-16 rounded-full" />
                </div>
                <div className="flex space-x-2">
                  <Skeleton className="h-8 flex-1" />
                  <Skeleton className="h-8 flex-1" />
                  <Skeleton className="h-8 flex-1" />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : filteredRulePacks.length === 0 ? (
        <Card>
          <CardContent className="p-12 text-center">
            <p className="text-muted-foreground">
              {searchTerm || statusFilter !== "all" ? "No rulepacks match your filters" : "No rulepacks found"}
            </p>
            {!searchTerm && statusFilter === "all" && (
              <Button 
                onClick={() => setIsCreateModalOpen(true)} 
                className="mt-4"
                data-testid="button-create-first-rulepack"
              >
                <Plus className="mr-2 h-4 w-4" />
                Create Your First RulePack
              </Button>
            )}
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
          {filteredRulePacks.map((rulePack) => (
            <Card key={rulePack.id} className="hover:shadow-md transition-shadow" data-testid={`rulepack-card-${rulePack.id}`}>
              <CardContent className="p-4 sm:p-6">
                <div className="flex flex-col sm:flex-row sm:items-start justify-between mb-4 gap-3 sm:gap-4">
                  <div className="flex-1 min-w-0">
                    <h3 className="text-base sm:text-lg font-semibold text-foreground mb-1 truncate" data-testid={`rulepack-name-${rulePack.id}`}>
                      {rulePack.name}
                    </h3>
                    <p className="text-sm text-muted-foreground line-clamp-2">
                      {rulePack.description || "No description available"}
                    </p>
                  </div>
                  <Badge 
                    variant={rulePack.isActive ? "default" : "secondary"}
                    data-testid={`rulepack-status-${rulePack.id}`}
                    className="self-start"
                  >
                    {rulePack.isActive ? "Active" : "Inactive"}
                  </Badge>
                </div>
                
                <div className="space-y-2 mb-4 text-sm">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Version:</span>
                    <span className="font-medium" data-testid={`rulepack-version-${rulePack.id}`}>
                      {rulePack.currentVersionId ? rulePack.currentVersionId.substring(0, 8) : '1.0'}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Rules:</span>
                    <span className="font-medium" data-testid={`rulepack-rules-count-${rulePack.id}`}>
                      {getRuleCount(rulePack)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Updated:</span>
                    <span className="font-medium">
                      {formatTimeAgo(
                        typeof rulePack.updatedAt === 'string'
                          ? rulePack.updatedAt
                          : ((rulePack.updatedAt as unknown as Date | undefined)?.toISOString() || new Date().toISOString())
                      )}
                    </span>
                  </div>
                </div>
                
                <div className="flex flex-col sm:flex-row gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleViewDetails(rulePack)}
                    data-testid={`button-view-${rulePack.id}`}
                    className="w-full sm:w-auto"
                  >
                    <Eye className="h-4 w-4 sm:mr-0" />
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleToggleStatus(rulePack)}
                    disabled={activateMutation.isPending || deactivateMutation.isPending}
                    data-testid={`button-toggle-status-${rulePack.id}`}
                    className="w-full sm:w-auto"
                  >
                    {rulePack.isActive ? (
                      <PowerOff className="h-4 w-4 sm:mr-0" />
                    ) : (
                      <Power className="h-4 w-4 sm:mr-0" />
                    )}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleDelete(rulePack.id)}
                    disabled={deleteMutation.isPending}
                    data-testid={`button-delete-${rulePack.id}`}
                    className="w-full sm:w-auto"
                  >
                    <Trash2 className="h-4 w-4 sm:mr-0" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <RulePackModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onSubmit={handleCreate}
        isLoading={createMutation.isPending}
      />
      
      <RulePackDetailsModal
        isOpen={isDetailsModalOpen}
        onClose={handleCloseDetails}
        rulepack={selectedRulePack}
      />
    </Layout>
  );
}
