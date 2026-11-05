// @vitest-environment jsdom
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { makeDecision } from '../../../test/factories';
import { DecisionFeed } from './DecisionFeed';

const records = [
  makeDecision({ requestId: 'deny-1', decision: 'deny', reason: 'pi-ignore', status: 403, matches: [{ ruleId: 'pi-ignore', severity: 'CRITICAL', category: '', action: 'block', message: '' }] }),
  makeDecision({ requestId: 'allow-1' }),
];

describe('DecisionFeed', () => {
  it('renders rows with outcome text and rule', () => {
    render(<DecisionFeed records={records} status="live" error={null} selectedId={null} onSelect={vi.fn()} />);
    expect(screen.getAllByTestId('decision-row')).toHaveLength(2);
    expect(screen.getByText('Blocked', { selector: '.badge' })).toBeInTheDocument();
    expect(screen.getByText('pi-ignore')).toBeInTheDocument();
    expect(screen.getByTestId('feed-count')).toHaveTextContent('Showing 2 of 2');
  });

  it('filters by outcome chip and search', async () => {
    const user = userEvent.setup();
    render(<DecisionFeed records={records} status="live" error={null} selectedId={null} onSelect={vi.fn()} />);
    const blocked = screen.getByRole('button', { name: 'Blocked' });
    await user.click(blocked);
    expect(blocked).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getAllByTestId('decision-row')).toHaveLength(1);
    await user.click(screen.getByRole('button', { name: 'All' }));
    await user.type(screen.getByRole('searchbox', { name: 'Search decisions' }), 'allow-1');
    expect(screen.getByTestId('feed-count')).toHaveTextContent('Showing 1 of 2');
    await user.clear(screen.getByRole('searchbox'));
    await user.type(screen.getByRole('searchbox'), 'zzz');
    expect(screen.getByText('No decisions match the current filter.')).toBeInTheDocument();
  });

  it('selects a decision and marks the current row', async () => {
    const onSelect = vi.fn();
    render(<DecisionFeed records={records} status="live" error={null} selectedId="allow-1" onSelect={onSelect} />);
    await userEvent.click(screen.getByRole('button', { name: 'Inspect decision deny-1' }));
    expect(onSelect).toHaveBeenCalledWith(records[0]);
    expect(screen.getByRole('button', { name: 'Inspect decision allow-1' })).toHaveAttribute('aria-current', 'true');
  });

  it('explains an empty feed', () => {
    const { rerender } = render(<DecisionFeed records={[]} status="connecting" error={null} selectedId={null} onSelect={vi.fn()} />);
    expect(screen.getByText('Connecting to the event stream…')).toBeInTheDocument();
    rerender(<DecisionFeed records={[]} status="live" error={null} selectedId={null} onSelect={vi.fn()} />);
    expect(screen.getByText(/Waiting for traffic/)).toBeInTheDocument();
  });
});
