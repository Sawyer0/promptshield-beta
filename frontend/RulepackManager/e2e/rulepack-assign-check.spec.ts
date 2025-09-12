import { test, expect } from '@playwright/test';

// E2E: create tenant, create rulepack, assign policy, run a live /v1/check via BFF proxy
// Assumes the app is running at http://localhost:3000 and gateway at promptshield:9090
// Uses dev bypass if enabled in your environment; otherwise requires Clerk auth

const APP_URL = process.env.APP_URL || 'http://localhost:3000';
const TENANT_NAME = process.env.E2E_TENANT_NAME || `e2e-tenant-${Date.now()}`;

// Helper: create a tenant via BFF admin route (UI would normally drive this)
async function createTenantViaAPI() {
  const resp = await fetch(`${APP_URL}/api/v1/admin/tenants`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ name: TENANT_NAME }),
  });
  const json = await resp.json();
  if (!resp.ok) throw new Error(`createTenant failed: ${resp.status} ${JSON.stringify(json)}`);
  return json.id || json.data?.id || json?.tenant?.id;
}

// Minimal pack to block the word "replace_me" in requests
const PACK_YAML = `apiVersion: promptshield.io/v1\nkind: RulePack\nmetadata:\n  name: e2e-pack\nrules:\n  - id: e2e-block\n    level: 1\n    severity: HIGH\n    keywords: ["replace_me"]\n    response:\n      action: quarantine\n`;

test('Create tenant, add rulepack, assign, and verify decision', async ({ page, request }) => {
  // 1) Load app
  await page.goto(APP_URL);

  // If a sign-in UI appears, the test runner needs credentials or dev bypass.
  // For now assume dev bypass or prior session is active.

  // 2) Create tenant via API (admin; BFF forwards auth)
  const tenantId = await createTenantViaAPI();
  expect(tenantId).toBeTruthy();

  // 3) Select tenant in UI (if a tenant selector appears)
  try {
    await page.getByText('Choose Tenant').waitFor({ timeout: 2000 });
    await page.getByRole('button', { name: /Select/i }).first().click();
  } catch {}

  // 4) Navigate to RulePacks and create one
  await page.getByRole('link', { name: /RulePacks/i }).click();
  await page.getByRole('button', { name: /New RulePack|Create RulePack|Add RulePack/i }).click();
  await page.getByLabel(/YAML/i).fill(PACK_YAML);
  await page.getByRole('button', { name: /Save|Create/i }).click();
  await expect(page.getByText(/e2e-pack/i)).toBeVisible({ timeout: 10000 });

  // 5) Assign policy to endpoints (simplest: method * and scope *)
  await page.getByRole('link', { name: /Policies|Assignments/i }).click();
  await page.getByRole('button', { name: /Add Assignment|Create Assignment|New Assignment/i }).click();
  await page.getByLabel(/RulePack/i).selectOption({ label: /e2e-pack/i });
  try { await page.getByLabel(/Method/i).selectOption('*'); } catch {}
  try { await page.getByLabel(/Endpoint|Scope/i).fill('*'); } catch {}
  await page.getByRole('button', { name: /Save|Create/i }).click();

  // 6) Verify live decision via BFF → Gateway
const gw = process.env.PS_GATEWAY_URL || 'http://localhost:9090';
  const check = await request.post(`${gw}/check`, {
    headers: {
      'content-type': 'application/json',
      'x-ps-tenant-id': tenantId,
      'x-ps-token': process.env.PS_FRONTEND_TOKEN || 'ps-admin-token-dev',
    },
    data: {
      direction: 'request',
      tenant_id: tenantId,
      content_type: 'text/plain',
      body: 'replace_me',
    },
  });
  expect(check.ok()).toBeTruthy();
  const decision = await check.json();
  // In enforce/quarantine mode this may be a 403, but body should indicate block/quarantine
  // Accept either allow with message (observe) or quarantine/deny depending on mode
  // Here we assert decision is not 'allow'
  expect((decision.decision || '').toLowerCase()).not.toBe('allow');
});

