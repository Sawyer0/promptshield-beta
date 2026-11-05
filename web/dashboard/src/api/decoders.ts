// Runtime decoders: turn untrusted JSON (unknown) into typed contracts.
// Every gateway payload passes through here, so a contract drift fails
// loudly in one place instead of rendering undefined values.
import {
  DECISION_OUTCOMES,
  type ActiveRule,
  type CheckResult,
  type DecisionOutcome,
  type DecisionPage,
  type DecisionRecord,
  type DecisionStats,
  type RuleMatch,
  type RulePackSummary,
  type RulesSnapshot,
} from './contracts';

export class DecodeError extends Error {
  constructor(path: string, expected: string) {
    super(`unexpected gateway payload at ${path}: expected ${expected}`);
    this.name = 'DecodeError';
  }
}

type JsonObject = Record<string, unknown>;

function asObject(value: unknown, path: string): JsonObject {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    throw new DecodeError(path, 'object');
  }
  return value as JsonObject;
}

function asArray(value: unknown, path: string): unknown[] {
  if (!Array.isArray(value)) throw new DecodeError(path, 'array');
  return value;
}

function str(obj: JsonObject, key: string, path: string, fallback?: string): string {
  const v = obj[key];
  if (typeof v === 'string') return v;
  if (v === undefined && fallback !== undefined) return fallback;
  throw new DecodeError(`${path}.${key}`, 'string');
}

function num(obj: JsonObject, key: string, path: string, fallback?: number): number {
  const v = obj[key];
  if (typeof v === 'number' && Number.isFinite(v)) return v;
  if (v === undefined && fallback !== undefined) return fallback;
  throw new DecodeError(`${path}.${key}`, 'number');
}

function bool(obj: JsonObject, key: string, path: string): boolean {
  const v = obj[key];
  if (typeof v === 'boolean') return v;
  throw new DecodeError(`${path}.${key}`, 'boolean');
}

export function decodeOutcome(value: unknown, path = 'decision'): DecisionOutcome {
  const match = DECISION_OUTCOMES.find((d) => d === value);
  if (!match) throw new DecodeError(path, DECISION_OUTCOMES.join('|'));
  return match;
}

/** Unwraps the {data, meta} envelope used by admin routes. */
export function unwrapEnvelope(value: unknown): unknown {
  return asObject(value, '$').data;
}

function decodeMatch(value: unknown, path: string): RuleMatch {
  const o = asObject(value, path);
  return {
    ruleId: str(o, 'rule_id', path),
    severity: str(o, 'severity', path, ''),
    category: str(o, 'category', path, ''),
    action: str(o, 'action', path, ''),
    message: str(o, 'message', path, ''),
  };
}

export function decodeDecision(value: unknown, path = 'decision'): DecisionRecord {
  const o = asObject(value, path);
  const matches = o.matches === null || o.matches === undefined ? [] : asArray(o.matches, `${path}.matches`);
  return {
    requestId: str(o, 'request_id', path),
    seq: num(o, 'seq', path),
    time: str(o, 'time', path),
    path: str(o, 'path', path, ''),
    decision: decodeOutcome(o.decision, `${path}.decision`),
    reason: str(o, 'reason', path, ''),
    status: num(o, 'status', path, 0),
    latencyMs: num(o, 'latency_ms', path, 0),
    bytesIn: num(o, 'bytes_in', path, 0),
    violations: num(o, 'violations', path, 0),
    matches: matches.map((m, i) => decodeMatch(m, `${path}.matches[${i}]`)),
  };
}

export function decodeDecisionPage(value: unknown): DecisionPage {
  const o = asObject(value, 'data');
  const next = o.next === null || o.next === undefined ? null : num(o, 'next', 'data');
  return {
    records: asArray(o.decisions, 'data.decisions').map((d, i) => decodeDecision(d, `data.decisions[${i}]`)),
    next,
  };
}

export function decodeStats(value: unknown): DecisionStats {
  const o = asObject(value, 'stats');
  const by = asObject(o.by_decision, 'stats.by_decision');
  return {
    total: num(o, 'total', 'stats'),
    byDecision: {
      allow: num(by, 'allow', 'stats.by_decision', 0),
      quarantine: num(by, 'quarantine', 'stats.by_decision', 0),
      deny: num(by, 'deny', 'stats.by_decision', 0),
    },
    blockRate: num(o, 'block_rate', 'stats'),
    latencyP50Ms: num(o, 'latency_p50_ms', 'stats'),
    latencyP95Ms: num(o, 'latency_p95_ms', 'stats'),
    lastMinute: num(o, 'last_minute', 'stats'),
    retained: num(o, 'retained', 'stats'),
    capacity: num(o, 'capacity', 'stats'),
    since: str(o, 'since', 'stats'),
  };
}

function decodeRule(value: unknown, path: string): ActiveRule {
  const o = asObject(value, path);
  return {
    id: str(o, 'id', path),
    name: str(o, 'name', path, ''),
    level: num(o, 'level', path, 0),
    severity: str(o, 'severity', path, ''),
    category: str(o, 'category', path, ''),
    action: str(o, 'action', path, ''),
    enabled: bool(o, 'enabled', path),
    keywords: num(o, 'keywords', path, 0),
    patterns: num(o, 'patterns', path, 0),
  };
}

function decodePack(value: unknown, path: string): RulePackSummary {
  const o = asObject(value, path);
  return {
    name: str(o, 'name', path),
    version: str(o, 'version', path, ''),
    description: str(o, 'description', path, ''),
    rules: asArray(o.rules, `${path}.rules`).map((r, i) => decodeRule(r, `${path}.rules[${i}]`)),
  };
}

export function decodeRulesSnapshot(value: unknown): RulesSnapshot {
  const o = asObject(value, 'rules');
  return {
    source: str(o, 'source', 'rules', ''),
    loadedAt: str(o, 'loaded_at', 'rules'),
    generation: num(o, 'generation', 'rules'),
    ruleCount: num(o, 'rule_count', 'rules'),
    packs: asArray(o.packs, 'rules.packs').map((p, i) => decodePack(p, `rules.packs[${i}]`)),
  };
}

export function decodeCheckResult(value: unknown, status: number): CheckResult {
  const o = asObject(value, 'check');
  return {
    status,
    decision: decodeOutcome(o.decision, 'check.decision'),
    reason: str(o, 'reason', 'check', ''),
    violations: num(o, 'violations', 'check', 0),
    requestId: str(o, 'request_id', 'check', ''),
  };
}

export function decodeVersion(value: unknown): string {
  return str(asObject(value, 'version'), 'version', 'version', 'unknown');
}

/** Reads the error code from a gateway error body, if present. */
export function errorCodeOf(value: unknown): string | undefined {
  if (typeof value !== 'object' || value === null) return undefined;
  const err = (value as JsonObject).error;
  if (typeof err !== 'object' || err === null) return undefined;
  const code = (err as JsonObject).code;
  return typeof code === 'string' ? code : undefined;
}
