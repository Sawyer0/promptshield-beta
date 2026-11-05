import type { DecisionOutcome } from '../api/contracts';

const countFormat = new Intl.NumberFormat('en-US');
const percentFormat = new Intl.NumberFormat('en-US', { style: 'percent', maximumFractionDigits: 1 });
const timeFormat = new Intl.DateTimeFormat('en-US', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false });
const dateTimeFormat = new Intl.DateTimeFormat('en-US', { dateStyle: 'medium', timeStyle: 'medium' });

export function formatCount(n: number): string {
  return countFormat.format(n);
}

export function formatPercent(ratio: number): string {
  return percentFormat.format(ratio);
}

export function formatLatency(ms: number): string {
  if (ms <= 0) return '—';
  if (ms < 1) return `${(ms * 1000).toFixed(0)} µs`;
  if (ms < 100) return `${ms.toFixed(1)} ms`;
  return `${Math.round(ms)} ms`;
}

export function formatClock(iso: string): string {
  const t = Date.parse(iso);
  return Number.isNaN(t) ? '—' : timeFormat.format(t);
}

export function formatDateTime(iso: string): string {
  const t = Date.parse(iso);
  return Number.isNaN(t) || t <= 0 ? '—' : dateTimeFormat.format(t);
}

export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

export const OUTCOME_LABEL: Record<DecisionOutcome, string> = {
  allow: 'Allowed',
  quarantine: 'Quarantined',
  deny: 'Blocked',
};
