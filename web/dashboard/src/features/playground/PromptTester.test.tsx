// @vitest-environment jsdom
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { fakeGateway, json } from '../../../test/fakeGateway';
import { SessionProvider } from '../../session/SessionContext';
import { PromptTester } from './PromptTester';

function renderTester(status: number, body: unknown) {
  sessionStorage.setItem('promptshield.adminToken', 'tok');
  const gw = fakeGateway({ 'POST /check': () => json(body, status) });
  render(
    <SessionProvider fetchFn={gw.fetchFn}>
      <PromptTester />
    </SessionProvider>,
  );
  return gw;
}

describe('PromptTester', () => {
  it('sends a sample prompt and shows the decision', async () => {
    const { calls } = renderTester(403, { decision: 'deny', reason: 'pi-ignore', violations: 1, request_id: 'r' });
    const send = screen.getByRole('button', { name: 'Send' });
    expect(send).toBeDisabled();
    await userEvent.click(screen.getByRole('button', { name: 'Injection attempt' }));
    await userEvent.click(send);
    expect(await screen.findByTestId('tester-result')).toHaveTextContent('Blocked HTTP 403 · reason pi-ignore');
    expect(calls[0]?.init?.body).toMatch(/Ignore previous instructions/);
  });

  it('reports gateway errors', async () => {
    renderTester(401, 'unauthorized');
    await userEvent.type(screen.getByLabelText('Prompt sent to POST /check'), 'hello');
    await userEvent.click(screen.getByRole('button', { name: 'Send' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('check failed with HTTP 401');
  });
});
