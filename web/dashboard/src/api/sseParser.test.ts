import { describe, expect, it } from 'vitest';
import { SseParser } from './sseParser';

describe('SseParser', () => {
  it('parses complete messages with event names', () => {
    const p = new SseParser();
    expect(p.push('event: ready\ndata: {"status":"ok"}\n\n')).toEqual([{ event: 'ready', data: '{"status":"ok"}' }]);
  });

  it('reassembles messages split across chunks', () => {
    const p = new SseParser();
    expect(p.push('event: deci')).toEqual([]);
    expect(p.push('sion\ndata: {"a":')).toEqual([]);
    expect(p.push('1}\n')).toEqual([]);
    expect(p.push('\n')).toEqual([{ event: 'decision', data: '{"a":1}' }]);
  });

  it('joins multi-line data, ignores comments, defaults the event name', () => {
    const p = new SseParser();
    const out = p.push(': keep-alive\ndata: one\ndata: two\n\nevent: x\ndata:three\n\n');
    expect(out).toEqual([
      { event: 'message', data: 'one\ntwo' },
      { event: 'x', data: 'three' },
    ]);
  });

  it('accepts CRLF line endings and skips events without data', () => {
    const p = new SseParser();
    expect(p.push('event: ping\r\n\r\nevent: a\r\ndata: 1\r\n\r\n')).toEqual([{ event: 'a', data: '1' }]);
  });
});
