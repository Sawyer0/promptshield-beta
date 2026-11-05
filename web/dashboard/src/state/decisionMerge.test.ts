import { describe, expect, it } from 'vitest';
import { makeDecision } from '../../test/factories';
import { filterDecisions, mergeDecisions } from './decisionMerge';

const a = makeDecision({ requestId: 'a', time: '2026-11-04T12:00:01Z' });
const b = makeDecision({ requestId: 'b', time: '2026-11-04T12:00:02Z', decision: 'deny', reason: 'pi-ignore', matches: [{ ruleId: 'pi-ignore', severity: 'CRITICAL', category: '', action: 'block', message: '' }] });
const c = makeDecision({ requestId: 'c', time: '2026-11-04T12:00:03Z', decision: 'quarantine', reason: 'signals_detected' });

describe('mergeDecisions', () => {
  it('orders newest first and de-duplicates by request id', () => {
    expect(mergeDecisions([b, a], [a, c]).map((r) => r.requestId)).toEqual(['c', 'b', 'a']);
  });

  it('caps the feed', () => {
    expect(mergeDecisions([a, b, c], [], 2).map((r) => r.requestId)).toEqual(['c', 'b']);
  });
});

describe('filterDecisions', () => {
  const all = [c, b, a];
  it('filters by outcome', () => {
    expect(filterDecisions(all, { outcome: 'deny', search: '' })).toEqual([b]);
    expect(filterDecisions(all, { outcome: 'all', search: '' })).toHaveLength(3);
  });

  it('searches request id, reason and matched rule, case-insensitively', () => {
    expect(filterDecisions(all, { outcome: 'all', search: 'PI-IGNORE' })).toEqual([b]);
    expect(filterDecisions(all, { outcome: 'all', search: ' SIGNALS_detected ' })).toEqual([c]);
    expect(filterDecisions(all, { outcome: 'allow', search: 'pi-ignore' })).toEqual([]);
  });
});
