import { describe, expect, it } from 'vitest';
import { makeDecision } from '../../test/factories';
import { appendOlder, FEED_CAPACITY, filterDecisions, mergeDecisions, mergeNewer, oldestSeq } from './decisionMerge';

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

describe('paging the feed', () => {
  const seqs = (from: number, to: number) =>
    Array.from({ length: from - to + 1 }, (_, i) => makeDecision({ requestId: `r${from - i}`, seq: from - i }));

  it('orders by log sequence when present', () => {
    const older = makeDecision({ requestId: 'x', seq: 1 });
    const newer = makeDecision({ requestId: 'y', seq: 2 });
    expect(mergeDecisions([older], [newer]).map((r) => r.requestId)).toEqual(['y', 'x']);
  });

  it('finds the before cursor from the oldest decision', () => {
    expect(oldestSeq(seqs(30, 21))).toBe(21);
    expect(oldestSeq([])).toBeNull();
  });

  it('appends older pages without dropping rows past FEED_CAPACITY', () => {
    let window = { records: seqs(400, 201), hasOlder: true };
    window = appendOlder(seqs(200, 101), 101, window);
    expect(window.records).toHaveLength(300);
    expect(window.hasOlder).toBe(true);
    window = appendOlder(seqs(100, 1), null, window);
    expect(window.records.map((r) => r.seq)).toEqual(Array.from({ length: 400 }, (_, i) => 400 - i));
    expect(window.hasOlder).toBe(false);
  });

  it('keeps a widened window when live decisions arrive and flags what fell off', () => {
    const wide = { records: seqs(300, 1), hasOlder: false };
    const next = mergeNewer([makeDecision({ requestId: 'live', seq: 301 })], wide);
    expect(next.records).toHaveLength(300);
    expect(next.records[0]?.requestId).toBe('live');
    expect(oldestSeq(next.records)).toBe(2);
    expect(next.hasOlder).toBe(true);
  });

  it('does not flag older rows when nothing falls off', () => {
    const small = { records: seqs(10, 1), hasOlder: false };
    expect(mergeNewer([makeDecision({ requestId: 'live', seq: 11 })], small)).toMatchObject({ hasOlder: false });
    expect(FEED_CAPACITY).toBeGreaterThan(11);
  });
});
