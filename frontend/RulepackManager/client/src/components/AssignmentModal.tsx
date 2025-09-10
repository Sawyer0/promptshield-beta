import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Modal } from '@/components/common/Modal';
import { Button } from '@/components/ui/button';
import { ToggleField } from '@/components/common/FormFields';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { Switch } from '@/components/ui/switch';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { Input } from '@/components/ui/input';
import { Link2, Shield, Zap, AlertTriangle, Info } from 'lucide-react';
import type { RulePack } from '@shared/apiTypes';

type AssignmentFormData = z.infer<typeof assignmentModalSchema>;

interface AssignmentModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (data: { rulepackId: string; method: string; targetScope: string; priority: number; enabled: boolean }) => void;
  isLoading?: boolean;
  rulePacks: RulePack[];
  assignment?: any; // For editing
}

// Priority UX mapping (labels → numeric)
const PRIORITY_MAP: Record<string, number> = {
  Highest: 1000,
  High: 750,
  Medium: 500,
  Low: 250,
  Lowest: 0,
};

// Parse endpoints input (one per line or comma-separated) → unique, trimmed list
function parseEndpoints(input: string): string[] {
  return Array.from(
    new Set(
      input
        .split(/\r?\n|,/)
        .map((s) => s.trim())
        .filter((s) => s.length > 0)
    )
  );
}

// Basic endpoint validator: must start with '/', allow letters, digits, '-', '_', '/', and optional trailing '*'
function isValidEndpoint(ep: string): boolean {
  if (!ep.startsWith('/')) return false;
  if (ep.includes('://')) return false; // disallow full URLs
  if (!/^\/[A-Za-z0-9_\/-]*\*?$/.test(ep)) return false;
  // if '*' present, only allowed at end and as a single wildcard
  const starIdx = ep.indexOf('*');
  if (starIdx !== -1 && starIdx !== ep.length - 1) return false;
  return true;
}

const assignmentModalSchema = z.object({
  rulepackId: z.string().min(1, 'RulePack is required'),
  method: z.enum(['*','GET','POST','PUT','PATCH','DELETE','HEAD','OPTIONS']).default('*'),
  endpoints: z
    .array(z.string().min(1))
    .min(1, 'Add at least one endpoint')
    .refine((arr) => arr.every((ep) => isValidEndpoint(ep)), 'Endpoints must be valid paths (e.g., /, /v1/*, /api/orders/*, /api/payments/refund)'),
  priorityLabel: z.enum(['Highest', 'High', 'Medium', 'Low', 'Lowest']).default('Medium'),
  enabled: z.boolean().default(true),
});

export function AssignmentModal({ 
  isOpen, 
  onClose, 
  onSubmit, 
  isLoading, 
  rulePacks,
  assignment 
}: AssignmentModalProps) {
  const [showOverlapWarning, setShowOverlapWarning] = useState(false);
  // Keep a local mirror for method to ensure Radix Select interactions update RHF state reliably in tests
  const [methodValue, setMethodValue] = useState<string>(assignment?.method || '*');

  const form = useForm<AssignmentFormData>({
    resolver: zodResolver(assignmentModalSchema),
    mode: 'onChange',
    reValidateMode: 'onChange',
    defaultValues: {
      rulepackId: assignment?.rulepackId || '',
      method: assignment?.method || '*',
      endpoints: assignment?.targetScope ? [String(assignment.targetScope)] : [],
      priorityLabel: 'Medium',
      enabled: assignment?.enabled !== false,
    },
  });

  const selectedRulePack = rulePacks.find(rp => rp.id === form.watch('rulepackId'));

  const handleSubmit = (data: AssignmentFormData) => {
    const endpoints = Array.from(new Set((data.endpoints || []).map((s) => s.trim()).filter(Boolean))).filter(isValidEndpoint);
    const priority = PRIORITY_MAP[data.priorityLabel];
    const method = data.method || methodValue || '*';
    if (assignment) {
      const first = endpoints[0];
      if (first) {
        onSubmit({ rulepackId: data.rulepackId, method, targetScope: first, priority, enabled: data.enabled });
      }
    } else {
      endpoints.forEach((ep) => {
        onSubmit({ rulepackId: data.rulepackId, method, targetScope: ep, priority, enabled: data.enabled });
      });
    }
  };

  const handleClose = () => {
    form.reset();
    setShowOverlapWarning(false);
    onClose();
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      size="lg"
      title={
        <div className="flex items-center">
          <Link2 className="h-5 w-5 mr-2" />
          {assignment ? 'Edit RulePack Assignment' : 'Assign RulePack to Endpoints'}
        </div>
      }
      description={assignment ? 'Modify the rulepack assignment.' : 'Assign a security rulepack to one or more API endpoints with clear priority.'}
      contentClassName="max-h-[90vh] overflow-y-auto"
    >

        <Form {...form}>
          <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-6">
            {/* RulePack Selection */}
            <FormField
              control={form.control}
              name="rulepackId"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="flex items-center">
                    <Shield className="w-4 h-4 mr-1" />
                    RulePack *
                  </FormLabel>
                  <Select onValueChange={field.onChange} value={field.value}>
                    <FormControl>
                      <SelectTrigger data-testid="select-rulepack">
                        <SelectValue placeholder="Select a rulepack to assign" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {rulePacks.map(rulepack => (
                        <SelectItem key={rulepack.id} value={rulepack.id}>
                          <div className="flex items-center justify-between w-full">
                            <div>
                              <div className="font-medium">{rulepack.name}</div>
                              {rulepack.description && (
                                <div className="text-xs text-muted-foreground">{rulepack.description}</div>
                              )}
                            </div>
                            <Badge variant={rulepack.isActive ? 'default' : 'secondary'} className="ml-2">
                              v{rulepack.currentVersionId || '1.0'}
                            </Badge>
                          </div>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                  {selectedRulePack && (
                    <div className="text-xs text-muted-foreground mt-1">
                      Priority: {selectedRulePack.priority} | Mode: {selectedRulePack.enforcementMode}
                    </div>
                  )}
                </FormItem>
              )}
            />

            {/* HTTP Method */}
            <FormField
              control={form.control}
              name="method"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="flex items-center">HTTP Method</FormLabel>
                  <Select
                    onValueChange={(val) => {
                      setMethodValue(val);
                      field.onChange(val);
                    }}
                    value={field.value ?? methodValue}
                  >
                    <FormControl>
                      <SelectTrigger data-testid="select-method">
                        <SelectValue placeholder="* (Any)" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      {['*','GET','POST','PUT','PATCH','DELETE','HEAD','OPTIONS'].map((m) => (
                        <SelectItem
                          key={m}
                          value={m}
                          onClick={() => {
                            setMethodValue(m);
                            form.setValue('method', m as any, { shouldValidate: true, shouldDirty: true });
                          }}
                        >
                          {m === '*' ? 'Any (*)' : m}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            {/* Endpoints (multiple) */}
            <FormField
              control={form.control}
              name="endpoints"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="flex items-center">
                    Endpoints *
                  </FormLabel>
                  <div className="space-y-2">
                    <FormControl>
                      <Input
                        placeholder="Type an endpoint and press Enter (use commas for multiple)"
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ',') {
                            e.preventDefault();
                            const input = e.currentTarget;
                            const value = input.value.trim();
                            if (value) {
                              const parts = value.split(',').map((p) => p.trim()).filter(Boolean);
                              const current = Array.isArray(form.getValues('endpoints')) ? form.getValues('endpoints')! : [];
                              const merged = Array.from(new Set([...current, ...parts]));
                              form.setValue('endpoints', merged, { shouldValidate: true });
                              input.value = '';
                            }
                          }
                        }}
                        data-testid="input-endpoint-chip"
                      />
                    </FormControl>
                    <p className="text-xs text-muted-foreground">Use exact paths or wildcards (e.g., /, /v1/*, /api/orders/*). Paths must start with /.</p>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {Array.isArray(form.watch('endpoints')) && form.watch('endpoints')!.length > 0 ? (
                        form.watch('endpoints')!.map((ep, idx) => (
                          <Badge key={`${idx}-${ep}`} variant="secondary" className="text-xs">
                            {ep}
                            <button
                              type="button"
                              onClick={() => {
                                const cur = form.getValues('endpoints') || [];
                                const next = cur.filter((_, i) => i !== idx);
                                form.setValue('endpoints', next, { shouldValidate: true });
                              }}
                              className="ml-1 hover:text-destructive"
                            >
                              ×
                            </button>
                          </Badge>
                        ))
                      ) : null}
                    </div>
                    <FormMessage />
                  </div>
                </FormItem>
              )}
            />

            {/* Priority and Enabled Status */}
            <div className="grid grid-cols-2 gap-4">
              <FormField
                control={form.control}
                name="priorityLabel"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className="flex items-center"><Zap className="w-4 h-4 mr-1" /> Priority</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                      <FormControl>
                        <SelectTrigger data-testid="select-priority">
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {Object.keys(PRIORITY_MAP).map((label) => (
                          <SelectItem key={label} value={label}>{label}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <div className="text-xs text-muted-foreground mt-1">Higher priority applies first when multiple assignments match.</div>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <ToggleField
                control={form.control}
                name="enabled"
                label="Enabled"
                description={<span className="text-xs">Toggle enforcement on/off (default: on)</span>}
                data-testid="switch-enabled"
              />
            </div>

            {/* Overlap Warning */}
            {showOverlapWarning && (
              <Alert className="border-orange-200 bg-orange-50 dark:bg-orange-950 dark:border-orange-800">
                <AlertTriangle className="h-4 w-4 text-orange-600" />
                <AlertDescription className="text-orange-800 dark:text-orange-200">
                  <strong>Priority Conflict:</strong> Another assignment for this scope has equal or higher priority. 
                  Consider adjusting the priority to ensure proper enforcement order.
                </AlertDescription>
              </Alert>
            )}

            {/* Info Box */}
            <Alert className="border-blue-200 bg-blue-50 dark:bg-blue-950 dark:border-blue-800">
              <Info className="h-4 w-4 text-blue-600" />
              <AlertDescription className="text-blue-800 dark:text-blue-200">
                <strong>Note:</strong> Tenant is automatically pulled from your session context. 
                One assignment per (tenant, target scope, rulepack) combination is enforced by the system.
              </AlertDescription>
            </Alert>

            {/* Assignment Preview */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Assignment Preview</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div>
                  <span className="text-sm font-medium text-muted-foreground">RulePack:</span>
                  <p className="text-sm">{selectedRulePack?.name || 'No rulepack selected'}</p>
                  {selectedRulePack?.description && (
                    <p className="text-xs text-muted-foreground">{selectedRulePack.description}</p>
                  )}
                </div>
                <Separator />
                <div>
                  <span className="text-sm font-medium text-muted-foreground">Method:</span>
                  <p className="text-sm font-mono">{form.watch('method') || '*'}</p>
                </div>
                <Separator />
                <div>
                  <span className="text-sm font-medium text-muted-foreground">Endpoints:</span>
                  <pre className="text-xs font-mono whitespace-pre-wrap break-words bg-muted/30 p-2 rounded">
                    {(form.watch('endpoints') || []).join('\n') || '—'}
                  </pre>
                </div>
                <Separator />
                <div className="flex items-center justify-between">
                  <div>
                    <span className="text-sm font-medium text-muted-foreground">Priority:</span>
                    <p className="text-sm font-bold">{form.watch('priorityLabel')}</p>
                  </div>
                  <div>
                    <span className="text-sm font-medium text-muted-foreground">Status:</span>
                    <Badge variant={form.watch('enabled') ? 'default' : 'secondary'}>
                      {form.watch('enabled') ? 'Enabled' : 'Disabled'}
                    </Badge>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Form Actions */}
            <div className="flex items-center justify-end space-x-4 pt-6 border-t">
              <Button type="button" variant="outline" onClick={handleClose} data-testid="button-cancel">
                Cancel
              </Button>
              <Button 
                type="submit" 
                disabled={isLoading}
                data-testid="button-submit-assignment"
              >
                {isLoading ? "Creating..." : assignment ? "Update Assignment" : "Create Assignment"}
              </Button>
            </div>
          </form>
        </Form>
    </Modal>
  );
}
