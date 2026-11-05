import type { DecisionRecord, DecisionStats, RulesSnapshot } from '../src/api/contracts';

export function makeDecision(overrides: Partial<DecisionRecord> = {}): DecisionRecord {
  return {
    seq: 0,
    requestId: 'req-1',
    time: '2026-11-04T12:00:00.000Z',
    path: '/check',
    decision: 'allow',
    reason: 'no_signals',
    status: 200,
    latencyMs: 0.42,
    bytesIn: 64,
    violations: 0,
    matches: [],
    ...overrides,
  };
}

export function makeStats(overrides: Partial<DecisionStats> = {}): DecisionStats {
  return {
    total: 10,
    byDecision: { allow: 7, quarantine: 1, deny: 2 },
    blockRate: 0.3,
    latencyP50Ms: 0.3,
    latencyP95Ms: 1.25,
    lastMinute: 4,
    retained: 10,
    capacity: 500,
    since: '2026-11-04T11:00:00Z',
    ...overrides,
  };
}

export function makeRules(overrides: Partial<RulesSnapshot> = {}): RulesSnapshot {
  return {
    source: '/etc/promptshield/rules.yaml',
    loadedAt: '2026-11-04T11:00:00Z',
    generation: 1,
    ruleCount: 1,
    packs: [
      {
        name: 'guardrails',
        version: '1.0.0',
        description: 'Test pack',
        rules: [
          { id: 'pi-ignore', name: 'Ignore instructions', level: 1, severity: 'CRITICAL', category: 'injection', action: 'block', enabled: true, keywords: 2, patterns: 0 },
        ],
      },
    ],
    ...overrides,
  };
}

/** Wire-format (snake_case) decision, as the gateway sends it. */
export function wireDecision(id: string, decision: 'allow' | 'deny' | 'quarantine', time: string, rule = '', seq = 1) {
  return {
    seq,
    request_id: id,
    time,
    path: '/check',
    decision,
    reason: rule || 'no_signals',
    status: decision === 'allow' ? 200 : 403,
    latency_ms: 0.5,
    bytes_in: 40,
    violations: rule ? 1 : 0,
    matches: rule ? [{ rule_id: rule, severity: 'CRITICAL', action: 'block', message: 'blocked' }] : [],
  };
}
