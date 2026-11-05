// @vitest-environment jsdom
import { act, renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { wireDecision } from '../../test/factories';
import { envelope, fakeGateway, json, sseResponse, type FakeCall } from '../../test/fakeGateway';
import { GatewayClient } from '../api/gatewayClient';
import { HISTORY_LIMIT, useDecisionFeed } from './useDecisionFeed';

const TOTAL = 250;

/** The gateway's paging, over seq TOTAL..1, newest first. */
function pagedLog(call: FakeCall): Response {
  const params = new URL(call.url, 'http://gw').searchParams;
  const limit = Number(params.get('limit'));
  const before = params.has('before') ? Number(params.get('before')) : TOTAL + 1;
  const seqs: number[] = [];
  for (let s = before - 1; s >= 1 && seqs.length < limit + 1; s--) seqs.push(s);
  const page = seqs.slice(0, limit);
  const next = seqs.length > limit ? (page[page.length - 1] ?? null) : null;
  return envelope({ decisions: page.map((s) => wireDecision(`req-${s}`, 'allow', `2026-11-04T12:00:00Z`, '', s)), next });
}

function setup(listRoute: (call: FakeCall) => Response = pagedLog) {
  const { fetchFn, calls } = fakeGateway({
    'GET /v1/decisions': listRoute,
    'GET /v1/events': (c) => sseResponse('event: ready\ndata: {"status":"ok"}\n\n', c.init?.signal),
  });
  const client = new GatewayClient({ token: 't', fetchFn });
  const onUnauthorized = vi.fn();
  const hook = renderHook(() => useDecisionFeed(client, onUnauthorized));
  const listCalls = () => calls.filter((c) => c.url.startsWith('/v1/decisions'));
  return { ...hook, listCalls };
}

describe('useDecisionFeed paging', () => {
  it('loads older pages with the before cursor until the gateway says there are no more', async () => {
    const { result, listCalls } = setup();
    await waitFor(() => expect(result.current.records).toHaveLength(HISTORY_LIMIT));
    expect(result.current.older.available).toBe(true);

    act(() => result.current.older.load());
    await waitFor(() => expect(result.current.records).toHaveLength(200));
    expect(listCalls().some((c) => c.url === '/v1/decisions?limit=100&before=151')).toBe(true);

    act(() => result.current.older.load());
    await waitFor(() => expect(result.current.records).toHaveLength(TOTAL));
    expect(listCalls().some((c) => c.url === '/v1/decisions?limit=100&before=51')).toBe(true);

    const seqs = result.current.records.map((r) => r.seq);
    expect(seqs).toEqual(Array.from({ length: TOTAL }, (_, i) => TOTAL - i));
    expect(result.current.older.available).toBe(false);
    expect(result.current.older.loading).toBe(false);
    // Exactly two older-page requests; the rest are the newest-page (re)syncs.
    expect(listCalls().filter((c) => c.url.includes('before=')).map((c) => c.url)).toEqual([
      '/v1/decisions?limit=100&before=151',
      '/v1/decisions?limit=100&before=51',
    ]);
  });

  it('hides Load older when the first page is the whole log', async () => {
    const { result } = setup(() => envelope({ decisions: [wireDecision('only', 'allow', 't', '', 1)], next: null }));
    await waitFor(() => expect(result.current.records).toHaveLength(1));
    expect(result.current.older.available).toBe(false);
  });

  it('surfaces a failed older page without losing the feed', async () => {
    const { result } = setup((call) =>
      call.url.includes('before=') ? json({ error: { code: 'INVALID_ARGUMENT', message: 'bad cursor' } }, 400) : pagedLog(call),
    );
    await waitFor(() => expect(result.current.records).toHaveLength(HISTORY_LIMIT));
    act(() => result.current.older.load());
    await waitFor(() => expect(result.current.older.error).toBe('bad cursor'));
    expect(result.current.records).toHaveLength(HISTORY_LIMIT);
    expect(result.current.older.available).toBe(true);
  });
});
