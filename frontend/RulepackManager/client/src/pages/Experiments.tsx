import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { format } from 'date-fns';
import { 
  FlaskConical, 
  Play, 
  Pause, 
  Square, 
  BarChart3, 
  Users, 
  TrendingUp,
  Plus,
  Filter,
  RefreshCw,
  Target,
  Zap
} from 'lucide-react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Progress } from '@/components/ui/progress';
import { useAuth } from '@/hooks/useAuth';

// Mock API functions - in production these would call the actual API
const experimentApi = {
  getExperiments: async (params?: any) => {
    // Mock data
    return {
      experiments: [
        {
          id: '1',
          name: 'Pricing Tier Test',
          description: 'Test new pricing structure for Professional tier',
          type: 'pricing_tier',
          status: 'active',
          traffic_allocation: 0.5,
          primary_metric: 'conversion_rate',
          start_date: '2024-01-01T00:00:00Z',
          end_date: '2024-02-01T00:00:00Z',
          results: {
            total_participants: 1250,
            control_participants: 625,
            test_participants: 625,
            conversion_rate: 0.12,
            control_conversion_rate: 0.10,
            test_conversion_rate: 0.14,
            lift: 40.0,
            statistical_significance: true,
            confidence_level: 95.0
          },
          variants: [
            {
              id: 'v1',
              name: 'Control',
              type: 'control',
              weight: 0.5,
              configuration: {
                professional_monthly: 29900,
                professional_yearly: 299000
              }
            },
            {
              id: 'v2',
              name: 'Test',
              type: 'test',
              weight: 0.5,
              configuration: {
                professional_monthly: 24900,
                professional_yearly: 249000
              }
            }
          ]
        },
        {
          id: '2',
          name: 'Usage Addon Test',
          description: 'Test premium analytics addon',
          type: 'usage_addon',
          status: 'completed',
          traffic_allocation: 1.0,
          primary_metric: 'revenue_per_user',
          start_date: '2023-12-01T00:00:00Z',
          end_date: '2023-12-31T00:00:00Z',
          results: {
            total_participants: 800,
            control_participants: 400,
            test_participants: 400,
            conversion_rate: 0.08,
            control_conversion_rate: 0.06,
            test_conversion_rate: 0.10,
            lift: 66.7,
            statistical_significance: true,
            confidence_level: 99.0
          },
          variants: [
            {
              id: 'v3',
              name: 'Control',
              type: 'control',
              weight: 0.5,
              configuration: {
                addons: ['basic_analytics']
              }
            },
            {
              id: 'v4',
              name: 'Test',
              type: 'test',
              weight: 0.5,
              configuration: {
                addons: ['basic_analytics', 'premium_analytics', 'advanced_reporting']
              }
            }
          ]
        }
      ]
    };
  },
  
  createExperiment: async (data: any) => {
    console.log('Creating experiment:', data);
    return { id: 'new-experiment-id', ...data };
  },
  
  updateExperimentStatus: async (experimentId: string, status: string) => {
    console.log('Updating experiment status:', experimentId, status);
    return { success: true };
  },
  
  getExperimentAnalytics: async (experimentId: string) => {
    return {
      experiment_id: experimentId,
      total_users: 1250,
      active_users: 1100,
      conversion_funnel: {
        'page_view': 1250,
        'pricing_view': 800,
        'trial_start': 200,
        'conversion': 150
      },
      revenue_impact: 0.15,
      churn_impact: -0.05,
      variant_breakdown: {
        'control': {
          variant_id: 'v1',
          users: 625,
          conversions: 63,
          conversion_rate: 0.10,
          revenue: 15750.0,
          average_order_value: 250.0,
          churn_rate: 0.05,
          lifetime_value: 5000.0
        },
        'test': {
          variant_id: 'v2',
          users: 625,
          conversions: 88,
          conversion_rate: 0.14,
          revenue: 22000.0,
          average_order_value: 250.0,
          churn_rate: 0.03,
          lifetime_value: 7000.0
        }
      }
    };
  }
};

const statusColors = {
  draft: 'bg-gray-100 text-gray-800',
  active: 'bg-green-100 text-green-800',
  paused: 'bg-yellow-100 text-yellow-800',
  completed: 'bg-blue-100 text-blue-800',
  cancelled: 'bg-red-100 text-red-800',
};

const typeIcons = {
  pricing_tier: Target,
  usage_addon: Zap,
  feature_flag: FlaskConical,
  ui_optimization: BarChart3,
};

export default function Experiments() {
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [filters, setFilters] = useState({
    status: '',
    type: '',
  });
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [createForm, setCreateForm] = useState({
    name: '',
    description: '',
    type: 'pricing_tier',
    primary_metric: 'conversion_rate',
    traffic_allocation: 0.5,
  });

  // Fetch experiments
  const { data: experimentsData, isLoading: experimentsLoading, refetch: refetchExperiments } = useQuery({
    queryKey: ['experiments', filters],
    queryFn: () => experimentApi.getExperiments(filters),
  });

  // Create experiment mutation
  const createExperimentMutation = useMutation({
    mutationFn: experimentApi.createExperiment,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['experiments'] });
      setShowCreateDialog(false);
      setCreateForm({
        name: '',
        description: '',
        type: 'pricing_tier',
        primary_metric: 'conversion_rate',
        traffic_allocation: 0.5,
      });
    },
  });

  // Update status mutation
  const updateStatusMutation = useMutation({
    mutationFn: ({ experimentId, status }: { experimentId: string; status: string }) =>
      experimentApi.updateExperimentStatus(experimentId, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['experiments'] });
    },
  });

  const handleCreateExperiment = () => {
    createExperimentMutation.mutate(createForm);
  };

  const handleUpdateStatus = (experimentId: string, status: string) => {
    updateStatusMutation.mutate({ experimentId, status });
  };

  const experiments = experimentsData?.experiments || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">A/B Experiments</h1>
          <p className="text-muted-foreground">
            Manage A/B tests and pricing optimization experiments
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => refetchExperiments()}
            disabled={experimentsLoading}
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Refresh
          </Button>
          <Dialog open={showCreateDialog} onOpenChange={setShowCreateDialog}>
            <DialogTrigger asChild>
              <Button>
                <Plus className="h-4 w-4 mr-2" />
                New Experiment
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Create New Experiment</DialogTitle>
                <DialogDescription>
                  Set up a new A/B test to optimize pricing or features
                </DialogDescription>
              </DialogHeader>
              <div className="space-y-4">
                <div>
                  <Label htmlFor="name">Experiment Name</Label>
                  <Input
                    id="name"
                    value={createForm.name}
                    onChange={(e) => setCreateForm(prev => ({ ...prev, name: e.target.value }))}
                    placeholder="Enter experiment name"
                  />
                </div>
                <div>
                  <Label htmlFor="description">Description</Label>
                  <Input
                    id="description"
                    value={createForm.description}
                    onChange={(e) => setCreateForm(prev => ({ ...prev, description: e.target.value }))}
                    placeholder="Enter experiment description"
                  />
                </div>
                <div>
                  <Label htmlFor="type">Experiment Type</Label>
                  <Select value={createForm.type} onValueChange={(value) => setCreateForm(prev => ({ ...prev, type: value }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="pricing_tier">Pricing Tier</SelectItem>
                      <SelectItem value="usage_addon">Usage Addon</SelectItem>
                      <SelectItem value="feature_flag">Feature Flag</SelectItem>
                      <SelectItem value="ui_optimization">UI Optimization</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div>
                  <Label htmlFor="primary_metric">Primary Metric</Label>
                  <Select value={createForm.primary_metric} onValueChange={(value) => setCreateForm(prev => ({ ...prev, primary_metric: value }))}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="conversion_rate">Conversion Rate</SelectItem>
                      <SelectItem value="revenue_per_user">Revenue Per User</SelectItem>
                      <SelectItem value="churn_rate">Churn Rate</SelectItem>
                      <SelectItem value="lifetime_value">Lifetime Value</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div>
                  <Label htmlFor="traffic_allocation">Traffic Allocation</Label>
                  <Input
                    id="traffic_allocation"
                    type="number"
                    min="0"
                    max="1"
                    step="0.1"
                    value={createForm.traffic_allocation}
                    onChange={(e) => setCreateForm(prev => ({ ...prev, traffic_allocation: parseFloat(e.target.value) }))}
                  />
                </div>
                <div className="flex justify-end space-x-2">
                  <Button variant="outline" onClick={() => setShowCreateDialog(false)}>
                    Cancel
                  </Button>
                  <Button 
                    onClick={handleCreateExperiment}
                    disabled={createExperimentMutation.isPending}
                  >
                    {createExperimentMutation.isPending ? 'Creating...' : 'Create'}
                  </Button>
                </div>
              </div>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {/* Filters */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Filter className="h-4 w-4" />
            Filters
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <Label htmlFor="status">Status</Label>
              <Select value={filters.status} onValueChange={(value) => setFilters(prev => ({ ...prev, status: value }))}>
                <SelectTrigger>
                  <SelectValue placeholder="All statuses" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">All statuses</SelectItem>
                  <SelectItem value="draft">Draft</SelectItem>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="paused">Paused</SelectItem>
                  <SelectItem value="completed">Completed</SelectItem>
                  <SelectItem value="cancelled">Cancelled</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label htmlFor="type">Type</Label>
              <Select value={filters.type} onValueChange={(value) => setFilters(prev => ({ ...prev, type: value }))}>
                <SelectTrigger>
                  <SelectValue placeholder="All types" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="">All types</SelectItem>
                  <SelectItem value="pricing_tier">Pricing Tier</SelectItem>
                  <SelectItem value="usage_addon">Usage Addon</SelectItem>
                  <SelectItem value="feature_flag">Feature Flag</SelectItem>
                  <SelectItem value="ui_optimization">UI Optimization</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-end">
              <Button 
                variant="outline" 
                onClick={() => setFilters({ status: '', type: '' })}
                className="w-full"
              >
                Clear Filters
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Experiments Table */}
      <Card>
        <CardHeader>
          <CardTitle>Experiments</CardTitle>
          <CardDescription>
            {experiments.length} experiment{experiments.length !== 1 ? 's' : ''} found
          </CardDescription>
        </CardHeader>
        <CardContent>
          {experimentsLoading ? (
            <div className="text-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
              <p className="text-muted-foreground">Loading experiments...</p>
            </div>
          ) : experiments.length === 0 ? (
            <div className="text-center py-8">
              <FlaskConical className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
              <p className="text-muted-foreground">No experiments found</p>
            </div>
          ) : (
            <div className="space-y-4">
              {experiments.map((experiment: any) => {
                const TypeIcon = typeIcons[experiment.type as keyof typeof typeIcons] || FlaskConical;
                return (
                  <Card key={experiment.id}>
                    <CardHeader>
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-3">
                          <TypeIcon className="h-5 w-5 text-primary" />
                          <div>
                            <CardTitle className="text-lg">{experiment.name}</CardTitle>
                            <CardDescription>{experiment.description}</CardDescription>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <Badge className={statusColors[experiment.status as keyof typeof statusColors]}>
                            {experiment.status}
                          </Badge>
                          <div className="flex items-center gap-1">
                            {experiment.status === 'draft' && (
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => handleUpdateStatus(experiment.id, 'active')}
                              >
                                <Play className="h-4 w-4" />
                              </Button>
                            )}
                            {experiment.status === 'active' && (
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => handleUpdateStatus(experiment.id, 'paused')}
                              >
                                <Pause className="h-4 w-4" />
                              </Button>
                            )}
                            {experiment.status === 'paused' && (
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => handleUpdateStatus(experiment.id, 'active')}
                              >
                                <Play className="h-4 w-4" />
                              </Button>
                            )}
                            {(experiment.status === 'active' || experiment.status === 'paused') && (
                              <Button
                                size="sm"
                                variant="outline"
                                onClick={() => handleUpdateStatus(experiment.id, 'completed')}
                              >
                                <Square className="h-4 w-4" />
                              </Button>
                            )}
                          </div>
                        </div>
                      </div>
                    </CardHeader>
                    <CardContent>
                      <Tabs defaultValue="overview" className="w-full">
                        <TabsList>
                          <TabsTrigger value="overview">Overview</TabsTrigger>
                          <TabsTrigger value="variants">Variants</TabsTrigger>
                          <TabsTrigger value="results">Results</TabsTrigger>
                        </TabsList>
                        
                        <TabsContent value="overview" className="space-y-4">
                          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                            <div>
                              <p className="text-sm text-muted-foreground">Traffic Allocation</p>
                              <p className="text-lg font-semibold">{(experiment.traffic_allocation * 100).toFixed(0)}%</p>
                            </div>
                            <div>
                              <p className="text-sm text-muted-foreground">Primary Metric</p>
                              <p className="text-lg font-semibold">{experiment.primary_metric}</p>
                            </div>
                            <div>
                              <p className="text-sm text-muted-foreground">Start Date</p>
                              <p className="text-lg font-semibold">
                                {experiment.start_date ? format(new Date(experiment.start_date), 'MMM dd, yyyy') : 'Not started'}
                              </p>
                            </div>
                            <div>
                              <p className="text-sm text-muted-foreground">End Date</p>
                              <p className="text-lg font-semibold">
                                {experiment.end_date ? format(new Date(experiment.end_date), 'MMM dd, yyyy') : 'Ongoing'}
                              </p>
                            </div>
                          </div>
                        </TabsContent>
                        
                        <TabsContent value="variants" className="space-y-4">
                          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            {experiment.variants?.map((variant: any) => (
                              <Card key={variant.id}>
                                <CardHeader>
                                  <CardTitle className="flex items-center gap-2">
                                    {variant.type === 'control' ? 'Control' : 'Test'}
                                    <Badge variant="outline">{variant.name}</Badge>
                                  </CardTitle>
                                </CardHeader>
                                <CardContent>
                                  <div className="space-y-2">
                                    <div>
                                      <p className="text-sm text-muted-foreground">Weight</p>
                                      <p className="font-semibold">{(variant.weight * 100).toFixed(0)}%</p>
                                    </div>
                                    <div>
                                      <p className="text-sm text-muted-foreground">Configuration</p>
                                      <pre className="text-xs bg-muted p-2 rounded">
                                        {JSON.stringify(variant.configuration, null, 2)}
                                      </pre>
                                    </div>
                                  </div>
                                </CardContent>
                              </Card>
                            ))}
                          </div>
                        </TabsContent>
                        
                        <TabsContent value="results" className="space-y-4">
                          {experiment.results ? (
                            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                              <div>
                                <p className="text-sm text-muted-foreground">Total Participants</p>
                                <p className="text-2xl font-bold">{experiment.results.total_participants?.toLocaleString()}</p>
                              </div>
                              <div>
                                <p className="text-sm text-muted-foreground">Conversion Rate</p>
                                <p className="text-2xl font-bold">{(experiment.results.conversion_rate * 100).toFixed(1)}%</p>
                              </div>
                              <div>
                                <p className="text-sm text-muted-foreground">Lift</p>
                                <p className="text-2xl font-bold text-green-600">+{experiment.results.lift?.toFixed(1)}%</p>
                              </div>
                              <div>
                                <p className="text-sm text-muted-foreground">Confidence</p>
                                <p className="text-2xl font-bold">{experiment.results.confidence_level?.toFixed(0)}%</p>
                              </div>
                            </div>
                          ) : (
                            <p className="text-muted-foreground">No results available yet</p>
                          )}
                        </TabsContent>
                      </Tabs>
                    </CardContent>
                  </Card>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
