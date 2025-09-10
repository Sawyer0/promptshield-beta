import '@testing-library/jest-dom';
import { beforeAll, afterEach, afterAll, vi } from 'vitest';
import { server } from './mocks/server';

// Ensure React wraps updates in tests
// Helps reduce false-positive act(...) warnings from libraries that schedule microtasks
// See https://react.dev/reference/react/act#handling-asynchronous-tests
;(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;

// Polyfill requestAnimationFrame/cancelAnimationFrame for libraries using RAF in effects
if (!(globalThis as any).requestAnimationFrame) {
  (globalThis as any).requestAnimationFrame = (cb: FrameRequestCallback) => setTimeout(cb, 0) as unknown as number;
}
if (!(globalThis as any).cancelAnimationFrame) {
  (globalThis as any).cancelAnimationFrame = (id: number) => clearTimeout(id as unknown as any);
}

// Establish API mocking before all tests
// Allow local supertest requests (127.0.0.1/localhost) to pass through without errors
beforeAll(() => server.listen({
  onUnhandledRequest(req, print) {
    try {
      const url = new URL((req as any).url?.href || String((req as any).url));
      const host = (url.hostname || '').toLowerCase();
      if (host === '127.0.0.1' || host === 'localhost') {
        // Ignore unhandled requests to local test servers (used by supertest)
        return;
      }
    } catch {
      // If URL parsing fails, fall back to default error behavior
    }
    print.error();
  }
}));

// Reset any request handlers that we may add during the tests,
// so they don't affect other tests
afterEach(() => server.resetHandlers());

// Clean up after the tests are finished
afterAll(() => server.close());

// Global fetch mock that tests can override per-case
if (!(global as any).fetch) {
  (global as any).fetch = vi.fn();
}

// Polyfill ResizeObserver used by Radix UI
class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}
if (!(global as any).ResizeObserver) {
  (global as any).ResizeObserver = ResizeObserverMock as any;
}

// Polyfill scrollIntoView used by Radix Select
if (!(Element.prototype as any).scrollIntoView) {
  (Element.prototype as any).scrollIntoView = vi.fn();
}

// Polyfill pointer capture APIs used by Radix Select
if (!(Element.prototype as any).hasPointerCapture) {
  (Element.prototype as any).hasPointerCapture = () => false;
}
if (!(Element.prototype as any).setPointerCapture) {
  (Element.prototype as any).setPointerCapture = () => {};
}
if (!(Element.prototype as any).releasePointerCapture) {
  (Element.prototype as any).releasePointerCapture = () => {};
}

// user-event pointer events check workaround for Radix overlays in jsdom
// Ensure document body allows pointer events by default
document.body.style.pointerEvents = 'auto';

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
};
Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

// Mock navigator.clipboard if not present
if (!('clipboard' in navigator)) {
  Object.defineProperty(navigator, 'clipboard', {
    value: {
      writeText: vi.fn(),
    },
    configurable: true,
  });
}

// Ensure Toaster has a safe default for toasts list by mocking useToast hook
vi.mock('@/hooks/use-toast', () => ({
  useToast: () => ({
    toasts: [],
    toast: vi.fn(),
    dismiss: vi.fn(),
  }),
}));

// Mock window.location (minimal)
Object.defineProperty(window, 'location', {
  value: {
    href: 'http://localhost:3000',
    origin: 'http://localhost:3000',
    pathname: '/',
    search: '',
    hash: '',
  },
  writable: true,
});
