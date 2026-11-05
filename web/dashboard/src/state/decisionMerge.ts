import type { DecisionOutcome, DecisionRecord } from '../api/contracts';

export const FEED_CAPACITY = 200;

/** Newest first: by log sequence when both have one, else by time. */
function newerFirst(a: DecisionRecord, b: DecisionRecord): number {
  if (a.seq > 0 && b.seq > 0) return b.seq - a.seq;
  return Date.parse(b.time) - Date.parse(a.time);
}

/**
 * Merges incoming decisions into the feed: newest first, one entry per
 * request id, capped at capacity. Used for live events, history re-syncs
 * and older pages, so order of arrival does not matter.
 */
export function mergeDecisions(
  incoming: readonly DecisionRecord[],
  existing: readonly DecisionRecord[],
  capacity = FEED_CAPACITY,
): DecisionRecord[] {
  const byId = new Map<string, DecisionRecord>();
  for (const record of [...existing, ...incoming]) {
    byId.set(record.requestId, record);
  }
  return [...byId.values()].sort(newerFirst).slice(0, capacity);
}

/** The feed plus whether the gateway holds decisions older than its tail. */
export interface FeedWindow {
  records: DecisionRecord[];
  hasOlder: boolean;
}

/**
 * Merges newer decisions (live events, re-syncs) without shrinking a window
 * the user widened with "Load older": the cap is the larger of FEED_CAPACITY
 * and the current size. Anything pushed off the tail can be loaded again.
 */
export function mergeNewer(incoming: readonly DecisionRecord[], window: FeedWindow): FeedWindow {
  const capacity = Math.max(FEED_CAPACITY, window.records.length);
  const records = mergeDecisions(incoming, window.records, capacity);
  const kept = new Set(records.map((r) => r.requestId));
  const dropped = [...window.records, ...incoming].some((r) => !kept.has(r.requestId));
  return { records, hasOlder: window.hasOlder || dropped };
}

/** Appends an older page; nothing is dropped. */
export function appendOlder(page: readonly DecisionRecord[], next: number | null, window: FeedWindow): FeedWindow {
  return { records: mergeDecisions(page, window.records, Number.POSITIVE_INFINITY), hasOlder: next !== null };
}

/** The before cursor for the page after the feed's oldest decision. */
export function oldestSeq(records: readonly DecisionRecord[]): number | null {
  const seqs = records.map((r) => r.seq).filter((s) => s > 0);
  return seqs.length > 0 ? Math.min(...seqs) : null;
}

export interface FeedFilter {
  outcome: DecisionOutcome | 'all';
  search: string;
}

/** Applies the outcome chip and free-text search (request id, reason, rule, path). */
export function filterDecisions(records: readonly DecisionRecord[], filter: FeedFilter): DecisionRecord[] {
  const needle = filter.search.trim().toLowerCase();
  return records.filter((r) => {
    if (filter.outcome !== 'all' && r.decision !== filter.outcome) return false;
    if (!needle) return true;
    const haystack = [r.requestId, r.reason, r.path, ...r.matches.map((m) => m.ruleId)].join(' ').toLowerCase();
    return haystack.includes(needle);
  });
}
