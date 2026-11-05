import { copyFileSync, mkdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { defineConfig, devices } from '@playwright/test';
import { ADMIN_TOKEN, RULEPACK_FIXTURE, RULEPACK_PATH, SCRATCH_DIR } from './e2e/env';

// The e2e suite runs against the REAL gateway binary (go build ./gateway),
// serving the embedded dashboard build at /dashboard. The rulepack is copied
// to a scratch path so the reload test can rewrite it.
mkdirSync(SCRATCH_DIR, { recursive: true });
copyFileSync(RULEPACK_FIXTURE, RULEPACK_PATH);
const binary = process.env.PS_GATEWAY_BIN ?? fileURLToPath(new URL('../../bin/ps-gateway-e2e', import.meta.url));

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  timeout: 30_000,
  use: {
    baseURL: 'http://127.0.0.1:9090',
    trace: 'retain-on-failure',
    viewport: { width: 1440, height: 1000 },
  },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1440, height: 1000 },
        // Optional local override for platforms without a Playwright build.
        ...(process.env.PW_CHROMIUM_PATH ? { launchOptions: { executablePath: process.env.PW_CHROMIUM_PATH } } : {}),
      },
    },
  ],
  webServer: {
    command: binary,
    url: 'http://127.0.0.1:9090/readyz',
    reuseExistingServer: false,
    timeout: 30_000,
    stdout: 'ignore',
    stderr: 'pipe',
    env: {
      PS_ENFORCER_TLS_MODE: 'disable',
      PS_ENFORCER_ADMIN_TOKEN: ADMIN_TOKEN,
      PS_ENFORCER_RULEPACK: RULEPACK_PATH,
      PS_ENFORCER_MODE: 'enforce',
      PS_REQUIRE_RULEPACK_AT_STARTUP: 'true',
      PS_TELEMETRY: 'false',
    },
  },
});
