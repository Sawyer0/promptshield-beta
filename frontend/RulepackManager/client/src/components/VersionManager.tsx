import { useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogTrigger } from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { useToast } from '@/hooks/use-toast';
import { rulePackApi } from '@/lib/api';
import { queryClient } from '@/lib/queryClient';
import { 
  Plus, 
  GitBranch, 
  Play, 
  Square, 
  Eye,
  FileText,
  Clock,
  CheckCircle,
  XCircle,
  AlertCircle
} from 'lucide-react';

interface VersionManagerProps {
  rulepackId: string;
  rulepackName: string;
  currentVersion?: string;
  isActive?: boolean;
}

const defaultDSL = `rules:
  - id: 'basic-security-check'
    name: 'Basic Security Pattern'
    description: 'Detects common security threats'
    patterns:
      - regex: '(ignore|bypass|skip).*instructions'
        case_sensitive: false
      - keywords:
        - 'system prompt'
        - 'admin override'
        - 'developer mode'
    severity: 'high'
    action: 'block'
    
  - id: 'injection-detection'
    name: 'Injection Detection'
    description: 'Prevents prompt injection attempts'
    patterns:
      - regex: 'forget.*previous.*instructions'
        case_sensitive: false
      - regex: 'new.*instructions.*start.*now'
        case_sensitive: false
    severity: 'critical'
    action: 'block'`;

export function VersionManager({ rulepackId, rulepackName, currentVersion, isActive }: VersionManagerProps) {
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [isViewModalOpen, setIsViewModalOpen] = useState(false);
  const [selectedVersion, setSelectedVersion] = useState<any>(null);
  const [dslContent, setDslContent] = useState(defaultDSL);
  const [status, setStatus] = useState<'draft' | 'approved'>('approved');
  const { toast } = useToast();

  const { data: versionsResponse, isLoading: versionsLoading } = useQuery({
    queryKey: [`/api/v1/rulepacks/${rulepackId}/versions`],
    queryFn: () => rulePackApi.listVersions(rulepackId),
  });

  const createVersionMutation = useMutation({
    mutationFn: ({ dsl, status }: { dsl: string; status: 'draft' | 'approved' }) =>
      rulePackApi.createVersion(rulepackId, { dsl, status }),
    onSuccess: () => {
      toast({
        title: "Success",
        description: "New version created successfully!",
      });
      setIsCreateModalOpen(false);
      queryClient.invalidateQueries({ queryKey: [`/api/v1/rulepacks/${rulepackId}/versions`] });
      queryClient.invalidateQueries({ queryKey: ["/api/v1/rulepacks"] });
    },
    onError: (error: Error) => {
      toast({
        title: "Error",
        description: error.message || "Failed to create version",
        variant: "destructive",
      });
    },
  });

  const activateVersionMutation = useMutation({
    mutationFn: (versionId: string) => rulePackApi.activateVersion(rulepackId, versionId),
    onSuccess: () => {
      toast({
        title: "Success",
        description: "Version activated successfully!",
      });
      queryClient.invalidateQueries({ queryKey: [`/api/v1/rulepacks/${rulepackId}/versions`] });
      queryClient.invalidateQueries({ queryKey: ["/api/v1/rulepacks"] });
    },
    onError: (error: Error) => {
      toast({
        title: "Error",
        description: error.message || "Failed to activate version",
        variant: "destructive",
      });
    },
  });

  const activateLatestMutation = useMutation({
    mutationFn: () => rulePackApi.activateLatest(rulepackId),
    onSuccess: () => {
      toast({
        title: "Success",
        description: "Latest version activated successfully!",
      });
      queryClient.invalidateQueries({ queryKey: [`/api/v1/rulepacks/${rulepackId}/versions`] });
      queryClient.invalidateQueries({ queryKey: ["/api/v1/rulepacks"] });
    },
    onError: (error: Error) => {
      toast({
        title: "Error",
        description: error.message || "Failed to activate latest version",
        variant: "destructive",
      });
    },
  });

  const deactivateMutation = useMutation({
    mutationFn: () => rulePackApi.deactivate(rulepackId),
    onSuccess: () => {
      toast({
        title: "Success",
        description: "RulePack deactivated successfully!",
      });
      queryClient.invalidateQueries({ queryKey: [`/api/v1/rulepacks/${rulepackId}/versions`] });
      queryClient.invalidateQueries({ queryKey: ["/api/v1/rulepacks"] });
    },
    onError: (error: Error) => {
      toast({
        title: "Error",
        description: error.message || "Failed to deactivate rulepack",
        variant: "destructive",
      });
    },
  });

  const handleCreateVersion = () => {
    createVersionMutation.mutate({ dsl: dslContent, status });
  };

  const handleViewVersion = async (version: any) => {
    try {
      const response = await rulePackApi.getVersion(rulepackId, version.version_number || version.id);
      setSelectedVersion(response.data);
      setIsViewModalOpen(true);
    } catch (error) {
      toast({
        title: "Error",
        description: "Failed to load version details",
        variant: "destructive",
      });
    }
  };

  const versions = versionsResponse?.data || [];

  const getStatusBadge = (versionStatus: string, isActivated: boolean) => {
    if (isActivated) {
      return <Badge variant="default" className="bg-green-100 text-green-800 border-green-200"><Play className="w-3 h-3 mr-1" />Active</Badge>;
    }
    
    switch (versionStatus) {
      case 'approved':
        return <Badge variant="secondary"><CheckCircle className="w-3 h-3 mr-1" />Approved</Badge>;
      case 'draft':
        return <Badge variant="outline"><AlertCircle className="w-3 h-3 mr-1" />Draft</Badge>;
      default:
        return <Badge variant="outline">{versionStatus}</Badge>;
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <GitBranch className="h-5 w-5" />
          <h3 className="text-lg font-semibold">Version Management</h3>
          {currentVersion && (
            <Badge variant="outline">Current: v{currentVersion}</Badge>
          )}
          {isActive && (
            <Badge variant="default" className="bg-green-100 text-green-800 border-green-200">
              <Play className="w-3 h-3 mr-1" />Active
            </Badge>
          )}
        </div>
        
        <div className="flex space-x-2">
          <Dialog open={isCreateModalOpen} onOpenChange={setIsCreateModalOpen}>
            <DialogTrigger asChild>
              <Button size="sm" data-testid="button-create-version">
                <Plus className="w-4 h-4 mr-1" />
                New Version
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-4xl max-h-[90vh]">
              <DialogHeader>
                <DialogTitle>Create New Version - {rulepackName}</DialogTitle>
                <DialogDescription>
                  Create a new version of the rulepack with updated security rules and configurations.
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div>
                  <label className="text-sm font-medium">Status</label>
                  <Select value={status} onValueChange={(value: 'draft' | 'approved') => setStatus(value)}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="draft">Draft</SelectItem>
                      <SelectItem value="approved">Approved</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                
                <div>
                  <label className="text-sm font-medium">DSL Content</label>
                  <Textarea
                    value={dslContent}
                    onChange={(e) => setDslContent(e.target.value)}
                    className="min-h-[400px] font-mono text-sm"
                    placeholder="Enter your DSL rules here..."
                  />
                </div>
                
                <div className="flex justify-end space-x-2">
                  <Button
                    variant="outline"
                    onClick={() => setIsCreateModalOpen(false)}
                  >
                    Cancel
                  </Button>
                  <Button
                    onClick={handleCreateVersion}
                    disabled={createVersionMutation.isPending}
                  >
                    {createVersionMutation.isPending ? "Creating..." : "Create Version"}
                  </Button>
                </div>
              </div>
            </DialogContent>
          </Dialog>

          <Button
            size="sm"
            variant="outline"
            onClick={() => activateLatestMutation.mutate()}
            disabled={activateLatestMutation.isPending}
            data-testid="button-activate-latest"
          >
            <Play className="w-4 h-4 mr-1" />
            Activate Latest
          </Button>
          
          {isActive && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => deactivateMutation.mutate()}
              disabled={deactivateMutation.isPending}
              data-testid="button-deactivate"
            >
              <Square className="w-4 h-4 mr-1" />
              Deactivate
            </Button>
          )}
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center">
            <FileText className="w-4 h-4 mr-2" />
            Versions ({versions.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          {versionsLoading ? (
            <div className="text-center py-8">Loading versions...</div>
          ) : versions.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              No versions created yet. Create your first version to get started.
            </div>
          ) : (
            <ScrollArea className="h-64">
              <div className="space-y-3">
                {versions.map((version: any, index: number) => (
                  <div key={version.id || index} className="flex items-center justify-between p-3 border rounded-lg">
                    <div className="flex items-center space-x-3">
                      <Badge variant="outline">v{version.version_number || index + 1}</Badge>
                      {getStatusBadge(version.status, version.is_active)}
                      <div>
                        <div className="text-sm font-medium">
                          Version {version.version_number || index + 1}
                        </div>
                        <div className="text-xs text-muted-foreground flex items-center">
                          <Clock className="w-3 h-3 mr-1" />
                          {version.created_at ? new Date(version.created_at).toLocaleDateString() : 'Unknown date'}
                        </div>
                      </div>
                    </div>
                    
                    <div className="flex items-center space-x-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleViewVersion(version)}
                        data-testid={`button-view-version-${version.id || index}`}
                      >
                        <Eye className="w-3 h-3 mr-1" />
                        View
                      </Button>
                      
                      {!version.is_active && version.status === 'approved' && (
                        <Button
                          size="sm"
                          onClick={() => activateVersionMutation.mutate(version.id)}
                          disabled={activateVersionMutation.isPending}
                          data-testid={`button-activate-version-${version.id || index}`}
                        >
                          <Play className="w-3 h-3 mr-1" />
                          Activate
                        </Button>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </ScrollArea>
          )}
        </CardContent>
      </Card>

      {/* View Version Modal */}
      <Dialog open={isViewModalOpen} onOpenChange={setIsViewModalOpen}>
        <DialogContent className="max-w-4xl max-h-[90vh]">
          <DialogHeader>
            <DialogTitle>
              Version {selectedVersion?.version_number || 'Details'} - {rulepackName}
            </DialogTitle>
            <DialogDescription>
              View detailed information about this version of the rulepack.
            </DialogDescription>
          </DialogHeader>
          {selectedVersion && (
            <div className="space-y-4">
              <div className="flex items-center space-x-4">
                {getStatusBadge(selectedVersion.status, selectedVersion.is_active)}
                <div className="text-sm text-muted-foreground">
                  Created: {selectedVersion.created_at ? new Date(selectedVersion.created_at).toLocaleDateString() : 'Unknown'}
                </div>
              </div>
              
              <Separator />
              
              <div>
                <label className="text-sm font-medium">DSL Content</label>
                <ScrollArea className="h-96 border rounded-md p-4 bg-muted/50">
                  <pre className="text-xs font-mono whitespace-pre-wrap">
                    {selectedVersion.dsl || 'No DSL content available'}
                  </pre>
                </ScrollArea>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}