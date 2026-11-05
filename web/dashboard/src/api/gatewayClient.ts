// HTTP adapter for the gateway. Components never call fetch directly; they
// go through this client, which adds the admin token, unwraps envelopes,
// maps failures to typed errors, and decodes every payload.
import type {
  CheckResult,
  DecisionPage,
  DecisionQuery,
  DecisionRecord,
  DecisionStats,
  GatewayHealth,
  RulesSnapshot,
} from './contracts';
import {
  decodeCheckResult,
  decodeDecision,
  decodeDecisionPage,
  decodeRulesSnapshot,
  decodeStats,
  decodeVersion,
  errorCodeOf,
  unwrapEnvelope,
} from './decoders';

export type FetchLike = (input: string, init?: RequestInit) => Promise<Response>;

export class GatewayError extends Error {
  readonly status: number;
  readonly code: string | undefined;

  constructor(status: number, message: string, code?: string) {
    super(message);
    this.name = 'GatewayError';
    this.status = status;
    this.code = code;
  }
}

/** The admin token is missing, wrong, or the gateway has none configured. */
export class UnauthorizedError extends GatewayError {
  constructor() {
    super(401, 'The gateway rejected the admin token.', 'UNAUTHORIZED');
    this.name = 'UnauthorizedError';
  }
}

export interface GatewayClientOptions {
  token: string;
  baseUrl?: string;
  fetchFn?: FetchLike;
}

export class GatewayClient {
  readonly baseUrl: string;
  readonly token: string;
  private readonly fetchFn: FetchLike;

  constructor({ token, baseUrl = '', fetchFn }: GatewayClientOptions) {
    this.token = token;
    this.baseUrl = baseUrl.replace(/\/$/, '');
    this.fetchFn = fetchFn ?? ((input, init) => fetch(input, init));
  }

  authHeaders(): Record<string, string> {
    return { Authorization: `Bearer ${this.token}` };
  }

  url(path: string): string {
    return `${this.baseUrl}${path}`;
  }

  async listDecisions(query: DecisionQuery = {}): Promise<DecisionPage> {
    const params = new URLSearchParams();
    if (query.limit !== undefined) params.set('limit', String(query.limit));
    if (query.decision) params.set('decision', query.decision);
    if (query.before !== undefined) params.set('before', String(query.before));
    const qs = params.toString();
    return decodeDecisionPage(await this.admin(`/v1/decisions${qs ? `?${qs}` : ''}`));
  }

  async getDecision(requestId: string): Promise<DecisionRecord> {
    return decodeDecision(await this.admin(`/v1/decisions/${encodeURIComponent(requestId)}`));
  }

  async getStats(): Promise<DecisionStats> {
    return decodeStats(await this.admin('/v1/decisions/stats'));
  }

  async getRules(): Promise<RulesSnapshot> {
    return decodeRulesSnapshot(await this.admin('/v1/rules'));
  }

  async reloadRules(): Promise<RulesSnapshot> {
    return decodeRulesSnapshot(await this.admin('/v1/rules/reload', { method: 'POST' }));
  }

  /** Opens the raw SSE response for GET /v1/events; eventStream.ts reads it. */
  openEventStream(types: string, signal: AbortSignal): Promise<Response> {
    return this.fetchFn(this.url(`/v1/events?types=${types}`), {
      headers: { Accept: 'text/event-stream', ...this.authHeaders() },
      signal,
    });
  }

  /** Verifies the token against an admin route without side effects. */
  async verifyToken(): Promise<void> {
    await this.admin('/v1/decisions/stats');
  }

  async getHealth(): Promise<GatewayHealth> {
    const [live, ready, version] = await Promise.all([
      this.probe('/healthz'),
      this.probe('/readyz'),
      this.fetchFn(this.url('/v1/version'))
        .then(async (r) => (r.ok ? decodeVersion(await r.json()) : 'unknown'))
        .catch(() => 'unknown'),
    ]);
    return { live, ready, version };
  }

  /** Sends a prompt through POST /check, as an application would. */
  async checkPrompt(prompt: string): Promise<CheckResult> {
    const res = await this.fetchFn(this.url('/check'), {
      method: 'POST',
      headers: { 'Content-Type': 'text/plain' },
      body: prompt,
    });
    if (res.status !== 200 && res.status !== 403) {
      throw new GatewayError(res.status, `check failed with HTTP ${res.status}`);
    }
    return decodeCheckResult(await res.json(), res.status);
  }

  private async probe(path: string): Promise<boolean> {
    try {
      return (await this.fetchFn(this.url(path))).ok;
    } catch {
      return false;
    }
  }

  private async admin(path: string, init: RequestInit = {}): Promise<unknown> {
    const res = await this.fetchFn(this.url(path), {
      ...init,
      headers: { Accept: 'application/json', ...this.authHeaders() },
    });
    if (res.status === 401) throw new UnauthorizedError();
    const body: unknown = await res.json().catch(() => undefined);
    if (!res.ok) {
      const code = errorCodeOf(body);
      throw new GatewayError(res.status, describeFailure(res.status, body), code);
    }
    return unwrapEnvelope(body);
  }
}

function describeFailure(status: number, body: unknown): string {
  if (typeof body === 'object' && body !== null && 'error' in body) {
    const err = (body as { error: unknown }).error;
    if (typeof err === 'object' && err !== null && 'message' in err) {
      const message = (err as { message: unknown }).message;
      if (typeof message === 'string') return message;
    }
  }
  return `gateway answered HTTP ${status}`;
}
