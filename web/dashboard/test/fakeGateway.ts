import type { FetchLike } from '../src/api/gatewayClient';

export interface FakeCall {
  url: string;
  init: RequestInit | undefined;
}

type Route = (call: FakeCall) => Response | Promise<Response>;

export function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
}

export function envelope(data: unknown, status = 200): Response {
  return json({ data, meta: {} }, status);
}

/** A stream that emits the given SSE text, then stays open until aborted. */
export function sseResponse(text: string, signal?: AbortSignal | null): Response {
  const body = new ReadableStream<Uint8Array>({
    start(controller) {
      controller.enqueue(new TextEncoder().encode(text));
      signal?.addEventListener('abort', () => controller.close());
    },
  });
  return new Response(body, { status: 200, headers: { 'Content-Type': 'text/event-stream' } });
}

/**
 * Minimal fake gateway: routes are matched by "METHOD /path" prefix. Every
 * call is recorded so tests can assert headers and query strings.
 */
export function fakeGateway(routes: Record<string, Route>): { fetchFn: FetchLike; calls: FakeCall[] } {
  const calls: FakeCall[] = [];
  const fetchFn: FetchLike = async (url, init) => {
    const call = { url, init };
    calls.push(call);
    const method = init?.method ?? 'GET';
    const path = url.replace(/^https?:\/\/[^/]+/, '');
    const key = Object.keys(routes)
      .filter((k) => {
        const [m, p] = k.split(' ');
        return m === method && p !== undefined && path.startsWith(p);
      })
      .sort((a, b) => b.length - a.length)[0];
    const route = key ? routes[key] : undefined;
    if (!route) return json({ error: { code: 'NOT_FOUND', message: `no route for ${method} ${path}` } }, 404);
    return route(call);
  };
  return { fetchFn, calls };
}
