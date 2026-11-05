import { fileURLToPath } from 'node:url';
import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

// The build lands inside the Go embed directory so `go build ./gateway`
// ships it. During `npm run dev`, API calls proxy to a local gateway.
const gateway = process.env.PS_GATEWAY_URL ?? 'http://127.0.0.1:9090';
const outDir = fileURLToPath(new URL('../../internal/interfaces/http/dashboard/dist', import.meta.url));

export default defineConfig({
  base: '/dashboard/',
  plugins: [react()],
  build: {
    outDir,
    emptyOutDir: false,
    sourcemap: false,
  },
  server: {
    proxy: {
      '/v1': gateway,
      '/check': gateway,
      '/healthz': gateway,
      '/readyz': gateway,
    },
  },
  test: {
    // Pure modules run in node; component tests opt into jsdom per file.
    environment: 'node',
    setupFiles: ['./test/setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
    restoreMocks: true,
  },
});
