import type { DecisionOutcome } from '../api/contracts';
import { OUTCOME_LABEL } from './format';

/** Decision label that never relies on colour alone: the text says the outcome. */
export function OutcomeBadge({ outcome }: { outcome: DecisionOutcome }) {
  return <span className={`badge badge--${outcome}`}>{OUTCOME_LABEL[outcome]}</span>;
}
