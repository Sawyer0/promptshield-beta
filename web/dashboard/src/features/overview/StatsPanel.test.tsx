// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { makeStats } from '../../../test/factories';
import { StatsPanel } from './StatsPanel';

describe('StatsPanel', () => {
  it('shows totals, block rate, latency and the decision mix', () => {
    render(<StatsPanel stats={makeStats()} error={null} />);
    expect(within(screen.getByTestId('stat-total')).getByText('10')).toBeInTheDocument();
    expect(within(screen.getByTestId('stat-blocked')).getByText('3')).toBeInTheDocument();
    expect(screen.getByText('30% of requests')).toBeInTheDocument();
    expect(within(screen.getByTestId('stat-p95')).getByText('1.3 ms')).toBeInTheDocument();
    expect(screen.getByTestId('mix-deny')).toHaveTextContent('Blocked 2');
    expect(screen.getByRole('figure', { name: 'Decision mix' })).toBeInTheDocument();
  });

  it('reports loading and errors', () => {
    const { rerender } = render(<StatsPanel stats={null} error={null} />);
    expect(screen.getByText('Loading stats…')).toBeInTheDocument();
    rerender(<StatsPanel stats={null} error="boom" />);
    expect(screen.getByRole('alert')).toHaveTextContent('Stats unavailable: boom');
  });
});
