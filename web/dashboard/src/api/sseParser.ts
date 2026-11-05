// Incremental Server-Sent Events parser (WHATWG event-stream format).
// The dashboard reads /v1/events with fetch instead of EventSource so the
// admin token travels in the Authorization header, never in the URL.

export interface SseMessage {
  event: string;
  data: string;
}

export class SseParser {
  private buffer = '';
  private event = '';
  private data: string[] = [];

  /** Feeds a decoded text chunk and returns the messages it completed. */
  push(chunk: string): SseMessage[] {
    this.buffer += chunk;
    const out: SseMessage[] = [];
    let newline = this.buffer.search(/\r\n|\r|\n/);
    while (newline !== -1) {
      const line = this.buffer.slice(0, newline);
      const width = this.buffer.startsWith('\r\n', newline) ? 2 : 1;
      this.buffer = this.buffer.slice(newline + width);
      const message = this.line(line);
      if (message) out.push(message);
      newline = this.buffer.search(/\r\n|\r|\n/);
    }
    return out;
  }

  private line(line: string): SseMessage | undefined {
    if (line === '') return this.dispatch();
    if (line.startsWith(':')) return undefined;
    const colon = line.indexOf(':');
    const field = colon === -1 ? line : line.slice(0, colon);
    let value = colon === -1 ? '' : line.slice(colon + 1);
    if (value.startsWith(' ')) value = value.slice(1);
    if (field === 'event') this.event = value;
    if (field === 'data') this.data.push(value);
    return undefined;
  }

  private dispatch(): SseMessage | undefined {
    const message = this.data.length > 0 ? { event: this.event || 'message', data: this.data.join('\n') } : undefined;
    this.event = '';
    this.data = [];
    return message;
  }
}
