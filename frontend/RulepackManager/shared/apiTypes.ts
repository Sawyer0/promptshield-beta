import { z } from 'zod';

// UI/API primitive types (decoupled from Drizzle)

export interface Rule {
  id: string;
  name: string;
  description?: string;
  level: 1 | 2 | 3;
  action: 'observe' | 'redact' | 'quarantine' | 'deny';
  severity: 'low' | 'medium' | 'high' | 'critical';
  conditions?: RuleCondition[];
  pattern?: string[];
  keywords?: string[];
  semantic_analysis?: boolean;
}

export interface RuleCondition {
  field: string;
  operator: 'eq' | 'ne' | 'contains' | 'matches';
  value: string;
}

export interface RulePackMetadata {
  name: string;
  description?: string;
  version: string;
  authors?: string[];
  tags?: string[];
  created?: string;
  updated?: string;
}

export interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  meta?: {
    total?: number;
    page?: number;
    limit?: number;
  };
}

// Core entities as returned/consumed by UI/API

export interface RulePack {
  id: string;
  tenantId?: string | null;
  name: string;
  description?: string | null;
  currentVersionId?: string | null;
  createdAt?: string;
  updatedAt?: string;
  yamlContent?: string | null;
  rules?: any;
  isActive: boolean;
  status?: string;
  enforcementMode?: 'monitor' | 'enforce' | 'redact' | string;
  failOnSeverity?: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL' | string;
  priority?: number;
  metadata?: any;
}

export interface Tenant {
  id: string;
  name: string;
  enabled?: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface PolicyAssignment {
  id: string;
  tenantId?: string;
  rulepackId: string;
  method?: string; // HTTP method: GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS|*
  targetScope: string;
  priority: number;
  enabled: boolean;
  createdAt?: string;
  updatedAt?: string;
}

// UI form schemas (zod) decoupled from Drizzle

export const insertPolicyAssignmentSchema = z.object({
  rulepackId: z.string().min(1, 'RulePack is required'),
  method: z.enum(['*','GET','POST','PUT','PATCH','DELETE','HEAD','OPTIONS']).default('*'),
  targetScope: z.string().min(1, 'Target scope is required').refine((scope) => {
    const validPatterns = [
      /^\/$/,
      /^\/[a-zA-Z0-9_-]+\/?\*?$/,
      /^\/[a-zA-Z0-9_-]+\/[a-zA-Z0-9_-]+\/?\*?$/,
      /^\/[a-zA-Z0-9_-]+\/[a-zA-Z0-9_-]+\/[a-zA-Z0-9_-]+$/,
    ];
    return validPatterns.some((p) => p.test(scope));
  }, 'Target scope must be a valid route pattern (e.g., /, /v1/*, /api/orders/, /api/payments/refund)'),
  priority: z.number().int().min(0, 'Priority must be 0 or greater').default(100),
  enabled: z.boolean().default(true),
});

export type InsertPolicyAssignment = z.infer<typeof insertPolicyAssignmentSchema>;

export const insertTenantSchema = z.object({
  name: z.string().min(1, 'Tenant name is required'),
  enabled: z.boolean().optional(),
});

export type InsertTenant = z.infer<typeof insertTenantSchema>;

export const insertRulePackSchema = z.object({
  name: z.string().min(1, 'RulePack name is required'),
  description: z.string().optional(),
  status: z.enum(['active', 'draft', 'inactive']).default('active'),
  enforcementMode: z.enum(['monitor', 'enforce', 'redact']).default('monitor'),
  failOnSeverity: z.enum(['LOW', 'MEDIUM', 'HIGH', 'CRITICAL']).default('HIGH'),
  priority: z.number().int().min(0).default(100),
  isActive: z.boolean().default(true),
  metadata: z.any().optional(),
  rules: z.any().optional(),
});

export type InsertRulePack = z.infer<typeof insertRulePackSchema>;

