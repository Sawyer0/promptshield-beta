import { test, expect, request as pwRequest } from '@playwright/test';

const APP_URL = process.env.APP_URL || 'http://localhost:3000';
const GW = process.env.PS_GATEWAY_URL || 'http://localhost:9090';
const FRONTEND_TOKEN = process.env.PS_FRONTEND_TOKEN || 'ps-admin-token-dev';

async function createTenant(name: string) {
  const r = await fetch(`${APP_URL}/api/v1/admin/tenants`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ name })
  });
  const j = await r.json();
  if (!r.ok) throw new Error(`createTenant ${name} failed: ${r.status}`);
  return j.id || j.data?.id || j?.tenant?.id;
}

async function uploadPackAndAssign(tenantId: string, yaml: string) {
  const up = await fetch(`${GW}/v1/rulepacks`, {
    method: 'POST',
    headers: { 'content-type': 'text/yaml', 'authorization': `Bearer ${FRONTEND_TOKEN}` },
    body: yaml,
  });
  const ju = await up.json();
  if (!up.ok) throw new Error(`upload pack failed: ${up.status}`);
  const packId = ju.id || ju.data?.id;
  const asn = await fetch(`${GW}/v1/admin/tenants/${tenantId}/assignments`, {
    method: 'POST',
    headers: { 'content-type': 'application/json', 'authorization': `Bearer ${FRONTEND_TOKEN}` },
    body: JSON.stringify({ rulepack_id: packId, target_scope: '*', method: '*', priority: 100 })
  });
  if (!asn.ok) throw new Error(`assignment failed: ${asn.status}`);
  return packId;
}

const PACK_BLOCK = `apiVersion: promptshield.io/v1\nkind: RulePack\nmetadata:\n  name: e2e-pack-block\nrules:\n  - id: e2e-block\n    level: 1\n    severity: HIGH\n    keywords: ["replace_me"]\n    response:\n      action: quarantine\n`;

const PACK_ALLOW = `apiVersion: promptshield.io/v1\nkind: RulePack\nmetadata:\n  name: e2e-pack-allow\nrules:\n  - id: harmless-rule\n    level: 1\n    severity: INFO\n    keywords: ["harmless"]\n    response:\n      action: flag\n`;

async function check(gw: string, tenant: string, body: string) {
  const ctx = await pwRequest.newContext();
const resp = await ctx.post(`${gw}/check`, {
    headers: { 'content-type': 'application/json', 'x-ps-tenant-id': tenant, 'x-ps-token': FRONTEND_TOKEN },
    data: { direction: 'request', tenant_id: tenant, content_type: 'text/plain', body }
  });
  const json = await resp.json();
  return { status: resp.status(), json };
}

// Multi‑tenant isolation: Tenant A blocks "replace_me", Tenant B allows it
// Validates isolation enforced by X-PS-Tenant-ID and per-tenant assignments.

test('multi-tenant isolation: different decisions per tenant', async () => {
  const tenantA = await createTenant(`e2e-tenant-A-${Date.now()}`);
  const tenantB = await createTenant(`e2e-tenant-B-${Date.now()}`);

  await uploadPackAndAssign(tenantA, PACK_BLOCK);
  await uploadPackAndAssign(tenantB, PACK_ALLOW);

  // Same input under different tenants
  const a = await check(GW, tenantA, 'replace_me');
  const b = await check(GW, tenantB, 'replace_me');

  // Expect A not allow; B likely allow (or at least different)
  expect((a.json.decision || '').toLowerCase()).not.toBe('allow');
  // If B returns allow (or anything but deny/quarantine), we consider it passing
  // Accept allow or flag/warn; assert not equal to A's decision to prove isolation
  expect((b.json.decision || '').toLowerCase()).not.toBe((a.json.decision || '').toLowerCase());
});

