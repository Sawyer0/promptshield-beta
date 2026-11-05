import { fileURLToPath } from 'node:url';

// Shared by playwright.config.ts (which starts the gateway) and the specs.
export const ADMIN_TOKEN = 'e2e-admin-token';
export const SCRATCH_DIR = process.env.PS_E2E_SCRATCH ?? fileURLToPath(new URL('../.e2e-scratch/', import.meta.url));
export const RULEPACK_PATH = `${SCRATCH_DIR}e2e-rulepack.yaml`;
export const RULEPACK_FIXTURE = fileURLToPath(new URL('./fixtures/rulepack.yaml', import.meta.url));
export const SCREENSHOT_PATH = fileURLToPath(new URL('../../../docs/images/dashboard.png', import.meta.url));
