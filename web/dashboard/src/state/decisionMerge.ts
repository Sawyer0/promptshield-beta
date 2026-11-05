import type { DecisionOutcome, DecisionRecord } from '../api/contracts';

export const FEED_CAPACITY = 200;

/**
 * Merges incoming decisions into the feed: newest first, one entry per
 * request id, capped at FEED_CAPACITY. Used for both live events and
 * history re-syncs, so order of arrival does not matter.
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
  return [...byId.values()].sort((a, b) => Date.parse(b.time) - Date.parse(a.time)).slice(0, capacity);
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
