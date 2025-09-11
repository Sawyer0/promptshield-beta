import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'client', 'src'),
      '@shared': path.resolve(__dirname, 'shared'),
      '@assets': path.resolve(__dirname, 'attached_assets'),
    },
  },
  // Inline Radix UI packages so that vi.mock can intercept them
  server: {
    deps: {
      inline: [
        '@radix-ui/react-portal',
        '@radix-ui/react-select',
        '@radix-ui/react-dismissable-layer',
        '@radix-ui/react-focus-scope',
        '@radix-ui/react-popper',
        '@radix-ui/react-primitive',
        '@radix-ui/react-slot',
        '@radix-ui/react-collection',
      ],
    },
  },
  test: {
    pool: 'forks',
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./client/src/test/setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        'client/src/test/',
        '**/*.d.ts',
        '**/*.config.*',
        '**/coverage/**',
      ],
    },
  },
});
