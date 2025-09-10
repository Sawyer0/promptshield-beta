import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Modal } from '@/components/common/Modal';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { ScrollArea } from '@/components/ui/scroll-area';
import { VersionManager } from './VersionManager';
import { rulePackApi } from '@/lib/api';
import { 
  Shield, 
  Calendar, 
  User, 
  Activity,
  FileText,
  Settings,
  GitBranch,
  Play,
  Square
} from 'lucide-react';
import type { RulePack } from '@shared/apiTypes';

interface RulePackDetailsModalProps {
  isOpen: boolean;
  onClose: () => void;
  rulepack: RulePack | null;
}

export function RulePackDetailsModal({ isOpen, onClose, rulepack }: RulePackDetailsModalProps) {
  const [activeTab, setActiveTab] = useState('overview');

  const { data: rulepackDetails, isLoading } = useQuery({
    queryKey: [`/api/v1/rulepacks/${rulepack?.id}`, rulepack?.id],
    queryFn: () => rulePackApi.get(rulepack!.id),
    enabled: !!rulepack?.id && isOpen,
  });

  if (!rulepack) return null;

  const details = rulepackDetails?.data || rulepack;

  const formatDate = (dateString: string | null) => {
    if (!dateString) return 'Unknown';
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const getRuleCount = (rp: RulePack) => {
    if (Array.isArray(rp.rules)) {
      return rp.rules.length;
    }
    if (typeof rp.rules === 'object' && rp.rules !== null) {
      return Object.keys(rp.rules).length;
    }
    return 0;
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      size="2xl"
      title={
        <div className="flex items-center space-x-2">
          <Shield className="h-5 w-5" />
          <span>{details.name}</span>
          <Badge variant={details.isActive ? 'default' : 'secondary'} className={details.isActive ? 'bg-green-100 text-green-800 border-green-200' : ''}>
            {details.isActive ? <Play className="w-3 h-3 mr-1" /> : <Square className="w-3 h-3 mr-1" />}
            {details.isActive ? 'Active' : 'Inactive'}
          </Badge>
        </div>
      }
      description={"View and manage rulepack details, versions, and configuration settings."}
      contentClassName="max-h-[90vh]"
    >

        <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="overview" className="flex items-center">
              <Activity className="w-4 h-4 mr-2" />
              Overview
            </TabsTrigger>
            <TabsTrigger value="versions" className="flex items-center">
              <GitBranch className="w-4 h-4 mr-2" />
              Versions
            </TabsTrigger>
            <TabsTrigger value="settings" className="flex items-center">
              <Settings className="w-4 h-4 mr-2" />
              Settings
            </TabsTrigger>
          </TabsList>

          <ScrollArea className="h-[600px] mt-4">
            <TabsContent value="overview" className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                {/* Basic Information */}
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center text-base">
                      <FileText className="w-4 h-4 mr-2" />
                      Basic Information
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Name</label>
                      <p className="text-sm">{details.name}</p>
                    </div>
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Description</label>
                      <p className="text-sm">{details.description || 'No description provided'}</p>
                    </div>
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Version</label>
                      <p className="text-sm">v{details.currentVersionId ?? 'n/a'}</p>
                    </div>
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Rule Count</label>
                      <p className="text-sm">{getRuleCount(details)} rules</p>
                    </div>
                  </CardContent>
                </Card>

                {/* Metadata */}
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center text-base">
                      <Calendar className="w-4 h-4 mr-2" />
                      Metadata
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Created</label>
                      <p className="text-sm">{formatDate(details.createdAt?.toString() || null)}</p>
                    </div>
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Last Updated</label>
                      <p className="text-sm">{formatDate(details.updatedAt?.toString() || null)}</p>
                    </div>
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Status</label>
                      <Badge variant={details.isActive ? 'default' : 'secondary'} className={details.isActive ? 'bg-green-100 text-green-800 border-green-200' : ''}>
                        {details.isActive ? 'Active' : 'Inactive'}
                      </Badge>
                    </div>
                    <div>
                      <label className="text-sm font-medium text-muted-foreground">Creator</label>
                      <div className="flex items-center">
                        <User className="w-3 h-3 mr-1" />
                        <p className="text-sm">{(details as any).userId || 'Unknown'}</p>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </div>

              {/* Rules Preview */}
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center text-base">
                    <Shield className="w-4 h-4 mr-2" />
                    Rules Preview
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  {getRuleCount(details) > 0 ? (
                    <ScrollArea className="h-48 border rounded-md p-4 bg-muted/50">
                      <pre className="text-xs font-mono whitespace-pre-wrap">
                        {typeof details.rules === 'string' 
                          ? details.rules 
                          : JSON.stringify(details.rules, null, 2)
                        }
                      </pre>
                    </ScrollArea>
                  ) : (
                    <div className="text-center py-8 text-muted-foreground">
                      No rules defined yet. Create a version to add rules.
                    </div>
                  )}
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="versions" className="space-y-6">
              <VersionManager
                rulepackId={details.id}
                rulepackName={details.name}
                currentVersion={details.currentVersionId ?? 'n/a'}
                isActive={details.isActive}
              />
            </TabsContent>

            <TabsContent value="settings" className="space-y-6">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center text-base">
                    <Settings className="w-4 h-4 mr-2" />
                    RulePack Settings
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="text-sm font-medium">Enforcement Mode</label>
                      <Badge variant="outline" className="block w-fit mt-1">
                        {details.isActive ? 'Active Enforcement' : 'Monitoring Only'}
                      </Badge>
                    </div>
                    <div>
                      <label className="text-sm font-medium">Auto-Update</label>
                      <Badge variant="outline" className="block w-fit mt-1">Disabled</Badge>
                    </div>
                  </div>
                  
                  <Separator />
                  
                  <div>
                    <label className="text-sm font-medium">Metadata</label>
                    <div className="mt-2 space-y-2 text-xs">
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">RulePack ID:</span>
                        <span className="font-mono">{details.id}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">Internal Version:</span>
                        <span className="font-mono">{details.currentVersionId ?? 'n/a'}</span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-muted-foreground">Created by:</span>
                        <span>{(details as any).userId ?? 'Unknown'}</span>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center text-base text-destructive">
                    <Shield className="w-4 h-4 mr-2" />
                    Danger Zone
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="p-4 border border-destructive/20 rounded-lg bg-destructive/5">
                    <h4 className="font-medium text-sm mb-2">Delete RulePack</h4>
                    <p className="text-sm text-muted-foreground mb-4">
                      Once you delete a rulepack, there is no going back. This will permanently delete the rulepack and all its versions.
                    </p>
                    <Button variant="destructive" size="sm">
                      Delete RulePack
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </ScrollArea>
        </Tabs>
    </Modal>
  );
}
