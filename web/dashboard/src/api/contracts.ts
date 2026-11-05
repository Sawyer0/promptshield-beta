// Typed contracts for the PromptShield gateway HTTP API, as used by the
// dashboard. Field names are camelCase; decoders.ts maps them from the
// gateway's snake_case JSON and rejects payloads that do not match.

export const DECISION_OUTCOMES = ['allow', 'quarantine', 'deny'] as const;
export type DecisionOutcome = (typeof DECISION_OUTCOMES)[number];

export interface RuleMatch {
  ruleId: string;
  severity: string;
  category: string;
  action: string;
  message: string;
}

/** GET /v1/decisions, /v1/decisions/{id}, and "decision" SSE events. */
export interface DecisionRecord {
  requestId: string;
  time: string;
  path: string;
  decision: DecisionOutcome;
  reason: string;
  status: number;
  latencyMs: number;
  bytesIn: number;
  violations: number;
  matches: RuleMatch[];
}

/** GET /v1/decisions/stats */
export interface DecisionStats {
  total: number;
  byDecision: Record<DecisionOutcome, number>;
  blockRate: number;
  latencyP50Ms: number;
  latencyP95Ms: number;
  lastMinute: number;
  retained: number;
  capacity: number;
  since: string;
}

export interface ActiveRule {
  id: string;
  name: string;
  level: number;
  severity: string;
  category: string;
  action: string;
  enabled: boolean;
  keywords: number;
  patterns: number;
}

export interface RulePackSummary {
  name: string;
  version: string;
  description: string;
  rules: ActiveRule[];
}

/** GET /v1/rules and POST /v1/rules/reload */
export interface RulesSnapshot {
  source: string;
  loadedAt: string;
  generation: number;
  ruleCount: number;
  packs: RulePackSummary[];
}

/** /healthz, /readyz and /v1/version combined. */
export interface GatewayHealth {
  live: boolean;
  ready: boolean;
  version: string;
}

/** POST /check response (200 allowed, 403 blocked in enforce mode). */
export interface CheckResult {
  status: number;
  decision: DecisionOutcome;
  reason: string;
  violations: number;
  requestId: string;
}

export interface DecisionQuery {
  limit?: number;
  decision?: DecisionOutcome;
}

export type StreamStatus = 'connecting' | 'live' | 'reconnecting' | 'unauthorized' | 'closed';
