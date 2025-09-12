#!/usr/bin/env node
// tenant-bootstrap.mjs
// Creates a tenant and mints a frontend token (re-using admin token) for quick-start.
// Usage (PowerShell example):
//   node scripts/tenant-bootstrap.mjs --name "Acme" --gateway http://localhost:9090 --admin-token {{PS_ADMIN_TOKEN}}

import http from 'http';
import https from 'https';

function parseArgs() {
  const args = process.argv.slice(2);
  const opts = { name: '', gateway: process.env.PS_GATEWAY_URL || 'http://localhost:9090', adminToken: process.env.PS_ADMIN_TOKEN || '' };
  for (let i = 0; i < args.length; i++) {
    const k = args[i];
    const v = args[i+1];
    if (k === '--name') { opts.name = v; i++; continue; }
    if (k === '--gateway') { opts.gateway = v; i++; continue; }
    if (k === '--admin-token') { opts.adminToken = v; i++; continue; }
  }
  if (!opts.name) {
    console.error('Missing --name');
    process.exit(1);
  }
  if (!opts.adminToken) {
    console.error('Missing --admin-token (PS_ADMIN_TOKEN)');
    process.exit(1);
  }
  return opts;
}

function reqJson(method, urlStr, body, headers={}) {
  return new Promise((resolve, reject) => {
    const u = new URL(urlStr);
    const lib = u.protocol === 'https:' ? https : http;
    const finalHeaders = { 'content-type': 'application/json', ...headers };
    const ct = (finalHeaders['content-type'] || finalHeaders['Content-Type'] || '').toLowerCase();
    const isYaml = ct.includes('yaml');
    const isJson = ct.includes('json');
    const req = lib.request({
      hostname: u.hostname,
      port: u.port || (u.protocol === 'https:' ? 443 : 80),
      path: u.pathname + (u.search || ''),
      method,
      headers: finalHeaders,
    }, (res) => {
      let data = '';
      res.on('data', (d) => data += d);
      res.on('end', () => {
        try { resolve({ status: res.statusCode, json: data ? JSON.parse(data) : {} }); }
        catch { resolve({ status: res.statusCode, json: { raw: data } }); }
      });
    });
    req.on('error', reject);
    if (body) {
      if (isYaml && typeof body === 'string') {
        req.write(body);
      } else if (isJson && typeof body !== 'string') {
        req.write(JSON.stringify(body));
      } else if (typeof body === 'string') {
        req.write(body);
      } else {
        req.write(JSON.stringify(body));
      }
    }
    req.end();
  });
}

async function main() {
  const { name, gateway, adminToken } = parseArgs();
// 1) Create tenant (handle 409 by fetching existing)
let tenantId = '';
const create = await reqJson('POST', `${gateway}/admin/tenants`, { name }, { 'Authorization': `Bearer ${adminToken}` });
if (create.status === 201) {
  tenantId = create.json?.id || create.json?.data?.id || create.json?.tenant?.id;
} else if (create.status === 409) {
  // List and find by name
  const list = await reqJson('GET', `${gateway}/admin/tenants`, null, { 'Authorization': `Bearer ${adminToken}` });
  const items = list.json?.tenants || list.json?.data?.tenants || [];
  const found = items.find(t => (t.name||'').toLowerCase() === name.toLowerCase());
  if (found) tenantId = found.id || found.ID || found.Id;
} else {
  console.error('Tenant create failed', create);
  process.exit(2);
}
if (!tenantId) {
  console.error('No tenant id available after create/list');
  process.exit(3);
}
// 2) Create a minimal RulePack and assign it to all endpoints/methods for this tenant
//    We create a simple pack that blocks the keyword "replace_me" in requests.
const packYaml = [
  'apiVersion: promptshield.io/v1',
  'kind: RulePack',
  'metadata:',
  '  name: bootstrap-pack',
  'rules:',
  '  - id: bootstrap-block',
  '    name: Block replace_me',
  '    level: 1',
  '    severity: HIGH',
  '    keywords:',
  '      - "replace_me"',
  '    response:',
  '      action: block',
  '      message: "Blocked by bootstrap policy"',
].join('\n');

// Upload RulePack via admin endpoint
const up = await reqJson('POST', `${gateway}/rulepacks?activate=true`, packYaml, {
  'Authorization': `Bearer ${adminToken}`,
  'content-type': 'text/yaml',
  'X-PS-Tenant-ID': tenantId,
  'X-PS-User-Admin': 'true',
});
let rulepackId = up.json?.id || up.json?.data?.id;
if ((up.status||0) >= 300 || !rulepackId) {
  // Fallback: list existing rulepacks and find by name (idempotent bootstrap)
  const list = await reqJson('GET', `${gateway}/rulepacks`, null, {
    'Authorization': `Bearer ${adminToken}`,
    'X-PS-Tenant-ID': tenantId,
    'X-PS-User-Admin': 'true',
  });
  const arr = Array.isArray(list.json) ? list.json : (Array.isArray(list.json?.data) ? list.json.data : []);
  const foundPack = (arr || []).find(p => (p.name||p.Name||'').toLowerCase() === 'bootstrap-pack');
  if (foundPack) {
    rulepackId = foundPack.id || foundPack.ID || foundPack.Id;
  }
  if (!rulepackId) {
    console.error('RulePack upload failed', up);
    process.exit(4);
  }
}

// Assign the pack
if (rulepackId) {
const asn = await reqJson('POST', `${gateway}/admin/tenants/${tenantId}/assignments`, {
    rulepack_id: rulepackId,
    target_scope: '*',
    method: '*',
    priority: 100,
  }, {
    'Authorization': `Bearer ${adminToken}`,
    'X-PS-Tenant-ID': tenantId,
    'X-PS-User-Admin': 'true',
  });
  if ((asn.status||0) >= 300) {
    console.error('Assignment create failed', asn);
  }
}

// 3) For quick-start, reuse frontend token = PS_ADMIN_TOKEN (or configure dedicated token flow later)
const frontendToken = adminToken;
console.log(JSON.stringify({ tenant_id: tenantId, rulepack_id: rulepackId, frontend_token: frontendToken, gateway }, null, 2));
}

main().catch((e) => { console.error(e); process.exit(10); });

