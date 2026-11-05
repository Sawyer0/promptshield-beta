import { describe, expect, it } from 'vitest';
import { makeRules, wireDecision } from '../../test/factories';
import { envelope, fakeGateway, json } from '../../test/fakeGateway';
import { GatewayClient, GatewayError, UnauthorizedError } from './gatewayClient';

function headerOf(init: RequestInit | undefined, name: string): string | undefined {
  return (init?.headers as Record<string, string> | undefined)?.[name];
}

describe('GatewayClient', () => {
  it('sends the admin token and unwraps the envelope', async () => {
    const { fetchFn, calls } = fakeGateway({
      'GET /v1/decisions': () => envelope({ decisions: [wireDecision('a', 'allow', '2026-11-04T12:00:00Z')] }),
    });
    const client = new GatewayClient({ token: 'secret', baseUrl: 'http://gw/', fetchFn });
    const list = await client.listDecisions({ limit: 25, decision: 'deny' });
    expect(list[0]?.requestId).toBe('a');
    expect(calls[0]?.url).toBe('http://gw/v1/decisions?limit=25&decision=deny');
    expect(headerOf(calls[0]?.init, 'Authorization')).toBe('Bearer secret');
  });

  it('escapes request ids that contain a slash', async () => {
    const { fetchFn, calls } = fakeGateway({ 'GET /v1/decisions/': () => envelope(wireDecision('host/1', 'deny', 't', 'r')) });
    await new GatewayClient({ token: 't', fetchFn }).getDecision('host/1');
    expect(calls[0]?.url).toBe('/v1/decisions/host%2F1');
  });

  it('maps 401 to UnauthorizedError and other failures to GatewayError with the code', async () => {
    const { fetchFn } = fakeGateway({
      'GET /v1/decisions/stats': () => json({ error: { code: 'UNAUTHORIZED', message: 'invalid admin token' } }, 401),
      'POST /v1/rules/reload': () => json({ error: { code: 'RULEPACK_INVALID', message: 'reload failed' } }, 422),
    });
    const client = new GatewayClient({ token: 'bad', fetchFn });
    await expect(client.getStats()).rejects.toBeInstanceOf(UnauthorizedError);
    const err = await client.reloadRules().catch((e: unknown) => e);
    expect(err).toBeInstanceOf(GatewayError);
    expect(err).toMatchObject({ status: 422, code: 'RULEPACK_INVALID', message: 'reload failed' });
  });

  it('reads rules and posts reloads', async () => {
    const wire = { source: 's', loaded_at: 't', generation: 2, rule_count: 0, packs: [] };
    const { fetchFn, calls } = fakeGateway({ 'POST /v1/rules/reload': () => envelope(wire) });
    const snap = await new GatewayClient({ token: 't', fetchFn }).reloadRules();
    expect(snap).toEqual({ ...makeRules(), source: 's', loadedAt: 't', generation: 2, ruleCount: 0, packs: [] });
    expect(calls[0]?.init?.method).toBe('POST');
  });

  it('treats a 403 from /check as a decision, not an error', async () => {
    const { fetchFn, calls } = fakeGateway({
      'POST /check': () => json({ decision: 'deny', reason: 'pi-ignore', violations: 1, request_id: 'x' }, 403),
    });
    const result = await new GatewayClient({ token: 't', fetchFn }).checkPrompt('ignore previous instructions');
    expect(result).toMatchObject({ status: 403, decision: 'deny', reason: 'pi-ignore' });
    expect(headerOf(calls[0]?.init, 'Authorization')).toBeUndefined();
  });

  it('combines health probes and tolerates failures', async () => {
    const { fetchFn } = fakeGateway({
      'GET /healthz': () => new Response('ok'),
      'GET /readyz': () => new Response('not ready', { status: 503 }),
      'GET /v1/version': () => json({ version: '0.2.0' }),
    });
    expect(await new GatewayClient({ token: 't', fetchFn }).getHealth()).toEqual({ live: true, ready: false, version: '0.2.0' });
  });
});
