import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { X, Plus, Trash2, Code2, Brain } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useToast } from "@/hooks/use-toast";
import { Form } from "@/components/ui/form";
import { Modal } from "@/components/common/Modal";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "@/components/ui/select";
import type { Rule, RulePackMetadata } from "@shared/apiTypes";

const ruleSchema = z.object({
  id: z.string().min(1, "Rule ID is required"),
  name: z.string().min(1, "Rule name is required"),
  description: z.string().optional(),
  level: z.union([z.literal(1), z.literal(2), z.literal(3)]).default(1),
  action: z.enum(["observe", "redact", "quarantine", "deny"]),
  severity: z.enum(["low", "medium", "high", "critical"]),
  keywords: z.array(z.string()).optional(),
  pattern: z.array(z.string()).optional(),
  semantic_analysis: z.boolean().optional(),
  controls: z.array(z.string()).optional(),
});

const rulePackSchema = z.object({
  name: z.string().min(1, "RulePack name is required"),
  description: z.string().optional(),
  status: z.enum(["active", "draft", "inactive"]).default("active"),
  enforcementMode: z.enum(["monitor", "enforce", "redact"]).default("monitor"),
  failOnSeverity: z.enum(["LOW", "MEDIUM", "HIGH", "CRITICAL"]).default("HIGH"),
  priority: z.number().min(1).max(10).default(1),
  isActive: z.boolean().default(true),
});

type RulePackFormData = z.infer<typeof rulePackSchema>;

interface RulePackModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (data: any) => void;
  isLoading?: boolean;
}

export function RulePackModal({ isOpen, onClose, onSubmit, isLoading }: RulePackModalProps) {
  const { toast } = useToast();
  type UIRule = Partial<Rule> & { controls?: string[] };
  const [rules, setRules] = useState<UIRule[]>([
    {
      id: "rule-001",
      name: "",
      description: "",
      level: 1,
      action: "observe",
      severity: "medium",
      keywords: [] as string[],
      pattern: [] as string[],
      semantic_analysis: false,
      controls: ["OWASP_LLM.LLM01", "SOC2.CC7.2", "GDPR.Art_32"],
    },
  ]);
  const form = useForm<RulePackFormData>({
    resolver: zodResolver(rulePackSchema),
    defaultValues: {
      // Metadata fields hidden; keep defaults (not shown)
      name: "",
      description: "",
      status: "active" as const,
      enforcementMode: "monitor" as const,
      failOnSeverity: "HIGH" as const,
      priority: 1,
      isActive: true,
    },
  });

  const addRule = () => {
    const newRule: UIRule = {
      id: `rule-${String(rules.length + 1).padStart(3, '0')}`,
      name: "",
      description: "",
      level: 1,
      action: "observe",
      severity: "medium",
      keywords: [] as string[],
      pattern: [] as string[],
      semantic_analysis: false,
      controls: ["OWASP_LLM.LLM01"],
    };
    setRules([...rules, newRule]);
  };

  const removeRule = (index: number) => {
    if (rules.length > 1) {
      const newRules = rules.filter((_, i) => i !== index);
      setRules(newRules);
    }
  };

  const updateRule = (index: number, field: keyof UIRule, value: any) => {
    const newRules = [...rules];
    (newRules[index] as any)[field] = value;
    setRules(newRules);
  };


  const strictnessToThreshold = (s?: number) => {
    // Map 1..10 (low..high strictness) to threshold 0.95..0.50 (higher strictness -> lower threshold)
    const v = typeof s === 'number' ? Math.min(10, Math.max(1, s)) : 7; // default 7 ~ 0.66
    const maxT = 0.95, minT = 0.50;
    const step = (maxT - minT) / 9;
    return maxT - (v - 1) * step;
  };

  const handleSubmit = (data: RulePackFormData) => {
    // Validate rules: require id, name, severity, action and at least one detection (L1/L2/L3)
    const problems: string[] = [];
    if (!rules.length) problems.push('At least one rule is required.');
    rules.forEach((r, i) => {
      const label = `Rule ${i + 1}`;
      if (!r.name || !r.name.trim()) problems.push(`${label}: name is required`);
      if (!r.id || !r.id.trim()) problems.push(`${label}: id is required`);
      const hasKeywords = Array.isArray(r.keywords) && r.keywords.filter(Boolean).length > 0;
      const hasPatterns = Array.isArray(r.pattern) && r.pattern.filter(Boolean).length > 0;
      const l3 = !!r.semantic_analysis;
      if (!hasKeywords && !hasPatterns && !l3) problems.push(`${label}: add keywords, regex patterns, or enable Semantic Analysis`);
      if (l3) {
        // Categories are optional; ProtectAI DeBERTa v2 scores across its taxonomy automatically.
      }
    });
    if (problems.length) {
      toast({ title: 'Please fix the issues below', description: problems.join('\n'), variant: 'destructive' });
      return;
    }
    // Map to backend DSL schema (JSON acceptable by YAML parser)
    const mappedRules = rules
      .filter(rule => rule.name && rule.name.trim() !== '' && rule.id && rule.id.trim() !== '')
      .map((r, i) => {
        const out: any = { id: r.id, name: r.name, description: r.description || '' }
        // severity mapping
        const sevMap: Record<string,string> = { low: 'WARNING', medium: 'HIGH', high: 'ERROR', critical: 'CRITICAL' }
        out.severity = sevMap[(r.severity || 'medium').toLowerCase()] || 'HIGH'
        // response action mapping
        const actMap: Record<string,string> = { observe: 'allow', redact: 'redact', quarantine: 'quarantine', deny: 'deny' }
        out.response = { action: actMap[(r.action || 'observe').toLowerCase()] || 'allow' }

        // Controls tagging (compliance mapping)
        const controls = Array.isArray((r as any).controls) ? (r as any).controls.filter(Boolean) : []
        if (controls.length) {
          out.controls = controls
        }

        if ((r.level || 1) === 1) {
          out.level = 1
          out.keywords = Array.isArray(r.keywords) ? r.keywords.filter(Boolean) : []
        } else if ((r.level || 1) === 2) {
          out.level = 2
          const patterns = Array.isArray(r.pattern) ? r.pattern.filter(Boolean) : []
          out.patterns = patterns.map((p: string, idx: number) => ({ name: `pattern-${idx+1}`, regex: p }))
        } else {
          out.level = 3
          out.semantic = {
            engine: 'protectai',
            model: 'protectai-deberta-v2',
            confidence_threshold: strictnessToThreshold(undefined),
            fallback_on_error: true,
            inputs: ['text'],
          }
        }
        return out
      })

    const autoName = data.name && data.name.trim() ? data.name.trim() : `rulepack-${new Date().toISOString().replace(/[:.]/g,'-')}`
    const dsl = {
      apiVersion: 'promptshield/v1',
      kind: 'RulePack',
      metadata: {
        name: autoName,
        version: '1.0',
        description: '',
      },
      rules: mappedRules,
      composition: { strategy: 'priority', priority: 1 },
      patterns: {
        action_selector: { enabled: true, mode: 'enforce', allowed_tool_query: 'read AND (network_get OR file-io_read) AND NOT write' },
        context_minimization: { enabled: true, strip_point: 'after_tool_selection', mask_token: '<USER_TEXT>' },
        plan_then_execute: { enabled: true, max_steps: 8, drift_policy: 'block' },
      },
      preset: {
        preset_id: 'agent_safe_defaults',
        mode: 'enforce',
      }
    }

    onSubmit(dsl)
    // Recommend semantic analysis if none of the rules enabled it
    if (!rules.some(r => r.semantic_analysis)) {
      toast({ title: 'Tip', description: 'Consider enabling Level 3 Semantic Analysis for better coverage.' });
    }
  }

  const handleClose = () => {
    form.reset();
    setRules([{
      id: "rule-001",
      name: "",
      description: "",
      level: 1,
      action: "observe",
      severity: "medium",
      keywords: [] as string[],
      pattern: [] as string[],
      semantic_analysis: false,
      controls: ["OWASP_LLM.LLM01", "SOC2.CC7.2", "GDPR.Art_32"],
    }]);
    onClose();
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={handleClose}
      size="xl"
      title={"Create New RulePack"}
      description={"Create a new security rule pack with custom rules and configurations."}
      contentClassName="max-h-[90vh] overflow-y-auto"
    >
      {/* RBAC banner */}
      {(() => { try { const raw = localStorage.getItem('ps_roles'); const r = raw ? JSON.parse(raw) : []; const can = Array.isArray(r) && (r.includes('tenant_admin') || r.includes('security_engineer')); return can ? null : (<div className="px-4 pt-4 text-xs text-muted-foreground">Read-only preview: you do not have permission to create or edit rules.</div>); } catch { return null; } })()}
      <Form {...form}>
          <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-6">
            {/* No metadata section — rulepack will auto-name and auto-activate */}

            {/* Rules Section */}
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-medium text-foreground">Rules</h3>
                <Button type="button" onClick={addRule} size="sm" data-testid="button-add-rule">
                  <Plus className="mr-1 h-4 w-4" />
                  Add Rule
                </Button>
              </div>
              
              <div className="space-y-4">
                {rules.map((rule, index) => (
                  <div key={index} className="bg-muted rounded-lg p-4 border border-border">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <div>
                        <label className="block text-sm font-medium text-foreground mb-1">Rule Name *</label>
                        <Input
                          value={rule.name || ""}
                          onChange={(e) => updateRule(index, 'name', e.target.value)}
                          placeholder="e.g., Prompt Injection Detection"
                          data-testid={`input-rule-name-${index}`}
                          className={!rule.name ? "border-red-300" : ""}
                        />
                        {!rule.name && <p className="text-xs text-red-500 mt-1">Rule name is required</p>}
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-foreground mb-1">Rule ID *</label>
                        <Input
                          value={rule.id || ""}
                          onChange={(e) => updateRule(index, 'id', e.target.value)}
                          placeholder={`e.g., rule-${String(index + 1).padStart(3, '0')}`}
                          data-testid={`input-rule-id-${index}`}
                          className={!rule.id ? "border-red-300" : ""}
                        />
                        {!rule.id && <p className="text-xs text-red-500 mt-1">Rule ID is required</p>}
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-foreground mb-1">Severity</label>
                        <Select 
                          value={rule.severity || "medium"} 
                          onValueChange={(value) => updateRule(index, 'severity', value)}
                        >
                          <SelectTrigger data-testid={`select-rule-severity-${index}`}>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="low">Low</SelectItem>
                            <SelectItem value="medium">Medium</SelectItem>
                            <SelectItem value="high">High</SelectItem>
                            <SelectItem value="critical">Critical</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-foreground mb-1">Action</label>
                        <Select 
                          value={rule.action || "observe"} 
                          onValueChange={(value) => updateRule(index, 'action', value)}
                        >
                          <SelectTrigger data-testid={`select-rule-action-${index}`}>
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="observe">Observe</SelectItem>
                            <SelectItem value="redact">Redact</SelectItem>
                            <SelectItem value="quarantine">Quarantine</SelectItem>
                            <SelectItem value="deny">Deny</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                    <div className="mt-4">
                      <label className="block text-sm font-medium text-foreground mb-1">Description</label>
                      <Textarea
                        value={rule.description || ""}
                        onChange={(e) => updateRule(index, 'description', e.target.value)}
                        placeholder="Describe what this rule detects..."
                        rows={2}
                        data-testid={`input-rule-description-${index}`}
                      />
                    </div>
                    {/* Detection Configuration */}
                    <div className="mt-4 space-y-4">
                      <h4 className="text-sm font-semibold text-foreground flex items-center">
                        <Brain className="mr-2 h-4 w-4" />
                        Detection Configuration
                      </h4>
                      
                      {/* Keywords Section */}
                      <div>
                        <label className="block text-sm font-medium text-foreground mb-2">Keywords</label>
                        <div className="space-y-2">
                          <Input
                            placeholder="Type keywords separated by commas, then press Enter"
                            onKeyDown={(e) => {
                              if (e.key === 'Enter' || e.key === ',') {
                                e.preventDefault();
                                const input = e.currentTarget;
                                const value = input.value.trim();
                                
                                if (value) {
                                  // Split by comma and add all keywords
                                  const newKeywords = value
                                    .split(',')
                                    .map(k => k.trim())
                                    .filter(k => k.length > 0);
                                  
                                  const currentKeywords = Array.isArray(rule.keywords) ? rule.keywords : [];
                                  const allKeywords = [...currentKeywords, ...newKeywords];
                                  
                                  updateRule(index, 'keywords', allKeywords);
                                  input.value = '';
                                }
                              }
                            }}
                            data-testid={`input-rule-keywords-${index}`}
                          />
                          <p className="text-xs text-muted-foreground">
                            Type keywords and press Enter to add them. You can also separate multiple keywords with commas.
                          </p>
                          <div className="flex flex-wrap gap-1 mt-2">
                            {Array.isArray(rule.keywords) && rule.keywords.length > 0 ? (
                              rule.keywords.map((keyword, kidx) => (
                                <Badge key={`${index}-${kidx}-${keyword}`} variant="secondary" className="text-xs">
                                  {keyword}
                                  <button 
                                    type="button"
                                    onClick={() => {
                                      const newKeywords = rule.keywords?.filter((_, i) => i !== kidx) || [];
                                      updateRule(index, 'keywords', newKeywords);
                                    }}
                                    className="ml-1 hover:text-destructive"
                                  >
                                    ×
                                  </button>
                                </Badge>
                              ))
                            ) : null}
                          </div>
                        </div>
                      </div>

                      {/* Controls (Compliance Tags) */}
                      <div>
                        <label className="block text-sm font-medium text-foreground mb-2">Controls (Compliance Tags)</label>
                        <div className="space-y-2">
                          <Input
                            placeholder="Add control tags like OWASP_LLM.LLM01, SOC2.CC7.2 (comma or Enter to add)"
                            onKeyDown={(e) => {
                              if (e.key === 'Enter' || e.key === ',') {
                                e.preventDefault();
                                const input = e.currentTarget as HTMLInputElement;
                                const value = input.value.trim();
                                if (value) {
                                  const newTags = value.split(',').map(v => v.trim()).filter(Boolean);
                                  const current = Array.isArray((rule as any).controls) ? (rule as any).controls as string[] : [];
                                  const all = [...current, ...newTags];
                                  updateRule(index, 'controls', all);
                                  input.value = '';
                                }
                              }
                            }}
                            data-testid={`input-rule-controls-${index}`}
                          />
                          <p className="text-xs text-muted-foreground">Examples: OWASP_LLM.LLM01, SOC2.CC7.2, HIPAA.164.312(b), GDPR.Art_32, NIST_AI_RMF.MAN.3</p>
                          <div className="flex flex-wrap gap-1 mt-2">
                            {Array.isArray((rule as any).controls) && (rule as any).controls.length > 0 ? (
                              ((rule as any).controls as string[]).map((tag: string, tIdx: number) => (
                                <Badge key={`${index}-ctl-${tIdx}-${tag}`} variant="secondary" className="text-xs">
                                  {tag}
                                  <button
                                    type="button"
                                    onClick={() => {
                                      const curr: string[] = Array.isArray((rule as any).controls) ? (rule as any).controls : [];
                                      const next = curr.filter((_, i) => i !== tIdx);
                                      updateRule(index, 'controls', next);
                                    }}
                                    className="ml-1 hover:text-destructive"
                                  >
                                    ×
                                  </button>
                                </Badge>
                              ))
                            ) : null}
                          </div>
                        </div>
                      </div>

                      {/* Regex Pattern Section */}
                      <div>
                        <label className="text-sm font-medium text-foreground mb-2 flex items-center">
                          <Code2 className="mr-1 h-3 w-3" />
                          Regex Pattern (Advanced)
                        </label>
                        <div className="space-y-2">
                          <Input
                            placeholder="Type regex patterns separated by commas, then press Enter"
                            onKeyDown={(e) => {
                              if (e.key === 'Enter' || e.key === ',') {
                                e.preventDefault();
                                const input = e.currentTarget;
                                const value = input.value.trim();
                                
                                if (value) {
                                  // Split by comma and add all patterns
                                  const newPatterns = value
                                    .split(',')
                                    .map(p => p.trim())
                                    .filter(p => p.length > 0);
                                  
                                  const currentPatterns = Array.isArray(rule.pattern) ? rule.pattern : [];
                                  const allPatterns = [...currentPatterns, ...newPatterns];
                                  
                                  updateRule(index, 'pattern', allPatterns);
                                  input.value = '';
                                }
                              }
                            }}
                            data-testid={`input-rule-pattern-${index}`}
                          />
                          <p className="text-xs text-muted-foreground">
                            Type regex patterns and press Enter to add them. You can also separate multiple patterns with commas.
                          </p>
                          <div className="flex flex-wrap gap-1 mt-2">
                            {Array.isArray(rule.pattern) && rule.pattern.length > 0 ? (
                              rule.pattern.map((pattern, pidx) => (
                                <Badge key={`${index}-${pidx}-${pattern}`} variant="secondary" className="text-xs">
                                  {pattern}
                                  <button 
                                    type="button"
                                    onClick={() => {
                                      const newPatterns = rule.pattern?.filter((_, i) => i !== pidx) || [];
                                      updateRule(index, 'pattern', newPatterns);
                                    }}
                                    className="ml-1 hover:text-destructive"
                                  >
                                    ×
                                  </button>
                                </Badge>
                              ))
                            ) : null}
                          </div>
                        </div>
                      </div>

                      {/* Semantic Analysis Section (L3) */}
                      <div>
                        <label className="text-sm font-medium text-foreground mb-2 flex items-center">
                          <Brain className="mr-1 h-3 w-3" />
                          Semantic Analysis
                        </label>
                        <div className="flex items-center space-x-2">
                          <Switch
                            checked={rule.semantic_analysis || false}
                            onCheckedChange={(checked) => updateRule(index, 'semantic_analysis', checked)}
                            data-testid={`toggle-rule-semantic-${index}`}
                          />
                          <span className="text-sm text-muted-foreground">
                            Enable Level 3 analysis
                          </span>
                        </div>
                        {rule.semantic_analysis ? (
                          <div className="mt-3 space-y-3">
                            {/* Engine selector */}
                            <div>
                              <label className="block text-sm font-medium text-foreground mb-1">Engine</label>
                              <div className="text-sm">ProtectAI DeBERTa v2 (no API key required)</div>
                            </div>

                            {/* Inputs are fixed to Text and strictness uses a sensible default. */}
                          </div>
                        ) : null}
                        <p className="text-xs text-muted-foreground mt-1">
Semantic analysis uses ProtectAI DeBERTa v2. No API key required.
                        </p>
                      </div>
                    </div>
                    {rules.length > 1 && (
                      <div className="mt-4 flex justify-end">
                        <Button 
                          type="button" 
                          variant="destructive" 
                          size="sm"
                          onClick={() => removeRule(index)}
                          data-testid={`button-remove-rule-${index}`}
                        >
                          <Trash2 className="mr-1 h-4 w-4" />
                          Remove Rule
                        </Button>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>

            {/* Form Actions */}
            <div className="flex items-center justify-end space-x-4 pt-6 border-t border-border">
              <Button type="button" variant="outline" onClick={handleClose} data-testid="button-cancel">
                Cancel
              </Button>
              <Button 
                type="button"
                onClick={() => form.handleSubmit(handleSubmit)()} 
                disabled={isLoading || !(() => { try { const raw = localStorage.getItem('ps_roles'); const r = raw ? JSON.parse(raw) : []; return Array.isArray(r) && (r.includes('tenant_admin') || r.includes('security_engineer')); } catch { return false; } })()} 
                data-testid="button-create-rulepack"
                title={(() => { try { const raw = localStorage.getItem('ps_roles'); const r = raw ? JSON.parse(raw) : []; const can = Array.isArray(r) && (r.includes('tenant_admin') || r.includes('security_engineer')); return can ? undefined : 'Read-only: insufficient permissions'; } catch { return undefined; } })()}
              >
                {isLoading ? "Creating..." : "Create RulePack"}
              </Button>
            </div>
          </form>
        </Form>
    </Modal>
  );
}
