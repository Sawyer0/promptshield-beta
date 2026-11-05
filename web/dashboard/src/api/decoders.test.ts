import { describe, expect, it } from 'vitest';
import { wireDecision } from '../../test/factories';
import { DecodeError, decodeCheckResult, decodeDecision, decodeDecisionList, decodeRulesSnapshot, decodeStats, errorCodeOf } from './decoders';

describe('decoders', () => {
  it('maps a wire decision to camelCase', () => {
    const d = decodeDecision(wireDecision('r1', 'deny', '2026-11-04T12:00:00Z', 'pi-ignore'));
    expect(d).toMatchObject({ requestId: 'r1', decision: 'deny', latencyMs: 0.5, violations: 1 });
    expect(d.matches[0]).toEqual({ ruleId: 'pi-ignore', severity: 'CRITICAL', category: '', action: 'block', message: 'blocked' });
  });

  it('treats null matches as empty and rejects unknown outcomes', () => {
    expect(decodeDecision({ ...wireDecision('r', 'allow', 't'), matches: null }).matches).toEqual([]);
    expect(() => decodeDecision({ ...wireDecision('r', 'allow', 't'), decision: 'maybe' })).toThrow(DecodeError);
    expect(() => decodeDecision({ time: 't', decision: 'allow' })).toThrow(/request_id/);
  });

  it('decodes lists, stats and rules', () => {
    expect(decodeDecisionList({ decisions: [wireDecision('a', 'allow', 't')] })).toHaveLength(1);
    const stats = decodeStats({
      total: 3, by_decision: { allow: 2, deny: 1 }, block_rate: 0.33, latency_p50_ms: 1, latency_p95_ms: 2,
      last_minute: 3, retained: 3, capacity: 500, since: 's',
    });
    expect(stats.byDecision).toEqual({ allow: 2, quarantine: 0, deny: 1 });
    const rules = decodeRulesSnapshot({
      source: 'r.yaml', loaded_at: 't', generation: 2, rule_count: 1,
      packs: [{ name: 'p', version: '1', description: '', rules: [{ id: 'x', name: 'X', level: 1, severity: 'HIGH', category: '', action: 'block', enabled: true, keywords: 1, patterns: 0 }] }],
    });
    expect(rules.packs[0]?.rules[0]?.id).toBe('x');
    expect(() => decodeStats({ total: '3' })).toThrow(DecodeError);
  });

  it('decodes check results and error codes', () => {
    expect(decodeCheckResult({ decision: 'deny', reason: 'r', violations: 1, request_id: 'id' }, 403)).toEqual({
      status: 403, decision: 'deny', reason: 'r', violations: 1, requestId: 'id',
    });
    expect(errorCodeOf({ error: { code: 'NOT_FOUND' } })).toBe('NOT_FOUND');
    expect(errorCodeOf('nope')).toBeUndefined();
  });
});
