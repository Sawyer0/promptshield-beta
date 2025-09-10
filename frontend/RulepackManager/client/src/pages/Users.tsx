import { useState, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Layout } from '@/components/Layout';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { SearchInput } from '@/components/common/SearchInput';
import { Skeleton } from '@/components/ui/skeleton';
import { useToast } from '@/hooks/use-toast';
import { userApi, handleApiError } from '@/lib/api';
import { 
  Users as UsersIcon, 
  UserPlus,
  Search,
  Filter,
  MoreHorizontal,
  Edit,
  Trash2,
  Shield,
  User,
  Crown,
  Building
} from 'lucide-react';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { SelectControl } from '@/components/common/SelectControl';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { UserModal } from '@/components/UserModal';

export interface SystemUser {
  id: string;
  email: string;
  firstName?: string;
  lastName?: string;
  name?: string;
  role: 'admin' | 'user';
  status: 'active' | 'inactive' | 'suspended';
  created_at: string;
  updated_at: string;
  tenants?: UserTenant[];
}

export interface UserTenant {
  tenant_id: string;
  tenant_name: string;
  role: string;
}

export default function Users() {
  const [searchTerm, setSearchTerm] = useState("");
  const [roleFilter, setRoleFilter] = useState<string>("all");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [modalUser, setModalUser] = useState<SystemUser | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const { toast } = useToast();
  const queryClient = useQueryClient();

  // Fetch users
  const { data: usersData, isLoading } = useQuery({
    queryKey: ['/api/v1/admin/users'],
    queryFn: () => userApi.getAll(),
    refetchOnWindowFocus: false,
    staleTime: 2 * 60 * 1000, // Data is fresh for 2 minutes
    gcTime: 5 * 60 * 1000, // Keep in cache for 5 minutes
  });

  const users: SystemUser[] = usersData?.data || [];

  // Filter users
  const filteredUsers = useMemo<SystemUser[]>(() => {
    return users.filter((user: SystemUser) => {
      const matchesSearch = !searchTerm || 
        user.email.toLowerCase().includes(searchTerm.toLowerCase()) ||
        user.name?.toLowerCase().includes(searchTerm.toLowerCase());

      const matchesRole = roleFilter === 'all' || user.role === roleFilter;
      const matchesStatus = statusFilter === 'all' || user.status === statusFilter;

      return matchesSearch && matchesRole && matchesStatus;
    });
  }, [users, searchTerm, roleFilter, statusFilter]);

  // Delete user mutation
  const deleteUserMutation = useMutation({
    mutationFn: (userId: string) => userApi.delete(userId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['/api/v1/admin/users'] });
      toast({
        title: "User Deleted",
        description: "User has been successfully deleted.",
      });
    },
    onError: (error) => {
      handleApiError(error as Error, toast);
    },
  });

  // Update user status mutation
  const updateUserStatusMutation = useMutation({
    mutationFn: ({ userId, status }: { userId: string; status: string }) => 
      userApi.updateStatus(userId, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['/api/v1/admin/users'] });
      toast({
        title: "User Updated",
        description: "User status has been updated.",
      });
    },
    onError: (error) => {
      handleApiError(error as Error, toast);
    },
  });

  const handleCreateUser = () => {
    setModalUser(null);
    setIsModalOpen(true);
  };

  const handleEditUser = (user: SystemUser) => {
    setModalUser(user);
    setIsModalOpen(true);
  };

  const handleDeleteUser = (userId: string) => {
    if (confirm('Are you sure you want to delete this user? This action cannot be undone.')) {
      deleteUserMutation.mutate(userId);
    }
  };

  const handleStatusChange = (userId: string, status: string) => {
    updateUserStatusMutation.mutate({ userId, status });
  };

  const getRoleIcon = (role: string) => {
    switch (role) {
      case 'admin':
        return <Crown className="h-4 w-4 text-purple-600" />;
      default:
        return <User className="h-4 w-4 text-gray-600" />;
    }
  };

  const getRoleBadgeVariant = (role: string) => {
    switch (role) {
      case 'admin':
        return 'default';
      default:
        return 'outline';
    }
  };

  const getStatusBadgeVariant = (status: string) => {
    switch (status) {
      case 'active':
        return 'default';
      case 'inactive':
        return 'secondary';
      case 'suspended':
        return 'destructive';
      default:
        return 'outline';
    }
  };

  return (
    <Layout title="User Management" description="Manage system users and their permissions">
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <UsersIcon className="h-6 w-6 text-primary" />
            <h1 className="text-2xl font-bold">User Management</h1>
          </div>
          
          <Button onClick={handleCreateUser} data-testid="button-create-user">
            <UserPlus className="h-4 w-4 mr-2" />
            Add User
          </Button>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <Card className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs font-medium text-muted-foreground mb-1">Total Users</p>
                <p className="text-xl font-bold">{users.length}</p>
              </div>
              <UsersIcon className="h-6 w-6 text-blue-600" />
            </div>
          </Card>
          
          <Card className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs font-medium text-muted-foreground mb-1">Active Users</p>
                <p className="text-xl font-bold text-green-600">
                  {users.filter((u: SystemUser) => u.status === 'active').length}
                </p>
              </div>
              <User className="h-6 w-6 text-green-600" />
            </div>
          </Card>
          
          <Card className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs font-medium text-muted-foreground mb-1">Platform Owners</p>
                <p className="text-xl font-bold text-purple-600">
                  {users.filter((u: SystemUser) => u.role === 'admin').length}
                </p>
              </div>
              <Shield className="h-6 w-6 text-purple-600" />
            </div>
          </Card>
          
          <Card className="p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs font-medium text-muted-foreground mb-1">Suspended</p>
                <p className="text-xl font-bold text-red-600">
                  {users.filter((u: SystemUser) => u.status === 'suspended').length}
                </p>
              </div>
              <Trash2 className="h-6 w-6 text-red-600" />
            </div>
          </Card>
        </div>

        {/* Filters */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <Filter className="h-5 w-5 mr-2" />
              Filters & Search
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col sm:flex-row gap-4">
              <div className="flex-1">
                <SearchInput value={searchTerm} onChange={setSearchTerm} placeholder="Search by email or name..." data-testid="input-search" />
              </div>
              <SelectControl
                value={roleFilter}
                onChange={setRoleFilter}
                options={[
                  { value: 'all', label: 'All Roles' },
                  { value: 'admin', label: 'Platform Owner' },
                  { value: 'user', label: 'User' },
                ]}
              />
              <SelectControl
                value={statusFilter}
                onChange={setStatusFilter}
                options={[
                  { value: 'all', label: 'All Status' },
                  { value: 'active', label: 'Active' },
                  { value: 'inactive', label: 'Inactive' },
                  { value: 'suspended', label: 'Suspended' },
                ]}
              />
            </div>
          </CardContent>
        </Card>

        {/* Users Table */}
        <Card>
          <CardHeader>
            <CardTitle>Users ({filteredUsers.length})</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="space-y-4">
                {[...Array(5)].map((_, i) => (
                  <div key={i} className="flex items-center space-x-4 p-4 border rounded-lg">
                    <Skeleton className="h-10 w-10 rounded-full" />
                    <div className="flex-1 space-y-2">
                      <Skeleton className="h-4 w-48" />
                      <Skeleton className="h-3 w-32" />
                    </div>
                    <Skeleton className="h-6 w-16" />
                    <Skeleton className="h-6 w-20" />
                  </div>
                ))}
              </div>
            ) : (
              <div className="rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>User</TableHead>
                      <TableHead>Role</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Tenants</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredUsers.map((user: SystemUser) => (
                      <TableRow key={user.id} data-testid={`row-user-${user.id}`}>
                        <TableCell>
                          <div className="flex items-center space-x-3">
                            <div className="flex-shrink-0">
                              <div className="h-10 w-10 bg-gradient-to-r from-blue-500 to-purple-600 rounded-full flex items-center justify-center">
                                <span className="text-white font-medium text-sm">
                                  {(user.firstName?.[0] || user.email[0]).toUpperCase()}
                                </span>
                              </div>
                            </div>
                            <div>
                              <div className="text-sm font-medium text-gray-900">
                                {user.name || `${user.firstName} ${user.lastName}`.trim() || 'Unnamed User'}
                              </div>
                              <div className="text-sm text-gray-500">{user.email}</div>
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center space-x-2">
                            {getRoleIcon(user.role)}
                            <Badge variant={getRoleBadgeVariant(user.role)} className="text-xs">
                              {user.role === 'admin' ? 'Platform Owner' : 'User'}
                            </Badge>
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge variant={getStatusBadgeVariant(user.status)} className="text-xs">
                            {user.status}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center space-x-1">
                            <Building className="h-4 w-4 text-muted-foreground" />
                            <span className="text-sm text-muted-foreground">
                              {user.tenants?.length || 0} tenants
                            </span>
                          </div>
                        </TableCell>
                        <TableCell>
                          <span className="text-sm text-muted-foreground">
                            {new Date(user.created_at).toLocaleDateString()}
                          </span>
                        </TableCell>
                        <TableCell className="text-right">
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" className="h-8 w-8 p-0">
                                <MoreHorizontal className="h-4 w-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem onClick={() => handleEditUser(user)}>
                                <Edit className="mr-2 h-4 w-4" />
                                Edit User
                              </DropdownMenuItem>
                              <DropdownMenuItem 
                                onClick={() => handleStatusChange(user.id, user.status === 'active' ? 'suspended' : 'active')}
                              >
                                <Shield className="mr-2 h-4 w-4" />
                                {user.status === 'active' ? 'Suspend' : 'Activate'}
                              </DropdownMenuItem>
                              <DropdownMenuItem 
                                className="text-red-600"
                                onClick={() => handleDeleteUser(user.id)}
                              >
                                <Trash2 className="mr-2 h-4 w-4" />
                                Delete User
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
            
            {filteredUsers.length === 0 && !isLoading && (
              <div className="text-center py-12">
                <UsersIcon className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                <h3 className="text-lg font-medium text-gray-900 mb-2">No users found</h3>
                <p className="text-sm text-muted-foreground">
                  {searchTerm || roleFilter !== 'all' || statusFilter !== 'all'
                    ? "Try adjusting your search or filter criteria"
                    : "Get started by creating your first user"}
                </p>
              </div>
            )}
          </CardContent>
        </Card>

        {/* User Modal */}
        <UserModal
          user={modalUser}
          isOpen={isModalOpen}
          onClose={() => setIsModalOpen(false)}
          onSave={() => {
            queryClient.invalidateQueries({ queryKey: ['/api/v1/admin/users'] });
            setIsModalOpen(false);
          }}
        />
      </div>
    </Layout>
  );
}
