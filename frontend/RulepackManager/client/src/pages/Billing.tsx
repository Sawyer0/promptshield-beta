import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useToast } from '@/hooks/use-toast';
import { billingApi } from '@/lib/api';
import { 
  CreditCard, 
  DollarSign, 
  TrendingUp, 
  AlertTriangle, 
  CheckCircle, 
  Clock,
  Users,
  Zap,
  Shield,
  Building
} from 'lucide-react';
import { format } from 'date-fns';

interface SubscriptionPlan {
  id: string;
  name: string;
  display_name: string;
  description: string;
  price_monthly: number;
  price_yearly?: number;
  features: {
    unlimited_rulepacks: boolean;
    compliance_reporting: string[];
    support_level: string;
    sla: string;
    custom_integrations: boolean;
    advanced_analytics: boolean;
    air_gapped_deployment: boolean;
    priority_processing: boolean;
    white_labeling: boolean;
    sso: boolean;
    audit_logs: boolean;
    custom_retention: boolean;
  };
  limits: {
    api_calls_monthly: number;
    llm_calls_monthly: number;
    rulepacks: number;
    users: number;
    data_retention_days: number;
  };
}

interface Subscription {
  id: string;
  tenant_id: string;
  plan_id: string;
  status: 'active' | 'trial' | 'past_due' | 'canceled' | 'unpaid' | 'incomplete';
  billing_cycle: 'monthly' | 'yearly';
  current_period_start: string;
  current_period_end: string;
  trial_start?: string;
  trial_end?: string;
  cancel_at_period_end: boolean;
  canceled_at?: string;
}

interface UsageData {
  llm_calls: number;
  api_calls: number;
  violations: number;
}

interface QuotaStatus {
  resource_type: string;
  used: number;
  limit: number;
  remaining: number;
  is_unlimited: boolean;
  reset_date?: string;
  overage_allowed: boolean;
}

export default function Billing() {
  const { toast } = useToast();
  const queryClient = useQueryClient();
  const [selectedPlan, setSelectedPlan] = useState<string | null>(null);
  const [billingCycle, setBillingCycle] = useState<'monthly' | 'yearly'>('monthly');

  // Fetch subscription plans
  const { data: plansData, isLoading: plansLoading } = useQuery({
    queryKey: ['billing', 'plans'],
    queryFn: () => billingApi.getPlans(),
  });

  // Fetch current subscription
  const { data: subscription, isLoading: subscriptionLoading } = useQuery({
    queryKey: ['billing', 'subscription'],
    queryFn: () => billingApi.getSubscription(),
  });

  // Fetch usage data
  const { data: usageData, isLoading: usageLoading } = useQuery({
    queryKey: ['billing', 'usage'],
    queryFn: () => billingApi.getUsage(),
  });

  // Fetch quota status
  const { data: apiQuota } = useQuery({
    queryKey: ['billing', 'quota', 'api_calls'],
    queryFn: () => billingApi.checkQuota('api_calls'),
  });

  const { data: llmQuota } = useQuery({
    queryKey: ['billing', 'quota', 'llm_calls'],
    queryFn: () => billingApi.checkQuota('llm_calls'),
  });

  // Fetch billing history
  const { data: billingHistory, isLoading: historyLoading } = useQuery({
    queryKey: ['billing', 'history'],
    queryFn: () => billingApi.getBillingHistory(),
  });

  // Create subscription mutation
  const createSubscriptionMutation = useMutation({
    mutationFn: (data: { plan_id: string; billing_cycle: 'monthly' | 'yearly' }) =>
      billingApi.createSubscription(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['billing'] });
      toast({
        title: "Subscription Created",
        description: "Your subscription has been created successfully.",
      });
    },
    onError: (error: any) => {
      toast({
        title: "Error",
        description: error.message || "Failed to create subscription",
        variant: "destructive",
      });
    },
  });

  // Cancel subscription mutation
  const cancelSubscriptionMutation = useMutation({
    mutationFn: (subscriptionId: string) =>
      billingApi.cancelSubscription(subscriptionId, true),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['billing'] });
      toast({
        title: "Subscription Canceled",
        description: "Your subscription will be canceled at the end of the current period.",
      });
    },
    onError: (error: any) => {
      toast({
        title: "Error",
        description: error.message || "Failed to cancel subscription",
        variant: "destructive",
      });
    },
  });

  const plans: SubscriptionPlan[] = plansData?.plans || [];
  const currentSubscription: Subscription | null = subscription || null;
  const usage: UsageData = usageData || { llm_calls: 0, api_calls: 0, violations: 0 };

  const formatPrice = (priceInCents: number) => {
    return `$${(priceInCents / 100).toFixed(2)}`;
  };

  const getStatusBadge = (status: string) => {
    const statusConfig = {
      active: { variant: 'default' as const, icon: CheckCircle, text: 'Active' },
      trial: { variant: 'secondary' as const, icon: Clock, text: 'Trial' },
      past_due: { variant: 'destructive' as const, icon: AlertTriangle, text: 'Past Due' },
      canceled: { variant: 'outline' as const, icon: AlertTriangle, text: 'Canceled' },
      unpaid: { variant: 'destructive' as const, icon: AlertTriangle, text: 'Unpaid' },
      incomplete: { variant: 'secondary' as const, icon: Clock, text: 'Incomplete' },
    };

    const config = statusConfig[status as keyof typeof statusConfig] || statusConfig.incomplete;
    const Icon = config.icon;

    return (
      <Badge variant={config.variant} className="flex items-center gap-1">
        <Icon className="h-3 w-3" />
        {config.text}
      </Badge>
    );
  };

  const getCurrentPlan = () => {
    if (!currentSubscription || !plans.length) return null;
    return plans.find(plan => plan.id === currentSubscription.plan_id);
  };

  const currentPlan = getCurrentPlan();

  if (plansLoading || subscriptionLoading) {
    return (
      <div className="container mx-auto p-6">
        <div className="flex items-center justify-center h-64">
          <div className="text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
            <p className="text-muted-foreground">Loading billing information...</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Billing & Subscription</h1>
          <p className="text-muted-foreground">
            Manage your subscription, view usage, and billing history
          </p>
        </div>
        {currentSubscription && (
          <div className="flex items-center gap-2">
            {getStatusBadge(currentSubscription.status)}
          </div>
        )}
      </div>

      <Tabs defaultValue="overview" className="space-y-6">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="plans">Plans</TabsTrigger>
          <TabsTrigger value="usage">Usage</TabsTrigger>
          <TabsTrigger value="history">Billing History</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          {/* Current Subscription */}
          {currentSubscription && currentPlan && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <CreditCard className="h-5 w-5" />
                  Current Subscription
                </CardTitle>
                <CardDescription>
                  {currentPlan.display_name} - {currentSubscription.billing_cycle}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="text-center">
                    <div className="text-2xl font-bold">
                      {formatPrice(currentPlan.price_monthly)}
                    </div>
                    <div className="text-sm text-muted-foreground">per month</div>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold">
                      {format(new Date(currentSubscription.current_period_end), 'MMM dd, yyyy')}
                    </div>
                    <div className="text-sm text-muted-foreground">renews on</div>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold">
                      {currentSubscription.trial_end ? 'Trial' : 'Active'}
                    </div>
                    <div className="text-sm text-muted-foreground">status</div>
                  </div>
                </div>
                
                {currentSubscription.cancel_at_period_end && (
                  <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
                    <div className="flex items-center gap-2 text-yellow-800">
                      <AlertTriangle className="h-4 w-4" />
                      <span className="font-medium">Subscription will be canceled</span>
                    </div>
                    <p className="text-yellow-700 text-sm mt-1">
                      Your subscription will end on {format(new Date(currentSubscription.current_period_end), 'MMM dd, yyyy')}
                    </p>
                  </div>
                )}

                <div className="flex gap-2">
                  <Button variant="outline" size="sm">
                    Update Payment Method
                  </Button>
                  {!currentSubscription.cancel_at_period_end && (
                    <Button 
                      variant="destructive" 
                      size="sm"
                      onClick={() => cancelSubscriptionMutation.mutate(currentSubscription.id)}
                      disabled={cancelSubscriptionMutation.isPending}
                    >
                      Cancel Subscription
                    </Button>
                  )}
                </div>
              </CardContent>
            </Card>
          )}

          {/* Usage Overview */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">API Calls</CardTitle>
                <Zap className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{usage.api_calls.toLocaleString()}</div>
                {apiQuota && (
                  <div className="text-xs text-muted-foreground">
                    {apiQuota.used.toLocaleString()} / {apiQuota.is_unlimited ? '∞' : apiQuota.limit.toLocaleString()} used
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">LLM Calls</CardTitle>
                <Shield className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{usage.llm_calls.toLocaleString()}</div>
                {llmQuota && (
                  <div className="text-xs text-muted-foreground">
                    {llmQuota.used.toLocaleString()} / {llmQuota.is_unlimited ? '∞' : llmQuota.limit.toLocaleString()} used
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">Violations</CardTitle>
                <AlertTriangle className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{usage.violations.toLocaleString()}</div>
                <div className="text-xs text-muted-foreground">
                  Security violations detected
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="plans" className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {plans.map((plan) => (
              <Card 
                key={plan.id} 
                className={`relative ${currentPlan?.id === plan.id ? 'ring-2 ring-primary' : ''}`}
              >
                {currentPlan?.id === plan.id && (
                  <div className="absolute -top-2 left-1/2 transform -translate-x-1/2">
                    <Badge className="bg-primary">Current Plan</Badge>
                  </div>
                )}
                
                <CardHeader>
                  <CardTitle className="flex items-center justify-between">
                    {plan.display_name}
                    <div className="text-right">
                      <div className="text-2xl font-bold">
                        {formatPrice(plan.price_monthly)}
                      </div>
                      <div className="text-sm text-muted-foreground">/month</div>
                    </div>
                  </CardTitle>
                  <CardDescription>{plan.description}</CardDescription>
                </CardHeader>
                
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <div className="flex items-center gap-2">
                      <Users className="h-4 w-4" />
                      <span className="text-sm">
                        {plan.limits.users === -1 ? 'Unlimited' : plan.limits.users} users
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Zap className="h-4 w-4" />
                      <span className="text-sm">
                        {plan.limits.api_calls_monthly === -1 ? 'Unlimited' : plan.limits.api_calls_monthly.toLocaleString()} API calls/month
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      <Shield className="h-4 w-4" />
                      <span className="text-sm">
                        {plan.limits.llm_calls_monthly === -1 ? 'Unlimited' : plan.limits.llm_calls_monthly.toLocaleString()} LLM calls/month
                      </span>
                    </div>
                  </div>

                  <div className="space-y-1">
                    {plan.features.compliance_reporting.map((framework) => (
                      <div key={framework} className="flex items-center gap-2">
                        <CheckCircle className="h-3 w-3 text-green-500" />
                        <span className="text-xs">{framework} compliance</span>
                      </div>
                    ))}
                    {plan.features.advanced_analytics && (
                      <div className="flex items-center gap-2">
                        <CheckCircle className="h-3 w-3 text-green-500" />
                        <span className="text-xs">Advanced analytics</span>
                      </div>
                    )}
                    {plan.features.priority_processing && (
                      <div className="flex items-center gap-2">
                        <CheckCircle className="h-3 w-3 text-green-500" />
                        <span className="text-xs">Priority processing</span>
                      </div>
                    )}
                  </div>

                  {currentPlan?.id !== plan.id && (
                    <Button 
                      className="w-full" 
                      onClick={() => {
                        setSelectedPlan(plan.id);
                        createSubscriptionMutation.mutate({
                          plan_id: plan.id,
                          billing_cycle: billingCycle,
                        });
                      }}
                      disabled={createSubscriptionMutation.isPending}
                    >
                      {createSubscriptionMutation.isPending ? 'Processing...' : 'Select Plan'}
                    </Button>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="usage" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Usage Details</CardTitle>
              <CardDescription>
                Current usage for this billing period
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {apiQuota && (
                  <div>
                    <div className="flex justify-between text-sm mb-1">
                      <span>API Calls</span>
                      <span>{apiQuota.used.toLocaleString()} / {apiQuota.is_unlimited ? '∞' : apiQuota.limit.toLocaleString()}</span>
                    </div>
                    <div className="w-full bg-gray-200 rounded-full h-2">
                      <div 
                        className="bg-blue-600 h-2 rounded-full" 
                        style={{ 
                          width: apiQuota.is_unlimited ? '0%' : `${Math.min((apiQuota.used / apiQuota.limit) * 100, 100)}%` 
                        }}
                      ></div>
                    </div>
                  </div>
                )}

                {llmQuota && (
                  <div>
                    <div className="flex justify-between text-sm mb-1">
                      <span>LLM Calls</span>
                      <span>{llmQuota.used.toLocaleString()} / {llmQuota.is_unlimited ? '∞' : llmQuota.limit.toLocaleString()}</span>
                    </div>
                    <div className="w-full bg-gray-200 rounded-full h-2">
                      <div 
                        className="bg-green-600 h-2 rounded-full" 
                        style={{ 
                          width: llmQuota.is_unlimited ? '0%' : `${Math.min((llmQuota.used / llmQuota.limit) * 100, 100)}%` 
                        }}
                      ></div>
                    </div>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="history" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>Billing History</CardTitle>
              <CardDescription>
                Your past invoices and payments
              </CardDescription>
            </CardHeader>
            <CardContent>
              {historyLoading ? (
                <div className="text-center py-8">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
                  <p className="text-muted-foreground">Loading billing history...</p>
                </div>
              ) : (billingHistory?.billing_history?.length ?? 0) > 0 ? (
                <div className="space-y-4">
                  {billingHistory?.billing_history?.map((bill: any) => (
                    <div key={bill.id} className="flex items-center justify-between p-4 border rounded-lg">
                      <div>
                        <div className="font-medium">
                          {format(new Date(bill.billing_period_start), 'MMM dd')} - {format(new Date(bill.billing_period_end), 'MMM dd, yyyy')}
                        </div>
                        <div className="text-sm text-muted-foreground">
                          {bill.status}
                        </div>
                      </div>
                      <div className="text-right">
                        <div className="font-medium">
                          {formatPrice(bill.total_amount)}
                        </div>
                        <div className="text-sm text-muted-foreground">
                          {bill.usage_amount > 0 && `+${formatPrice(bill.usage_amount)} usage`}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8">
                  <DollarSign className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                  <p className="text-muted-foreground">No billing history available</p>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
