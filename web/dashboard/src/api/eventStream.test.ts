import { describe, expect, it, vi } from 'vitest';
import { wireDecision } from '../../test/factories';
import { fakeGateway, json, sseResponse } from '../../test/fakeGateway';
import type { DecisionRecord, StreamStatus } from './contracts';
import { runEventStream } from './eventStream';
import { GatewayClient } from './gatewayClient';

function frame(event: string, data: unknown): string {
  return `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`;
}

describe('runEventStream', () => {
  it('goes live, dispatches decisions and reload events, and reconnects when the stream ends', async () => {
    const controller = new AbortController();
    const decision = wireDecision('r1', 'deny', '2026-11-04T12:00:00Z', 'pi-ignore');
    let connections = 0;
    const { fetchFn, calls } = fakeGateway({
      'GET /v1/events': () => {
        connections += 1;
        if (connections === 1) {
          const text = frame('ready', { status: 'ok' }) + frame('decision', { type: 'decision', data: decision }) + frame('rules_reloaded', { data: {} });
          return new Response(text, { status: 200 });
        }
        return sseResponse(frame('ready', { status: 'ok' }), controller.signal);
      },
    });
    const statuses: StreamStatus[] = [];
    const decisions: DecisionRecord[] = [];
    const onRulesReloaded = vi.fn();
    const sleep = vi.fn(async () => undefined);

    const done = runEventStream(
      { client: new GatewayClient({ token: 'tok', fetchFn }), signal: controller.signal, sleep, initialBackoffMs: 10 },
      { onDecision: (d) => decisions.push(d), onRulesReloaded, onStatus: (s) => statuses.push(s) },
    );
    await vi.waitFor(() => expect(connections).toBe(2));
    await vi.waitFor(() => expect(statuses.filter((s) => s === 'live')).toHaveLength(2));
    controller.abort();
    await done;

    expect(decisions.map((d) => d.requestId)).toEqual(['r1']);
    expect(onRulesReloaded).toHaveBeenCalledOnce();
    expect(statuses).toEqual(['connecting', 'live', 'reconnecting', 'live', 'closed']);
    expect(sleep).toHaveBeenCalledWith(10, controller.signal);
    expect((calls[0]?.init?.headers as Record<string, string>).Authorization).toBe('Bearer tok');
    expect(calls[0]?.url).toBe('/v1/events?types=decision,rules_reloaded');
  });

  it('stops for good when the token is rejected', async () => {
    const { fetchFn, calls } = fakeGateway({ 'GET /v1/events': () => json({ error: { code: 'UNAUTHORIZED' } }, 401) });
    const statuses: StreamStatus[] = [];
    await runEventStream(
      { client: new GatewayClient({ token: 'bad', fetchFn }), signal: new AbortController().signal },
      { onDecision: vi.fn(), onRulesReloaded: vi.fn(), onStatus: (s) => statuses.push(s) },
    );
    expect(statuses).toEqual(['connecting', 'unauthorized']);
    expect(calls).toHaveLength(1);
  });

  it('backs off exponentially on network errors up to the cap', async () => {
    const controller = new AbortController();
    let attempts = 0;
    const fetchFn = async (): Promise<Response> => {
      attempts += 1;
      if (attempts === 4) controller.abort();
      throw new TypeError('network down');
    };
    const delays: number[] = [];
    await runEventStream(
      {
        client: new GatewayClient({ token: 't', fetchFn }),
        signal: controller.signal,
        sleep: async (ms) => void delays.push(ms),
        initialBackoffMs: 100,
        maxBackoffMs: 250,
      },
      { onDecision: vi.fn(), onRulesReloaded: vi.fn(), onStatus: vi.fn() },
    );
    expect(delays).toEqual([100, 200, 250]);
  });
});
