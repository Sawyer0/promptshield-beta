// @vitest-environment jsdom
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { makeRules } from '../../../test/factories';
import type { ActiveRules } from '../../state/useActiveRules';
import { RulesPanel } from './RulesPanel';

function rules(overrides: Partial<ActiveRules> = {}): ActiveRules {
  return { snapshot: makeRules(), loadError: null, reloading: false, reloadError: null, reload: vi.fn(async () => undefined), ...overrides };
}

describe('RulesPanel', () => {
  it('lists packs and rules without exposing match terms', () => {
    render(<RulesPanel rules={rules()} />);
    expect(screen.getByTestId('rules-summary')).toHaveTextContent('1 rules in 1 pack · generation 1');
    expect(screen.getByRole('table', { name: 'Rules in guardrails' })).toBeInTheDocument();
    expect(screen.getByText('pi-ignore')).toBeInTheDocument();
    expect(screen.getByTitle('2 keywords, 0 patterns')).toHaveTextContent('2');
  });

  it('reloads on click and shows failures', async () => {
    const state = rules({ reloadError: 'reload failed; previous rules stay active' });
    render(<RulesPanel rules={state} />);
    await userEvent.click(screen.getByRole('button', { name: 'Reload from disk' }));
    expect(state.reload).toHaveBeenCalledOnce();
    expect(screen.getByRole('alert')).toHaveTextContent('previous rules stay active');
  });

  it('disables the button while reloading and explains an empty set', () => {
    render(<RulesPanel rules={rules({ reloading: true, snapshot: makeRules({ packs: [], ruleCount: 0 }) })} />);
    expect(screen.getByRole('button', { name: 'Reloading…' })).toBeDisabled();
    expect(screen.getByText(/No RulePacks are loaded/)).toBeInTheDocument();
  });
});
