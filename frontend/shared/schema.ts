// Shared frontend schema types mirroring the PromptShield database schema

export interface Policy {
  id: string;
  name: string;
  type: "security" | "compliance" | "custom" | string;
  content: string;
  version: number;
  created_by: string | null;
  is_active: boolean;
  created_at: Date;
  updated_at: Date;
}

export type InsertPolicy = Omit<Policy, "id" | "created_at" | "updated_at">;

export interface Violation {
  id: string;
  request_id: string;
  policy_id: string | null;
  content: string | null;
  decision: string;
  reason: string | null;
  severity: string | null;
  rule_matched: string | null;
  processing_time_ms: number | null;
  metadata: unknown | null;
  timestamp: Date;
}

export type InsertViolation = Omit<Violation, "id" | "timestamp">;

export interface SystemMetric {
  id: string;
  metric_name: string;
  metric_value: number | null;
  metric_type: string | null;
  timestamp: Date;
}

export type InsertSystemMetric = Omit<SystemMetric, "id" | "timestamp">;

export interface RulePack {
  id: string;
  name: string;
  description: string | null;
  version: string | null;
  author: string | null;
  tags: string[] | null;
  category: string | null;
  rules_count: number | null;
  status: string;
  content: unknown | null;
  created_at: Date;
  updated_at: Date;
}

export type InsertRulePack = Omit<RulePack, "id" | "created_at" | "updated_at">;

export interface User {
  id: string;
  email: string | null;
  firstName: string | null;
  lastName: string | null;
  profileImageUrl: string | null;
  createdAt: Date;
  updatedAt: Date;
}

export interface UpsertUser {
  id?: string;
  email?: string | null;
  firstName?: string | null;
  lastName?: string | null;
  profileImageUrl?: string | null;
}

export interface DashboardViolationPoint {
  timestamp: string;
  count: number;
}

export interface DashboardPolicyEffectiveness {
  decision: string;
  count: number;
}

export interface DashboardMetrics {
  total_violations: number;
  active_policies: number;
  requests_today: number;
  avg_response_time: number;
  violation_trend: DashboardViolationPoint[];
  policy_effectiveness: DashboardPolicyEffectiveness[];
}

export interface SystemHealth {
  api_gateway: string;
  policy_engine: string;
  ml_models: string;
  database: string;
  cpu_usage: number;
  memory_usage: number;
}

// Minimal schema objects with a Zod-like interface used in routes.ts
// These perform identity validation but keep the API surface compatible.

type Schema<T> = {
  parse: (input: unknown) => T;
  partial: () => Schema<Partial<T>>;
};

const insertPolicyPartialSchema: Schema<Partial<InsertPolicy>> = {
  parse(input: unknown): Partial<InsertPolicy> {
    if (!input || typeof input !== "object") {
      throw new Error("Invalid policy payload");
    }
    const src = input as any;
    const out: Partial<InsertPolicy> = {};
    if ("name" in src) {
      if (typeof src.name !== "string") throw new Error("name must be a string");
      out.name = src.name;
    }
    if ("type" in src) {
      if (typeof src.type !== "string") throw new Error("type must be a string");
      out.type = src.type;
    }
    if ("content" in src) {
      if (typeof src.content !== "string") throw new Error("content must be a string");
      out.content = src.content;
    }
    if ("version" in src) {
      if (typeof src.version !== "number") throw new Error("version must be a number");
      out.version = src.version;
    }
    if ("created_by" in src) {
      if (src.created_by !== null && typeof src.created_by !== "string") {
        throw new Error("created_by must be string or null");
      }
      out.created_by = src.created_by;
    }
    if ("is_active" in src) {
      if (typeof src.is_active !== "boolean") {
        throw new Error("is_active must be a boolean");
      }
      out.is_active = src.is_active;
    }
    return out;
  },
  partial(): Schema<Partial<InsertPolicy>> {
    return insertPolicyPartialSchema;
  },
};

export const insertPolicySchema: Schema<InsertPolicy> = {
  parse(input: unknown): InsertPolicy {
    if (!input || typeof input !== "object") {
      throw new Error("Invalid policy payload");
    }
    const src = input as any;
    if (typeof src.name !== "string" || !src.name.trim()) {
      throw new Error("name is required");
    }
    if (typeof src.type !== "string" || !src.type.trim()) {
      throw new Error("type is required");
    }
    if (typeof src.content !== "string" || !src.content.trim()) {
      throw new Error("content is required");
    }
    const version = src.version === undefined ? 1 : src.version;
    if (typeof version !== "number") {
      throw new Error("version must be a number");
    }
    let created_by: string | null = null;
    if (src.created_by !== undefined) {
      if (src.created_by !== null && typeof src.created_by !== "string") {
        throw new Error("created_by must be string or null");
      }
      created_by = src.created_by;
    }
    let is_active = false;
    if (src.is_active !== undefined) {
      if (typeof src.is_active !== "boolean") {
        throw new Error("is_active must be a boolean");
      }
      is_active = src.is_active;
    }
    return {
      name: src.name,
      type: src.type,
      content: src.content,
      version,
      created_by,
      is_active,
    };
  },
  partial(): Schema<Partial<InsertPolicy>> {
    return insertPolicyPartialSchema;
  },
};

const insertViolationPartialSchema: Schema<Partial<InsertViolation>> = {
  parse(input: unknown): Partial<InsertViolation> {
    if (!input || typeof input !== "object") {
      throw new Error("Invalid violation payload");
    }
    const src = input as any;
    const out: Partial<InsertViolation> = {};
    if ("request_id" in src) {
      if (typeof src.request_id !== "string") {
        throw new Error("request_id must be a string");
      }
      out.request_id = src.request_id;
    }
    if ("policy_id" in src) {
      if (src.policy_id !== null && typeof src.policy_id !== "string") {
        throw new Error("policy_id must be string or null");
      }
      out.policy_id = src.policy_id;
    }
    if ("content" in src) {
      if (src.content !== null && typeof src.content !== "string") {
        throw new Error("content must be string or null");
      }
      out.content = src.content;
    }
    if ("decision" in src) {
      if (typeof src.decision !== "string") {
        throw new Error("decision must be a string");
      }
      out.decision = src.decision;
    }
    if ("reason" in src) {
      if (src.reason !== null && typeof src.reason !== "string") {
        throw new Error("reason must be string or null");
      }
      out.reason = src.reason;
    }
    if ("severity" in src) {
      if (src.severity !== null && typeof src.severity !== "string") {
        throw new Error("severity must be string or null");
      }
      out.severity = src.severity;
    }
    if ("rule_matched" in src) {
      if (src.rule_matched !== null && typeof src.rule_matched !== "string") {
        throw new Error("rule_matched must be string or null");
      }
      out.rule_matched = src.rule_matched;
    }
    if ("processing_time_ms" in src) {
      if (src.processing_time_ms !== null && typeof src.processing_time_ms !== "number") {
        throw new Error("processing_time_ms must be number or null");
      }
      out.processing_time_ms = src.processing_time_ms;
    }
    if ("metadata" in src) {
      out.metadata = src.metadata;
    }
    return out;
  },
  partial(): Schema<Partial<InsertViolation>> {
    return insertViolationPartialSchema;
  },
};

export const insertViolationSchema: Schema<InsertViolation> = {
  parse(input: unknown): InsertViolation {
    if (!input || typeof input !== "object") {
      throw new Error("Invalid violation payload");
    }
    const src = input as any;
    if (typeof src.request_id !== "string" || !src.request_id.trim()) {
      throw new Error("request_id is required");
    }
    if (typeof src.decision !== "string" || !src.decision.trim()) {
      throw new Error("decision is required");
    }
    let policy_id: string | null = null;
    if (src.policy_id !== undefined) {
      if (src.policy_id !== null && typeof src.policy_id !== "string") {
        throw new Error("policy_id must be string or null");
      }
      policy_id = src.policy_id;
    }
    let content: string | null = null;
    if (src.content !== undefined) {
      if (src.content !== null && typeof src.content !== "string") {
        throw new Error("content must be string or null");
      }
      content = src.content;
    }
    let reason: string | null = null;
    if (src.reason !== undefined) {
      if (src.reason !== null && typeof src.reason !== "string") {
        throw new Error("reason must be string or null");
      }
      reason = src.reason;
    }
    let severity: string | null = null;
    if (src.severity !== undefined) {
      if (src.severity !== null && typeof src.severity !== "string") {
        throw new Error("severity must be string or null");
      }
      severity = src.severity;
    }
    let rule_matched: string | null = null;
    if (src.rule_matched !== undefined) {
      if (src.rule_matched !== null && typeof src.rule_matched !== "string") {
        throw new Error("rule_matched must be string or null");
      }
      rule_matched = src.rule_matched;
    }
    let processing_time_ms: number | null = null;
    if (src.processing_time_ms !== undefined) {
      if (src.processing_time_ms !== null && typeof src.processing_time_ms !== "number") {
        throw new Error("processing_time_ms must be number or null");
      }
      processing_time_ms = src.processing_time_ms;
    }
    const metadata = src.metadata ?? null;
    return {
      request_id: src.request_id,
      policy_id,
      content,
      decision: src.decision,
      reason,
      severity,
      rule_matched,
      processing_time_ms,
      metadata,
    };
  },
  partial(): Schema<Partial<InsertViolation>> {
    return insertViolationPartialSchema;
  },
};

const insertRulePackPartialSchema: Schema<Partial<InsertRulePack>> = {
  parse(input: unknown): Partial<InsertRulePack> {
    if (!input || typeof input !== "object") {
      throw new Error("Invalid rulepack payload");
    }
    const src = input as any;
    const out: Partial<InsertRulePack> = {};
    if ("name" in src) {
      if (typeof src.name !== "string") throw new Error("name must be a string");
      out.name = src.name;
    }
    if ("description" in src) {
      if (src.description !== null && typeof src.description !== "string") {
        throw new Error("description must be string or null");
      }
      out.description = src.description;
    }
    if ("version" in src) {
      if (src.version !== null && typeof src.version !== "string") {
        throw new Error("version must be string or null");
      }
      out.version = src.version;
    }
    if ("author" in src) {
      if (src.author !== null && typeof src.author !== "string") {
        throw new Error("author must be string or null");
      }
      out.author = src.author;
    }
    if ("tags" in src) {
      if (src.tags !== null && !Array.isArray(src.tags)) {
        throw new Error("tags must be array of strings or null");
      }
      out.tags = src.tags;
    }
    if ("category" in src) {
      if (src.category !== null && typeof src.category !== "string") {
        throw new Error("category must be string or null");
      }
      out.category = src.category;
    }
    if ("rules_count" in src) {
      if (src.rules_count !== null && typeof src.rules_count !== "number") {
        throw new Error("rules_count must be number or null");
      }
      out.rules_count = src.rules_count;
    }
    if ("status" in src) {
      if (typeof src.status !== "string") {
        throw new Error("status must be a string");
      }
      out.status = src.status;
    }
    if ("content" in src) {
      out.content = src.content;
    }
    return out;
  },
  partial(): Schema<Partial<InsertRulePack>> {
    return insertRulePackPartialSchema;
  },
};

export const insertRulePackSchema: Schema<InsertRulePack> = {
  parse(input: unknown): InsertRulePack {
    if (!input || typeof input !== "object") {
      throw new Error("Invalid rulepack payload");
    }
    const src = input as any;
    if (typeof src.name !== "string" || !src.name.trim()) {
      throw new Error("name is required");
    }
    let description: string | null = null;
    if (src.description !== undefined) {
      if (src.description !== null && typeof src.description !== "string") {
        throw new Error("description must be string or null");
      }
      description = src.description;
    }
    let version: string | null = null;
    if (src.version !== undefined) {
      if (src.version !== null && typeof src.version !== "string") {
        throw new Error("version must be string or null");
      }
      version = src.version;
    }
    let author: string | null = null;
    if (src.author !== undefined) {
      if (src.author !== null && typeof src.author !== "string") {
        throw new Error("author must be string or null");
      }
      author = src.author;
    }
    let tags: string[] | null = null;
    if (src.tags !== undefined) {
      if (src.tags !== null && !Array.isArray(src.tags)) {
        throw new Error("tags must be array of strings or null");
      }
      tags = src.tags;
    }
    let category: string | null = null;
    if (src.category !== undefined) {
      if (src.category !== null && typeof src.category !== "string") {
        throw new Error("category must be string or null");
      }
      category = src.category;
    }
    let rules_count: number | null = null;
    if (src.rules_count !== undefined) {
      if (src.rules_count !== null && typeof src.rules_count !== "number") {
        throw new Error("rules_count must be number or null");
      }
      rules_count = src.rules_count;
    }
    const status: string = src.status ?? "draft";
    if (typeof status !== "string") {
      throw new Error("status must be a string");
    }
    const content = src.content ?? null;
    return {
      name: src.name,
      description,
      version,
      author,
      tags,
      category,
      rules_count,
      status,
      content,
    };
  },
  partial(): Schema<Partial<InsertRulePack>> {
    return insertRulePackPartialSchema;
  },
};
