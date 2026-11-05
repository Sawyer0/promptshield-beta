// Long-lived subscription to GET /v1/events. Reconnects with exponential
// backoff because the gateway closes streams at its request timeout; the
// caller re-syncs history on every "live" transition to cover any gap.
import type { DecisionRecord, StreamStatus } from './contracts';
import { decodeDecision } from './decoders';
import type { GatewayClient } from './gatewayClient';
import { SseParser, type SseMessage } from './sseParser';

export interface StreamHandlers {
  onDecision: (record: DecisionRecord) => void;
  onRulesReloaded: () => void;
  onStatus: (status: StreamStatus) => void;
}

export interface StreamOptions {
  client: GatewayClient;
  signal: AbortSignal;
  initialBackoffMs?: number;
  maxBackoffMs?: number;
  sleep?: (ms: number, signal: AbortSignal) => Promise<void>;
}

const EVENT_TYPES = 'decision,rules_reloaded';

function abortableSleep(ms: number, signal: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    const timer = setTimeout(resolve, ms);
    signal.addEventListener('abort', () => {
      clearTimeout(timer);
      resolve();
    });
  });
}

/** Runs until the signal aborts or the gateway rejects the token. */
export async function runEventStream(options: StreamOptions, handlers: StreamHandlers): Promise<void> {
  const { client, signal } = options;
  const sleep = options.sleep ?? abortableSleep;
  const initial = options.initialBackoffMs ?? 500;
  const max = options.maxBackoffMs ?? 10_000;
  let backoff = initial;

  handlers.onStatus('connecting');
  while (!signal.aborted) {
    try {
      const res = await client.openEventStream(EVENT_TYPES, signal);
      if (res.status === 401) {
        handlers.onStatus('unauthorized');
        return;
      }
      if (res.ok && res.body) {
        await consume(res.body, signal, (message) => {
          if (message.event === 'ready') {
            backoff = initial;
            handlers.onStatus('live');
          } else {
            dispatch(message, handlers);
          }
        });
      }
    } catch {
      // Network failure or abort; fall through to backoff or exit.
    }
    if (signal.aborted) break;
    handlers.onStatus('reconnecting');
    await sleep(backoff, signal);
    backoff = Math.min(backoff * 2, max);
  }
  handlers.onStatus('closed');
}

async function consume(
  body: ReadableStream<Uint8Array>,
  signal: AbortSignal,
  onMessage: (message: SseMessage) => void,
): Promise<void> {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  const parser = new SseParser();
  const cancel = () => void reader.cancel().catch(() => undefined);
  signal.addEventListener('abort', cancel);
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) return;
      for (const message of parser.push(decoder.decode(value, { stream: true }))) {
        onMessage(message);
      }
    }
  } finally {
    signal.removeEventListener('abort', cancel);
  }
}

function dispatch(message: SseMessage, handlers: StreamHandlers): void {
  if (message.event === 'rules_reloaded') {
    handlers.onRulesReloaded();
    return;
  }
  if (message.event !== 'decision') return;
  try {
    const envelope: unknown = JSON.parse(message.data);
    const data = typeof envelope === 'object' && envelope !== null ? (envelope as { data?: unknown }).data : undefined;
    handlers.onDecision(decodeDecision(data, 'event.data'));
  } catch {
    // Ignore malformed events rather than dropping the stream.
  }
}
