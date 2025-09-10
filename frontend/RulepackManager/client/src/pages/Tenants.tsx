import { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Layout } from "@/components/Layout";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
import { Switch } from "@/components/ui/switch";
import { useToast } from "@/hooks/use-toast";
import { tenantApi, handleApiError } from "@/lib/api";
import { queryClient } from "@/lib/queryClient";
import { Plus, Building, Edit, Trash2, Users } from "lucide-react";
import type { Tenant, InsertTenant } from "@shared/apiTypes";
import { PageHeader } from "@/components/PageHeader";

const tenantSchema = z.object({
  name: z.string().min(1, "Tenant name is required"),
  enabled: z.boolean().default(true),
});

type TenantFormData = z.infer<typeof tenantSchema>;

export default function Tenants() {
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null);
  const { toast } = useToast();

  const { data: tenantsResponse, isLoading } = useQuery({
    queryKey: ["/api/tenants"],
    queryFn: () => tenantApi.getAll(),
    staleTime: 2 * 60 * 1000, // 2 minutes cache for better navigation
  });

  const createMutation = useMutation({
    mutationFn: tenantApi.create,
    onSuccess: () => {
      toast({
        title: "Success",
        description: "Tenant created successfully!",
      });
      setIsCreateModalOpen(false);
      queryClient.invalidateQueries({ queryKey: ["/api/tenants"] });
    },
    onError: (error) => handleApiError(error, toast),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<InsertTenant> }) => 
      tenantApi.update(id, data),
    onSuccess: () => {
      toast({
        title: "Success",
        description: "Tenant updated successfully!",
      });
      setEditingTenant(null);
      queryClient.invalidateQueries({ queryKey: ["/api/tenants"] });
    },
    onError: (error) => handleApiError(error, toast),
  });

  const deleteMutation = useMutation({
    mutationFn: tenantApi.delete,
    onSuccess: () => {
      toast({
        title: "Success",
        description: "Tenant deleted successfully!",
      });
      queryClient.invalidateQueries({ queryKey: ["/api/tenants"] });
    },
    onError: (error) => handleApiError(error, toast),
  });

  const form = useForm<TenantFormData>({
    resolver: zodResolver(tenantSchema),
    defaultValues: {
      name: "",
      enabled: true,
    },
  });

  const tenants: Tenant[] = tenantsResponse?.data || [];

  const handleCreate = (data: TenantFormData) => {
    createMutation.mutate(data);
  };

  const handleUpdate = (data: TenantFormData) => {
    if (editingTenant) {
      updateMutation.mutate({ id: editingTenant.id, data });
    }
  };

  const handleDelete = (id: string) => {
    if (window.confirm("Are you sure you want to delete this tenant?")) {
      deleteMutation.mutate(id);
    }
  };

  const openCreateModal = () => {
    form.reset({ name: "", enabled: true });
    setIsCreateModalOpen(true);
  };

  const openEditModal = (tenant: Tenant) => {
    form.reset({ 
      name: tenant.name, 
      enabled: tenant.enabled ?? true
    });
    setEditingTenant(tenant);
  };

  const closeModal = () => {
    setIsCreateModalOpen(false);
    setEditingTenant(null);
    form.reset();
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

  const isModalOpen = isCreateModalOpen || !!editingTenant;
  const isEditing = !!editingTenant;

  return (
    <Layout 
      title="Tenants" 
      description="Manage tenant organizations and access"
    >
      <PageHeader
        title="Tenants"
        subtitle="Manage tenant organizations and access"
        actions={
          <Button onClick={openCreateModal} data-testid="button-create-tenant" className="w-full sm:w-auto">
            <Plus className="mr-2 h-4 w-4" />
            Add Tenant
          </Button>
        }
      />

      {/* Tenants Grid */}
      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
          {Array.from({ length: 6 }).map((_, i) => (
            <Card key={i}>
              <CardContent className="p-6">
                <div className="flex items-center justify-between mb-4">
                  <Skeleton className="h-12 w-12 rounded-lg" />
                  <Skeleton className="h-6 w-16 rounded-full" />
                </div>
                <Skeleton className="h-6 w-3/4 mb-2" />
                <Skeleton className="h-4 w-full mb-4" />
                <div className="space-y-2 mb-4">
                  <div className="flex justify-between">
                    <Skeleton className="h-4 w-20" />
                    <Skeleton className="h-4 w-8" />
                  </div>
                  <div className="flex justify-between">
                    <Skeleton className="h-4 w-24" />
                    <Skeleton className="h-4 w-16" />
                  </div>
                </div>
                <div className="flex space-x-2">
                  <Skeleton className="h-8 flex-1" />
                  <Skeleton className="h-8 flex-1" />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : tenants.length === 0 ? (
        <Card>
          <CardContent className="p-12 text-center">
            <Building className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-muted-foreground mb-4">No tenants found</p>
            <Button onClick={openCreateModal} data-testid="button-create-first-tenant">
              <Plus className="mr-2 h-4 w-4" />
              Create Your First Tenant
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
          {tenants.map((tenant: Tenant) => (
            <Card key={tenant.id} className="hover:shadow-md transition-shadow" data-testid={`tenant-card-${tenant.id}`}>
              <CardContent className="p-4 sm:p-6">
                <div className="flex items-center justify-between mb-4">
                  <div className="h-10 w-10 sm:h-12 sm:w-12 bg-primary/10 rounded-lg flex items-center justify-center flex-shrink-0">
                    <Building className="h-5 w-5 sm:h-6 sm:w-6 text-primary" />
                  </div>
                  <Badge 
                    variant={tenant.enabled ? "default" : "secondary"}
                    data-testid={`tenant-status-${tenant.id}`}
                  >
                    {tenant.enabled ? "Active" : "Inactive"}
                  </Badge>
                </div>
                
                <h3 className="text-base sm:text-lg font-semibold text-foreground mb-2 truncate" data-testid={`tenant-name-${tenant.id}`}>
                  {tenant.name}
                </h3>
                <p className="text-sm text-muted-foreground mb-3 sm:mb-4">
                  Enterprise tenant with managed access
                </p>
                
                <div className="space-y-2 mb-4 text-sm">
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground flex items-center">
                      <Users className="mr-1 h-3 w-3" />
                      RulePacks
                    </span>
                    <span className="font-medium">-</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">Last Activity</span>
                    <span className="font-medium">
                      {formatTimeAgo(tenant.updatedAt ? new Date(tenant.updatedAt).toISOString() : new Date().toISOString())}
                    </span>
                  </div>
                </div>
                
                <div className="flex flex-col sm:flex-row gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    className="flex-1"
                    onClick={() => openEditModal(tenant)}
                    data-testid={`button-edit-tenant-${tenant.id}`}
                  >
                    <Edit className="mr-1 h-4 w-4" />
                    <span className="sm:inline">Edit</span>
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleDelete(tenant.id)}
                    disabled={deleteMutation.isPending}
                    data-testid={`button-delete-tenant-${tenant.id}`}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Create/Edit Modal */}
      <Dialog open={isModalOpen} onOpenChange={closeModal}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {isEditing ? "Edit Tenant" : "Create New Tenant"}
            </DialogTitle>
          </DialogHeader>
          
          <Form {...form}>
            <form 
              onSubmit={form.handleSubmit(isEditing ? handleUpdate : handleCreate)} 
              className="space-y-4"
            >
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Tenant Name</FormLabel>
                    <FormControl>
                      <Input 
                        {...field} 
                        placeholder="e.g., Acme Corporation"
                        data-testid="input-tenant-name"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              
              <FormField
                control={form.control}
                name="enabled"
                render={({ field }) => (
                  <FormItem className="flex flex-row items-center justify-between rounded-lg border p-4">
                    <div className="space-y-0.5">
                      <FormLabel className="text-base">Active Status</FormLabel>
                      <div className="text-sm text-muted-foreground">
                        Enable this tenant for active use
                      </div>
                    </div>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                        data-testid="switch-tenant-enabled"
                      />
                    </FormControl>
                  </FormItem>
                )}
              />
              
              <div className="flex justify-end space-x-2 pt-4">
                <Button type="button" variant="outline" onClick={closeModal} data-testid="button-cancel">
                  Cancel
                </Button>
                <Button 
                  type="submit" 
                  disabled={createMutation.isPending || updateMutation.isPending}
                  data-testid="button-save-tenant"
                >
                  {(createMutation.isPending || updateMutation.isPending) ? "Saving..." : (isEditing ? "Update Tenant" : "Create Tenant")}
                </Button>
              </div>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </Layout>
  );
}
