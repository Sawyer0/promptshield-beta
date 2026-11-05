// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import { wireDecision } from '../test/factories';
import { envelope, fakeGateway, json, sseResponse, type FakeCall } from '../test/fakeGateway';
import { App } from './App';

const TOKEN = 'good-token';
const history = wireDecision('hist-1', 'allow', '2026-11-04T12:00:00Z');
const live = wireDecision('live-1', 'deny', '2026-11-04T12:00:05Z', 'pi-ignore');

function authorized(call: FakeCall): boolean {
  return (call.init?.headers as Record<string, string> | undefined)?.Authorization === `Bearer ${TOKEN}`;
}

function gateway() {
  const deny = () => json({ error: { code: 'UNAUTHORIZED', message: 'invalid admin token' } }, 401);
  return fakeGateway({
    'GET /healthz': () => new Response('ok'),
    'GET /readyz': () => new Response('ready'),
    'GET /v1/version': () => json({ version: '0.2.0' }),
    'GET /v1/decisions/stats': (c) =>
      authorized(c)
        ? envelope({ total: 2, by_decision: { allow: 1, deny: 1, quarantine: 0 }, block_rate: 0.5, latency_p50_ms: 0.4, latency_p95_ms: 0.9, last_minute: 2, retained: 2, capacity: 500, since: 't' })
        : deny(),
    'GET /v1/decisions/': (c) => (authorized(c) ? envelope(live) : deny()),
    'GET /v1/decisions': (c) => (authorized(c) ? envelope({ decisions: [history] }) : deny()),
    'GET /v1/rules': (c) => (authorized(c) ? envelope({ source: 'rules.yaml', loaded_at: 't', generation: 1, rule_count: 0, packs: [] }) : deny()),
    'GET /v1/events': (c) =>
      authorized(c)
        ? sseResponse(`event: ready\ndata: {"status":"ok"}\n\nevent: decision\ndata: ${JSON.stringify({ type: 'decision', data: live })}\n\n`, c.init?.signal)
        : deny(),
  });
}

describe('App', () => {
  it('signs in, then shows history plus live decisions and opens the detail', async () => {
    const user = userEvent.setup();
    const { fetchFn } = gateway();
    render(<App fetchFn={fetchFn} />);

    await user.type(screen.getByLabelText('Admin token'), 'wrong');
    await user.click(screen.getByRole('button', { name: 'Open dashboard' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('The gateway rejected that token.');

    await user.clear(screen.getByLabelText('Admin token'));
    await user.type(screen.getByLabelText('Admin token'), TOKEN);
    await user.click(screen.getByRole('button', { name: 'Open dashboard' }));

    expect(await screen.findByRole('button', { name: 'Inspect decision live-1' })).toBeInTheDocument();
    expect(await screen.findByRole('button', { name: 'Inspect decision hist-1' })).toBeInTheDocument();
    expect(await screen.findByText('Event stream: Live')).toBeInTheDocument();
    expect(within(await screen.findByTestId('stat-total')).getByText('2')).toBeInTheDocument();
    expect(sessionStorage.getItem('promptshield.adminToken')).toBe(TOKEN);

    await user.click(screen.getByRole('button', { name: 'Inspect decision live-1' }));
    const detail = screen.getByRole('complementary', { name: 'Decision detail' });
    expect(within(detail).getByTestId('detail-request-id')).toHaveTextContent('live-1');
    expect(within(detail).getByText('pi-ignore', { selector: '.match__title .mono' })).toBeInTheDocument();
    await user.click(within(detail).getByRole('button', { name: 'Close' }));
    expect(screen.queryByRole('complementary', { name: 'Decision detail' })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Sign out' }));
    expect(screen.getByLabelText('Admin token')).toBeInTheDocument();
    expect(sessionStorage.getItem('promptshield.adminToken')).toBeNull();
  });

  it('returns to sign-in with a notice when a stored token stops working', async () => {
    sessionStorage.setItem('promptshield.adminToken', 'stale');
    const { fetchFn } = gateway();
    render(<App fetchFn={fetchFn} />);
    expect(await screen.findByText(/Your session ended/)).toBeInTheDocument();
    expect(sessionStorage.getItem('promptshield.adminToken')).toBeNull();
  });
});
