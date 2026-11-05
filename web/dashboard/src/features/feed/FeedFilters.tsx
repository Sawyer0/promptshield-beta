import type { DecisionOutcome } from '../../api/contracts';
import type { FeedFilter } from '../../state/decisionMerge';

const CHOICES: { value: DecisionOutcome | 'all'; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'allow', label: 'Allowed' },
  { value: 'quarantine', label: 'Quarantined' },
  { value: 'deny', label: 'Blocked' },
];

interface FeedFiltersProps {
  filter: FeedFilter;
  onChange: (next: FeedFilter) => void;
}

export function FeedFilters({ filter, onChange }: FeedFiltersProps) {
  return (
    <div className="filters">
      <div role="group" aria-label="Filter by decision" className="chips">
        {CHOICES.map((c) => (
          <button
            key={c.value}
            type="button"
            className="chip"
            aria-pressed={filter.outcome === c.value}
            onClick={() => onChange({ ...filter, outcome: c.value })}
          >
            {c.label}
          </button>
        ))}
      </div>
      <label className="filters__search">
        <span className="visually-hidden">Search decisions</span>
        <input
          type="search"
          placeholder="Search request id, rule, reason"
          value={filter.search}
          onChange={(e) => onChange({ ...filter, search: e.target.value })}
        />
      </label>
    </div>
  );
}
