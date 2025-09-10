import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { X, Plus, Trash2, Code2, Brain } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useToast } from "@/hooks/use-toast";
import { TextField, NumberField, TextAreaField, ToggleField, MultiSelectChips, SliderField } from "@/components/common/FormFields";
import { SelectField } from "@/components/common/SelectField";
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from "@/components/ui/form";
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
  const [rules, setRules] = useState<Partial<Rule>[]>([
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
    const newRule: Partial<Rule> = {
      id: `rule-${String(rules.length + 1).padStart(3, '0')}`,
      name: "",
      description: "",
      level: 1,
      action: "observe",
      severity: "medium",
      keywords: [] as string[],
      pattern: [] as string[],
      semantic_analysis: false,
    };
    setRules([...rules, newRule]);
  };

  const removeRule = (index: number) => {
    if (rules.length > 1) {
      const newRules = rules.filter((_, i) => i !== index);
      setRules(newRules);
    }
  };

  const updateRule = (index: number, field: keyof Rule, value: any) => {
    const newRules = [...rules];
    newRules[index] = { ...newRules[index], [field]: value };
    setRules(newRules);
  };

  const SEMANTIC_TEXT_ONLY = [
    { value: 'harassment', label: 'Harassment (text)' },
    { value: 'harassment/threatening', label: 'Harassment/Threat (text)' },
    { value: 'hate', label: 'Hate (text)' },
    { value: 'hate/threatening', label: 'Hate/Threat (text)' },
    { value: 'illicit', label: 'Illicit (text)' },
    { value: 'illicit/violent', label: 'Illicit/Violent (text)' },
    { value: 'sexual/minors', label: 'Sexual/Minors (text)' },
  ]
  const SEMANTIC_TEXT_IMAGE = [
    { value: 'violence', label: 'Violence' },
    { value: 'violence/graphic', label: 'Violence/Graphic' },
    { value: 'sexual', label: 'Sexual' },
    { value: 'self-harm', label: 'Self‑harm' },
    { value: 'self-harm/intent', label: 'Self‑harm/Intent' },
    { value: 'self-harm/instructions', label: 'Self‑harm/Instructions' },
  ]
  const applyPreset = (idx: number, preset: 'default' | 'strict_minors' | 'violence_strict') => {
    const field = `rules.${idx}.semantic_categories` as any;
    if (preset === 'default') {
      form.setValue(field, ['violence','sexual','self-harm','harassment','hate']);
    } else if (preset === 'strict_minors') {
      form.setValue(field, ['sexual/minors','sexual','violence','violence/graphic']);
      form.setValue(`rules.${idx}.semantic_strictness` as any, 9);
    } else {
      form.setValue(field, ['violence','violence/graphic']);
      form.setValue(`rules.${idx}.semantic_strictness` as any, 8);
    }
  }

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
        const engine = (form.getValues() as any)?.rules?.[i]?.semantic_engine || 'omni';
        if (engine === 'custom' || engine === 'custom+omni') {
          const prof = (form.getValues() as any)?.rules?.[i]?.custom_provider_profile;
          const model = (form.getValues() as any)?.rules?.[i]?.custom_model;
          if (!prof || !String(prof).trim()) problems.push(`${label}: provider profile is required for Custom engine`);
          if (!model || !String(model).trim()) problems.push(`${label}: model is required for Custom engine`);
        }
        // For any engine using omni, ensure categories selected
        if (engine === 'omni' || engine === 'custom+omni') {
          const cats = (form.getValues() as any)?.rules?.[i]?.semantic_categories as string[] | undefined;
          if (!cats || cats.length === 0) problems.push(`${label}: select at least one Omni category`);
        }
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

        if ((r.level || 1) === 1) {
          out.level = 1
          out.keywords = Array.isArray(r.keywords) ? r.keywords.filter(Boolean) : []
        } else if ((r.level || 1) === 2) {
          out.level = 2
          const patterns = Array.isArray(r.pattern) ? r.pattern.filter(Boolean) : []
          out.patterns = patterns.map((p: string, idx: number) => ({ name: `pattern-${idx+1}`, regex: p }))
        } else {
          out.level = 3
          const values: any = (form.getValues() as any)?.rules?.[i] || {}
          const engine = values.semantic_engine || 'omni'
          if (engine === 'omni') {
            const inputs = values.semantic_inputs as string[] | undefined
            const categories = values.semantic_categories as string[] | undefined
            const strict = values.semantic_strictness as number | undefined
            const cats = categories && categories.length ? categories : ['violence','sexual','self-harm']
            const inps = inputs && inputs.length ? inputs : ['text']
            out.semantic = {
              engine: 'omni',
              model: 'omni-moderation-latest',
              confidence_threshold: strictnessToThreshold(strict),
              fallback_on_error: true,
              categories: cats,
              inputs: inps,
              analysis_prompt: 'omni:ignored',
            }
          } else if (engine === 'custom') {
            const strict = values.custom_strictness as number | undefined
            out.semantic = {
              engine: 'custom',
              provider_profile: values.custom_provider_profile || '',
              model: values.custom_model || 'gpt-4o-mini',
              analysis_prompt: values.custom_prompt || 'Return JSON {"flagged": boolean, "confidence": number, "categories": string[]}',
              temperature: Number(values.custom_temperature ?? 0),
              max_tokens: Number(values.custom_max_tokens ?? 256),
              confidence_threshold: strictnessToThreshold(strict),
              fallback_on_error: true,
              inputs: Array.isArray(values.semantic_inputs) && values.semantic_inputs.length ? values.semantic_inputs : ['text'],
            }
          } else {
            // ensemble
            const omniStrict = values.semantic_strictness as number | undefined
            const customStrict = values.custom_strictness as number | undefined
            const inputs = values.semantic_inputs as string[] | undefined
            const categories = values.semantic_categories as string[] | undefined
            const combineMode = values.combine_mode || 'or'
            const low = typeof values.combine_low === 'number' ? values.combine_low : 0.6
            const high = typeof values.combine_high === 'number' ? values.combine_high : 0.85
            const wOmni = typeof values.combine_weight_omni === 'number' ? values.combine_weight_omni : 0.6
            const wCustom = typeof values.combine_weight_custom === 'number' ? values.combine_weight_custom : 0.4
            out.semantic = {
              engine: 'ensemble',
              combine: {
                mode: combineMode,
                weights: combineMode === 'weighted' ? { omni: wOmni, custom: wCustom } : undefined,
                trigger_band: { low, high },
              },
              omni: {
                model: 'omni-moderation-latest',
                inputs: inputs && inputs.length ? inputs : ['text'],
                categories: categories && categories.length ? categories : ['violence','sexual','self-harm'],
                confidence_threshold: strictnessToThreshold(omniStrict),
                fallback_on_error: true,
              },
              custom: {
                provider_profile: values.custom_provider_profile || '',
                model: values.custom_model || 'gpt-4o-mini',
                analysis_prompt: values.custom_prompt || 'Return JSON {"flagged": boolean, "confidence": number, "categories": string[]}',
                temperature: Number(values.custom_temperature ?? 0),
                max_tokens: Number(values.custom_max_tokens ?? 256),
                confidence_threshold: strictnessToThreshold(customStrict),
                fallback_on_error: true,
                inputs: inputs && inputs.length ? inputs : ['text'],
              }
            }
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
                              <Select
                                value={(form.getValues() as any)?.rules?.[index]?.semantic_engine || 'omni'}
                                onValueChange={(v) => form.setValue(`rules.${index}.semantic_engine` as any, v)}
                              >
                                <SelectTrigger>
                                  <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectItem value="omni">Built‑in Omni Moderation (free)</SelectItem>
                                  <SelectItem value="custom">Custom (BYOK)</SelectItem>
                                  <SelectItem value="custom+omni">Custom + Omni (add‑on)</SelectItem>
                                </SelectContent>
                              </Select>
                            </div>

                            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                              <MultiSelectChips
                                control={form.control as any}
                                name={`rules.${index}.semantic_inputs`}
                                label={<span>Inputs</span>}
                                description={<span>Select what this rule should analyze.</span>}
                                options={[
                                  { value: 'text', label: 'Text' },
                                  { value: 'image', label: 'Image' },
                                ]}
                              />
                              {/* Strictness slider 1..10 mapped internally to threshold */}
                              <SliderField
                                control={form.control as any}
                                name={`rules.${index}.semantic_strictness`}
                                label={<span>Strictness</span>}
                                description={<span>Higher is stricter (maps to lower confidence threshold).</span>}
                                min={1}
                                max={10}
                                step={1}
                              />
                            </div>
                            {/* Preset pills */}
                            <div className="flex gap-2 text-xs">
                              <Button type="button" variant="secondary" size="sm" onClick={() => applyPreset(index, 'default')}>Default Safety</Button>
                              <Button type="button" variant="secondary" size="sm" onClick={() => applyPreset(index, 'strict_minors')}>Strict Minors</Button>
                              <Button type="button" variant="secondary" size="sm" onClick={() => applyPreset(index, 'violence_strict')}>Violence Strict</Button>
                            </div>

                            <MultiSelectChips
                              control={form.control as any}
                              name={`rules.${index}.semantic_categories`}
                              label={<span>Categories</span>}
                              description={<span>Choose categories to protect. Text‑only categories won’t score for image‑only inputs.</span>}
                              options={[...SEMANTIC_TEXT_IMAGE, ...SEMANTIC_TEXT_ONLY]}
                            />
                            {/* Custom model fields */}
                            {((form.getValues() as any)?.rules?.[index]?.semantic_engine || 'omni') !== 'omni' ? (
                              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                                <TextField control={form.control as any} name={`rules.${index}.custom_provider_profile`} label="Provider Profile ID" placeholder="prof_abc123" />
                                <TextField control={form.control as any} name={`rules.${index}.custom_model`} label="Model" placeholder="gpt-4o-mini" />
                                <TextField control={form.control as any} name={`rules.${index}.custom_prompt`} label="Analysis Output Contract" placeholder='Return JSON {"flagged": boolean, "confidence": number, "categories": string[]}' />
                                <NumberField control={form.control as any} name={`rules.${index}.custom_temperature`} label="Temperature" min={0} max={1} step={0.1} />
                                <NumberField control={form.control as any} name={`rules.${index}.custom_max_tokens`} label="Max Tokens" min={32} max={1024} step={32} />
                                <SliderField control={form.control as any} name={`rules.${index}.custom_strictness`} label={<span>Custom Strictness</span>} description={<span>Applies to the custom engine threshold.</span>} min={1} max={10} step={1} />
                              </div>
                            ) : null}

                            {/* Ensemble combiner fields */}
                            {((form.getValues() as any)?.rules?.[index]?.semantic_engine || 'omni') === 'custom+omni' ? (
                              <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                                <Select
                                  value={(form.getValues() as any)?.rules?.[index]?.combine_mode || 'or'}
                                  onValueChange={(v) => form.setValue(`rules.${index}.combine_mode` as any, v)}
                                >
                                  <SelectTrigger>
                                    <SelectValue placeholder="Combine Mode" />
                                  </SelectTrigger>
                                  <SelectContent>
                                    <SelectItem value="or">OR (flag if Omni or Custom flags)</SelectItem>
                                    <SelectItem value="and">AND (flag only if both flag)</SelectItem>
                                    <SelectItem value="weighted">Weighted</SelectItem>
                                  </SelectContent>
                                </Select>
                                <NumberField control={form.control as any} name={`rules.${index}.combine_low`} label="Trigger Low" min={0.0} max={0.95} step={0.05} />
                                <NumberField control={form.control as any} name={`rules.${index}.combine_high`} label="Trigger High" min={0.05} max={1.0} step={0.05} />
                                {(form.getValues() as any)?.rules?.[index]?.combine_mode === 'weighted' ? (
                                  <>
                                    <NumberField control={form.control as any} name={`rules.${index}.combine_weight_omni`} label="Weight Omni" min={0} max={1} step={0.05} />
                                    <NumberField control={form.control as any} name={`rules.${index}.combine_weight_custom`} label="Weight Custom" min={0} max={1} step={0.05} />
                                  </>
                                ) : null}
                              </div>
                            ) : null}
                            {/* Validation/warning */}
                            <div className="text-xs text-muted-foreground">
                              {Array.isArray((form.getValues() as any)?.rules?.[index]?.semantic_inputs) &&
                               (form.getValues() as any).rules[index].semantic_inputs?.includes('image') ? (
                                <p>Note: text‑only categories (e.g., harassment, hate, illicit, sexual/minors) return 0 when only images are provided.</p>
                              ) : null}
                            </div>
                          </div>
                        ) : null}
                        <p className="text-xs text-muted-foreground mt-1">
                          Omni uses OpenAI moderation (no prompt). Custom uses your model and a JSON output contract. Combined lets you use both.
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
                disabled={isLoading} 
                data-testid="button-create-rulepack"
              >
                {isLoading ? "Creating..." : "Create RulePack"}
              </Button>
            </div>
          </form>
        </Form>
    </Modal>
  );
}
