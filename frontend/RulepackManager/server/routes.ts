import type { Express, Response } from "express";
import { createServer, type Server } from "http";
import { isAuthenticated, getUserInfo } from "./clerkAuth";
import { createGatewayProxy } from "./gatewayProxy";
import { createClerkClient } from '@clerk/backend';
import { extractUserContext, generateGatewayJWT } from './jwtAuth.js';
import { registerMetricsApi } from '../api/metrics.js';

// Production BFF router: delegate app endpoints to the Go gateway via /api proxy
// Keep only auth/session endpoints and health locally.
import { isDevBypassEnabled } from './devBypass';

// Helper: determine if we are in dev-bypass mode
function isDevBypassEnv(): boolean {
  return isDevBypassEnabled();
}

// Helper: check if self-serve tenant signup is allowed
function allowSelfTenantSignup(): boolean {
  const dev = isDevBypassEnv();
  if (dev) return true;
  return (process.env.PS_ALLOW_SELF_TENANT_SIGNUP || '').toLowerCase() === 'true';
}

export async function registerRoutes(app: Express): Promise<Server> {
  // Auth middleware is mounted in server/index.ts as the very first middleware

  if ((process.env.NODE_ENV || '').toLowerCase() === 'test') {
    console.log('REGISTER_ROUTES_START');
  }

  // Simple test route for debugging
  app.get('/test-simple', (req, res) => {
    if ((process.env.NODE_ENV || '').toLowerCase() === 'test') {
      console.log('TEST_SIMPLE_ROUTE', { path: req.path });
    }
    res.json({ success: true, message: 'simple route works' });
  });

  // Minimal user endpoint derived from session claims (no DB dependency)
  app.get('/api/auth/user', isAuthenticated, async (req: any, res: Response) => {
    try {
      if ((process.env.NODE_ENV || '').toLowerCase() === 'test') {
        console.log('AUTH_USER_ENDPOINT', { path: req.path, hasAuth: !!req.auth, userId: req.auth?.userId });
      }
      const user = await getUserInfo(req);
      if (!user) {
        return res.status(401).json({ 
          error: {
            code: 'UNAUTHORIZED',
            message: 'User information not available',
            details: { reason: 'user_info_missing' }
          }
        });
      }

      // Derive roles
      const roles: string[] = [];
      // Platform admin via env allowlist or dev bypass
      const platformAdmins = (process.env.PS_PLATFORM_ADMINS || '').split(',').map(s => s.trim().toLowerCase()).filter(Boolean);
      const emailLc = String(user.email || '').toLowerCase();
      const devBypass = isDevBypassEnv();
      const devIsAdmin = (process.env.PS_DEV_IS_ADMIN || '').toLowerCase() === 'true';
      if ((platformAdmins.includes(emailLc)) || (devBypass && devIsAdmin)) {
        roles.push('platform_admin');
      }
      // Tenant role from signed cookie set during /api/orgs/select
      const tenantRole = (req.signedCookies && req.signedCookies.ps_tenant_role) || req.cookies?.ps_tenant_role || '';
      const tRole = String(tenantRole).toLowerCase();
      if (tRole === 'admin') roles.push('tenant_admin');
      else if (tRole) roles.push('developer'); // default member -> developer
      // Allow override via dev roles
      const devRoles = (process.env.PS_DEV_ROLES || '').split(',').map(s => s.trim()).filter(Boolean);
      for (const r of devRoles) if (!roles.includes(r)) roles.push(r);
      // Fallback default role
      if (!roles.length) roles.push('developer');

      // Best-effort sync to backend user directory
      try {
        const { generateGatewayJWT, extractUserContext } = await import('./jwtAuth.js');
        const ctx = extractUserContext(req);
        const token = generateGatewayJWT(ctx);
        await fetch(`${process.env.PS_GATEWAY_URL || 'http://localhost:8098'}/v1/users/sync`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
          },
          body: JSON.stringify({ email: user.email, first_name: user.firstName, last_name: user.lastName })
        });
      } catch (syncError) {
        console.warn('Backend user sync failed:', syncError instanceof Error ? syncError.message : String(syncError));
        // Don't fail the request if sync fails
      }

      res.json({ ...user, systemRole: roles.includes('platform_admin') ? 'admin' : 'user', roles });
    } catch (error) {
      console.error('Auth user endpoint error:', error);
      res.status(500).json({
        error: {
          code: 'INTERNAL_ERROR',
          message: 'Failed to get user information',
          details: { 
            reason: error instanceof Error ? error.message : String(error)
          }
        }
      });
    }
  });

  // Health under /api for convenience
  app.get('/api/healthz', (_req, res) => {
    res.json({ status: 'ok', timestamp: new Date().toISOString() });
  });

  // Metrics API (Prometheus-backed) — Clerk-protected upstream in gatewayProxy, but this route is local
  registerMetricsApi(app);

  // Compliance: return static mapping registry (repo default)
  app.get('/api/compliance/mapping', isAuthenticated, async (_req: any, res: Response) => {
    try {
      const fs = await import('fs');
      const path = await import('path');
      const filePath = path.resolve(process.cwd(), 'compliance', 'mappings', 'mapping.json');
      if (!fs.existsSync(filePath)) {
        return res.status(404).json({ error: { code: 'NOT_FOUND', message: 'mapping.json not found' } });
      }
      const data = JSON.parse(fs.readFileSync(filePath, 'utf8'));
      res.json({ success: true, data });
    } catch (error) {
      res.status(500).json({ error: { code: 'COMPLIANCE_MAPPING_ERROR', message: 'Failed to load mapping', details: String(error instanceof Error ? error.message : error) } });
    }
  });

  // Compliance: export evidence (aggregated from mapping + gateway audits/configs)
  app.get('/api/compliance/export', isAuthenticated, async (req: any, res: Response) => {
    try {
      const fs = await import('fs');
      const path = await import('path');

      // Load mapping
      const filePath = path.resolve(process.cwd(), 'compliance', 'mappings', 'mapping.json');
      if (!fs.existsSync(filePath)) {
        return res.status(404).json({ error: { code: 'NOT_FOUND', message: 'mapping.json not found' } });
      }
      const mapping = JSON.parse(fs.readFileSync(filePath, 'utf8'));

      // Params
      const framework = String(req.query.framework || '').trim();
      const controlsParam = String(req.query.controls || '').trim();
      const from = String(req.query.from || '');
      const to = String(req.query.to || '');
      const format = (String(req.query.format || 'json').toLowerCase() === 'csv') ? 'csv' : 'json';

      // Tenant context and gateway headers
      const tenantId = (req.signedCookies && req.signedCookies.ps_tenant_id) || req.cookies?.ps_tenant_id || null;
      const gw = process.env.PS_GATEWAY_URL || 'http://localhost:8098';
      const userCtx = extractUserContext(req);
      const token = generateGatewayJWT(userCtx);
      const commonHeaders: any = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
        ...(tenantId ? { 'X-PS-Tenant-ID': tenantId } : {}),
      };

      // Helper to try multiple path prefixes
      const tryGet = async (method: 'GET'|'POST', basePath: string, body?: any): Promise<any | null> => {
        const candidates = [basePath, `/v1${basePath}`, `/api${basePath}`, `/v1/api${basePath}`];
        for (const p of candidates) {
          try {
            const resp = await fetch(`${gw}${p}`, {
              method,
              headers: commonHeaders,
              ...(body ? { body: JSON.stringify(body) } : {}),
            } as any);
            if (resp.ok) {
              const ct = resp.headers.get('content-type') || '';
              return ct.includes('application/json') ? await resp.json() : await resp.text();
            }
          } catch (_) {}
        }
        return null;
      };

      // Build control list
      const fw = mapping?.frameworks?.[framework];
      if (!fw) {
        return res.status(400).json({ error: { code: 'INVALID_FRAMEWORK', message: `Unknown framework: ${framework}` } });
      }
      const controlIds = controlsParam ? controlsParam.split(',').map((s: string) => s.trim()).filter(Boolean) : Object.keys(fw);

      // Time range filters (best-effort)
      const rangeFilters: any = {};
      if (from) rangeFilters.start_time = from;
      if (to) rangeFilters.end_time = to;

      // Optionally fetch all rulepacks once to check for controls tags
      let rulepacksList: any[] = [];
      try {
        const rp = await tryGet('GET', '/rulepacks');
        rulepacksList = Array.isArray(rp) ? rp : (Array.isArray(rp?.data) ? rp.data : (Array.isArray(rp?.rulepacks) ? rp.rulepacks : []));
      } catch (_) {}

      // Aggregate evidence per control
      const controlsDetailed: any[] = [];
      for (const cid of controlIds) {
        const ctrl = fw[cid];
        if (!ctrl) continue;
        const evDef = ctrl.evidence || {};

        // Audits: limited sample
        let auditSummary: any = { total: 0, sample: [] as any[] };
        if (evDef.audits && Array.isArray(evDef.audits.actions) && evDef.audits.actions.length) {
          const body: any = {
            actions: evDef.audits.actions,
            limit: 50,
            offset: 0,
            ...rangeFilters,
          };
          if (evDef.audits.filters && typeof evDef.audits.filters === 'object') {
            body.filters = evDef.audits.filters;
          }
          const auditRes = await tryGet('POST', '/admin/audits/search', body);
          const events = Array.isArray(auditRes?.data) ? auditRes.data : (Array.isArray(auditRes) ? auditRes : []);
          auditSummary.total = typeof auditRes?.total === 'number' ? auditRes.total : events.length;
          auditSummary.sample = events.slice(0, 10).map((e: any) => ({
            id: e.id,
            action: e.action,
            ts: e.timestamp || e.created_at,
            decision: e.metadata?.decision,
            reason: e.metadata?.reason,
            path: e.metadata?.path,
          }));
        }

        // Config snapshots
        const configs: Record<string, any> = {};
        if (Array.isArray(evDef.configs)) {
          for (const c of evDef.configs) {
            if (c === 'rulepacks/active') {
              const data = await tryGet('GET', '/rulepacks/active');
              configs['rulepacks/active'] = data ?? null;
            } else if (c === 'assignments') {
              if (tenantId) {
                const data = await tryGet('GET', `/admin/tenants/${tenantId}/assignments`);
                configs['assignments'] = (Array.isArray(data?.assignments) ? data.assignments : (Array.isArray(data?.data) ? data.data : data)) ?? null;
              } else {
                configs['assignments'] = null;
              }
            } else if (c === 'enforcement_mode') {
              const sys = await tryGet('GET', '/system/info');
              configs['enforcement_mode'] = sys?.enforcement_mode ?? sys?.data?.enforcement_mode ?? null;
            } else if (c === 'audit_chain_integrity') {
              const sys = await tryGet('GET', '/system/info');
              configs['audit_chain_integrity'] = typeof sys?.version === 'string' ? { system_version: sys.version } : (sys ?? null);
            } else if (c === 'retention_policy') {
              // Placeholder until backend exposes retention config endpoint
              configs['retention_policy'] = null;
            } else if (c === 'tools/registry') {
              const tools = await tryGet('GET', '/tools');
              const list = Array.isArray(tools?.data) ? tools.data : (Array.isArray(tools) ? tools : (Array.isArray(tools?.tools) ? tools.tools : []));
              configs['tools/registry'] = list;
            } else if (c === 'preferences.egress_allowlist') {
              const prefs = await tryGet('GET', '/admin/settings');
              configs['preferences.egress_allowlist'] = prefs?.egress_allowlist ?? prefs?.data?.egress_allowlist ?? null;
            } else if (c === 'preferences.require_approval_tools') {
              const prefs = await tryGet('GET', '/admin/settings');
              configs['preferences.require_approval_tools'] = prefs?.require_approval_tools ?? prefs?.data?.require_approval_tools ?? null;
            } else if (c.startsWith('preferences.')) {
              const prefs = await tryGet('GET', '/admin/settings');
              const key = c.split('preferences.')[1];
              configs[c] = prefs?.[key] ?? prefs?.data?.[key] ?? null;
            }
          }
        }

        // Rules with this control tag present across rulepacks
        let rules_with_controls = 0;
        let rulepacks_with_controls = 0;
        try {
          for (const rp of rulepacksList) {
            const rules = Array.isArray(rp?.rules) ? rp.rules : (Array.isArray(rp?.spec?.rules) ? rp.spec.rules : []);
            if (!Array.isArray(rules) || !rules.length) continue;
            let anyInPack = false;
            for (const r of rules) {
              const ctrls = Array.isArray(r?.controls) ? r.controls : [];
              if (ctrls.includes(cid)) {
                rules_with_controls += 1;
                anyInPack = true;
              }
            }
            if (anyInPack) rulepacks_with_controls += 1;
          }
        } catch (_) {}

        // Summary
        const summary = {
          rules: Array.isArray(evDef.rules) ? evDef.rules.length : 0,
          rule_tags: Array.isArray(evDef.rule_tags) ? evDef.rule_tags.length : 0,
          audits_actions: Array.isArray(evDef.audits?.actions) ? evDef.audits.actions.length : 0,
          configs: Array.isArray(evDef.configs) ? evDef.configs.length : 0,
          reports: Array.isArray(evDef.reports) ? evDef.reports.length : 0,
          audit_events_found: auditSummary.total,
          rules_with_controls,
          rulepacks_with_controls,
        } as any;

        controlsDetailed.push({
          control_id: cid,
          name: ctrl.name,
          description: ctrl.description,
          mapping: evDef,
          audit: auditSummary,
          configs,
          summary,
        });
      }

      const report = {
        framework,
        tenant_id: tenantId,
        generated_at: new Date().toISOString(),
        range: { from: from || null, to: to || null },
        controls: controlsDetailed,
        version: mapping?.version || '0.0.0',
      };

      if (format === 'csv') {
        const headers = [
          'framework','control_id','control_name','tenant_id','evidence_period_start','evidence_period_end','rules','rule_tags','audits_actions','audit_events_found','rules_with_controls','rulepacks_with_controls','configs','reports','generated_at'
        ];
        const rows = controlsDetailed.map((c: any) => [
          framework,
          c.control_id,
          JSON.stringify(c.name || ''),
          tenantId || '',
          from || '',
          to || '',
          String(c.summary.rules || 0),
          String(c.summary.rule_tags || 0),
          String(c.summary.audits_actions || 0),
          String(c.summary.audit_events_found || 0),
          String(c.summary.rules_with_controls || 0),
          String(c.summary.rulepacks_with_controls || 0),
          String(c.summary.configs || 0),
          String(c.summary.reports || 0),
          report.generated_at,
        ]);
        const csv = [headers.join(','), ...rows.map(r => r.join(','))].join('\n');
        res.setHeader('Content-Type', 'text/csv');
        res.setHeader('Content-Disposition', `attachment; filename="compliance_export_${framework}_${Date.now()}.csv"`);
        return res.status(200).send(csv);
      }

      return res.json({ success: true, data: report });
    } catch (error) {
      res.status(500).json({ error: { code: 'COMPLIANCE_EXPORT_ERROR', message: 'Failed to export compliance evidence', details: String(error instanceof Error ? error.message : error) } });
    }
  });

  // Simple waitlist endpoint: append JSONL to local file (server-only)
  app.post('/api/waitlist', async (req: any, res: Response) => {
    try {
      const body = req.body || {};
      const now = new Date().toISOString();
      const entry = { ...body, _ip: req.ip, _ua: req.headers['user-agent'], _ts: now };
      const fs = await import('fs');
      const path = await import('path');
      const outDir = process.env.WAITLIST_DIR || path.resolve(process.cwd(), 'data');
      const outFile = path.resolve(outDir, 'waitlist.jsonl');
      if (!fs.existsSync(outDir)) fs.mkdirSync(outDir, { recursive: true });
      fs.appendFileSync(outFile, JSON.stringify(entry) + '\n', 'utf8');
      res.status(204).end();
    } catch (e: any) {
      res.status(500).json({ message: 'failed to record waitlist', details: String(e?.message || e) });
    }
  });

  // Onboarding: role/view selection logger (server-only, optional)
  app.post('/api/onboarding/role', isAuthenticated, async (req: any, res: Response) => {
    try {
      const body = req.body || {};
      const now = new Date().toISOString();
      const user = await getUserInfo(req).catch(() => null);
      const entry = {
        type: 'role_selection',
        selection: body?.selection,
        roles: body?.roles,
        use_case: body?.useCase || body?.use_case || null,
        deploy: body?.deploy || null,
        industry: body?.industry || null,
        frameworks: Array.isArray(body?.frameworks) ? body.frameworks : undefined,
        environment: body?.environment || null,
        tenantId: (req.signedCookies && req.signedCookies.ps_tenant_id) || req.cookies?.ps_tenant_id || null,
        user: user ? { email: user.email, firstName: user.firstName, lastName: user.lastName } : null,
        _ip: req.ip, _ua: req.headers['user-agent'], _ts: now,
      };
      const fs = await import('fs');
      const path = await import('path');
      const dir = process.env.ONBOARDING_DIR || path.resolve(process.cwd(), 'data');
      const file = path.resolve(dir, 'onboarding-roles.jsonl');
      if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
      fs.appendFileSync(file, JSON.stringify(entry) + '\n', 'utf8');
      res.status(204).end();
    } catch (e: any) {
      res.status(500).json({ message: 'failed to record role selection', details: String(e?.message || e) });
    }
  });

  // Onboarding: request access to tenant (notify tenant admins)
  app.post('/api/onboarding/request-access', isAuthenticated, async (req: any, res: Response) => {
    try {
      const { desiredRole, note, orgId } = req.body || {};
      const fs = await import('fs');
      const path = await import('path');
      const now = new Date().toISOString();
      const user = await getUserInfo(req).catch(() => null);

      // If we had email sending configured, we would mail tenant admins here.
      // For now, append a request log and return 202.
      const dir = process.env.ONBOARDING_DIR || path.resolve(process.cwd(), 'data');
      const file = path.resolve(dir, 'onboarding-access-requests.jsonl');
      if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
      const entry = {
        type: 'access_request',
        desiredRole,
        note,
        orgId: orgId || null,
        user: user ? { email: user.email, firstName: user.firstName, lastName: user.lastName } : null,
        _ip: req.ip, _ua: req.headers['user-agent'], _ts: now,
      };
      fs.appendFileSync(file, JSON.stringify(entry) + '\n', 'utf8');
      res.status(202).json({ accepted: true });
    } catch (e: any) {
      res.status(500).json({ message: 'failed to record access request', details: String(e?.message || e) });
    }
  });

  // Debug endpoints for authentication troubleshooting
  app.get('/api/debug/auth', isAuthenticated, async (req: any, res: Response) => {
    try {
      if ((process.env.NODE_ENV || '').toLowerCase() === 'test') {
        console.log('DEBUG_AUTH_ENDPOINT', { path: req.path, hasAuth: !!req.auth, userId: req.auth?.userId });
      }
      const { getJWTConfigStatus, validateJWTToken } = await import('./jwtAuth.js');
      const user = await getUserInfo(req);
      const userContext = extractUserContext(req);
      
      // Generate a test JWT to validate configuration
      let testJWT = null;
      let jwtValidation = null;
      try {
        testJWT = generateGatewayJWT(userContext);
        jwtValidation = validateJWTToken(testJWT);
      } catch (jwtError) {
        jwtValidation = { 
          valid: false, 
          error: jwtError instanceof Error ? jwtError.message : String(jwtError) 
        };
      }

      res.json({
        auth: {
          user,
          userContext,
          devBypass: isDevBypassEnv(),
        },
        jwt: {
          config: getJWTConfigStatus(),
          testToken: testJWT ? {
            generated: true,
            length: testJWT.length,
            validation: jwtValidation
          } : null,
        },
        environment: {
          nodeEnv: process.env.NODE_ENV,
          hasClerkSecret: !!process.env.CLERK_SECRET_KEY,
          hasJWTPrivateKey: !!process.env.PS_BFF_JWT_PRIVATE_KEY,
          devBypassAuth: process.env.PS_DEV_BYPASS_AUTH,
          gatewayUrl: process.env.PS_GATEWAY_URL,
        },
        timestamp: new Date().toISOString(),
      });
    } catch (error) {
      res.status(500).json({
        error: {
          code: 'DEBUG_ERROR',
          message: 'Failed to generate debug information',
          details: { reason: error instanceof Error ? error.message : String(error) }
        }
      });
    }
  });

  // Debug endpoint for JWT configuration validation
  app.get('/api/debug/jwt-config', async (_req, res) => {
    try {
      const { getJWTConfigStatus } = await import('./jwtAuth.js');
      const configStatus = getJWTConfigStatus();
      
      res.json({
        jwtConfig: configStatus,
        environment: {
          hasPrivateKey: !!process.env.PS_BFF_JWT_PRIVATE_KEY,
          privateKeyLength: process.env.PS_BFF_JWT_PRIVATE_KEY?.length || 0,
          issuer: process.env.PS_BFF_JWT_ISSUER,
          audience: process.env.PS_BFF_JWT_AUDIENCE,
          ttl: process.env.PS_BFF_JWT_TTL,
        },
        timestamp: new Date().toISOString(),
      });
    } catch (error) {
      res.status(500).json({
        error: {
          code: 'JWT_CONFIG_ERROR',
          message: 'Failed to validate JWT configuration',
          details: { reason: error instanceof Error ? error.message : String(error) }
        }
      });
    }
  });

  // Persist active tenant on the server via signed cookie
  app.get('/api/session/tenant', (req: any, res: Response) => {
    const tenantId = (req.signedCookies && req.signedCookies.ps_tenant_id) || req.cookies?.ps_tenant_id;
    const orgId = (req.signedCookies && req.signedCookies.ps_org_id) || req.cookies?.ps_org_id;
    const role = (req.signedCookies && req.signedCookies.ps_tenant_role) || req.cookies?.ps_tenant_role;
    res.json({ tenantId: tenantId || null, orgId: orgId || null, role: role || null });
  });

  app.post('/api/session/tenant', isAuthenticated, (req: any, res: Response) => {
    const { tenantId } = req.body || {};
    if (!tenantId) return res.status(400).json({ message: 'tenantId required' });
    res.cookie('ps_tenant_id', tenantId, {
      httpOnly: true,
      secure: (process.env.SESSION_COOKIE_SECURE || 'true').toLowerCase() !== 'false',
      sameSite: (process.env.SESSION_COOKIE_SAMESITE as any) || 'lax',
      signed: true,
      maxAge: 30 * 24 * 60 * 60 * 1000,
      path: '/',
    });
    res.status(204).end();
  });

  // Clear the signed tenant cookie and any session state
  app.post('/api/session/clear', (_req: any, res: Response) => {
    try { res.clearCookie('ps_tenant_id', { signed: true, path: '/' }); } catch {}
    try { res.clearCookie('ps_tenant_role', { signed: true, path: '/' }); } catch {}
    try { res.clearCookie('ps_org_id', { signed: true, path: '/' }); } catch {}
    res.status(204).end();
  });

  // Auth signout: clear app cookies (Clerk sign-out is handled client-side)
  app.post('/api/auth/signout', isAuthenticated, async (_req: any, res: Response) => {
    try { res.clearCookie('ps_tenant_id', { signed: true, path: '/' }); } catch {}
    try { res.clearCookie('ps_tenant_role', { signed: true, path: '/' }); } catch {}
    try { res.clearCookie('ps_org_id', { signed: true, path: '/' }); } catch {}
    res.status(204).end();
  });

  // Note: Mount the proxy AFTER local /api/* routes so they take precedence

  // Dev toggle route removed for production testing; use environment variables instead.

  // NOTE: /api/onboarding/tenant deprecated — use /api/orgs/create instead.

  // Organizations: list the current user's Clerk organizations
  app.get('/api/orgs', isAuthenticated, async (req: any, res: Response) => {
    try {
      const devBypass = isDevBypassEnv();
      if (devBypass) {
        // Dev: synthesize orgs from env PS_DEV_TENANTS="id1:Name 1,id2:Name 2" or PS_DEV_TENANT_ID
        const raw = process.env.PS_DEV_TENANTS || '';
        let orgs: Array<{ id: string; name: string }>= [];
        if (raw.trim()) {
          orgs = raw.split(',').map(p => {
            const [id, ...rest] = p.split(':');
            return { id: id.trim(), name: rest.join(':').trim() || id.trim() };
          }).filter(o => o.id);
        }
        if (orgs.length === 0) {
          const id = process.env.PS_DEV_TENANT_ID || 'dev-tenant';
          orgs = [{ id, name: 'Dev Tenant' }];
        }
        return res.json({ data: orgs });
      }

      if (!process.env.CLERK_SECRET_KEY) {
        return res.status(500).json({ 
          error: {
            code: 'CONFIGURATION_ERROR',
            message: 'Clerk not configured',
            details: { reason: 'missing_clerk_secret_key' }
          }
        });
      }

      const clerk = createClerkClient({ secretKey: process.env.CLERK_SECRET_KEY! });
      const userId = req.auth?.userId as string;
      if (!userId) {
        return res.status(401).json({ 
          error: {
            code: 'UNAUTHORIZED',
            message: 'User ID not found in authentication context',
            details: { reason: 'missing_user_id' }
          }
        });
      }

      // List memberships, map to { id, name }
      let orgs: Array<{ id: string; name: string }> = [];
      try {
        const mems: any = await (clerk as any).users.getOrganizationMembershipList({ userId });
        orgs = (mems?.data || mems || []).map((m: any) => ({ id: m.organization.id, name: m.organization.name }));
      } catch (clerkError) {
        console.warn('Failed to fetch Clerk organizations:', clerkError);
        orgs = [];
      }

      res.json({ data: orgs });
    } catch (e: any) {
      console.error('Organizations endpoint error:', e);
      res.status(500).json({ 
        error: {
          code: 'INTERNAL_ERROR',
          message: 'Failed to list organizations',
          details: { reason: String(e?.message || e) }
        }
      });
    }
  });

  // Discover organizations by email domain (enterprise-friendly flow)
  app.get('/api/orgs/discover', isAuthenticated, async (req: any, res: Response) => {
    try {
      // Determine email to use for discovery
      let email: string | undefined = String(req.query.email || '').trim() || undefined;
      if (!email) {
        const user = await getUserInfo(req).catch(() => null);
        email = (user?.email || '').toString();
      }
      if (!email || !email.includes('@')) {
        return res.json({ data: [] });
      }
      const domain = email.split('@')[1].toLowerCase();
      const devBypass = isDevBypassEnv();

      // Load mapping from env (JSON string): { "acme.com": [{"id":"org_123","name":"Acme","autoJoin":false}] }
      let mapStr = process.env.PS_DISCOVER_ORGS || process.env.PS_DOMAIN_ORG_MAP || '';
      let list: Array<{ id: string; name: string; autoJoin?: boolean }> = [];
      try {
        if (mapStr.trim()) {
          const parsed = JSON.parse(mapStr);
          const arr = parsed?.[domain];
          if (Array.isArray(arr)) list = arr.map((o: any) => ({ id: String(o.id), name: String(o.name || o.id), autoJoin: Boolean(o.autoJoin) }));
        }
      } catch (_) {
        list = [];
      }

      // In dev bypass without explicit mapping, optionally synthesize a demo org
      if (devBypass && list.length === 0) {
        list = [{ id: 'dev-demo-org', name: `${domain} (Demo)`, autoJoin: false }];
      }

      return res.json({ data: list });
    } catch (e: any) {
      res.status(500).json({ message: 'failed to discover organizations', details: String(e?.message || e) });
    }
  });

  // Create a new Clerk organization for the current user
  app.post('/api/orgs/create', isAuthenticated, async (req: any, res: Response) => {
    try {
      const { name } = req.body || {};
      if (!name || !name.trim()) return res.status(400).json({ message: 'name is required' });
      if (!allowSelfTenantSignup()) {
        return res.status(403).json({ code: 'TENANT_SIGNUP_DISABLED', message: 'Self-serve tenant signup is disabled' });
      }
      const devBypass = isDevBypassEnv();
      let orgId: string; let orgName = name.trim();
      if (devBypass) {
        orgId = process.env.PS_DEV_TENANT_ID || `dev-${Date.now()}`;
      } else {
        if (!process.env.CLERK_SECRET_KEY) return res.status(500).json({ message: 'clerk not configured' });
        const clerk = createClerkClient({ secretKey: process.env.CLERK_SECRET_KEY! });
        const userId = req.auth?.userId as string;
        const org = await clerk.organizations.createOrganization({ name: orgName, createdBy: userId });
        orgId = org.id; orgName = org.name;
      }
      // Persist org id cookie
      res.cookie('ps_org_id', orgId, {
        httpOnly: true,
        secure: (process.env.SESSION_COOKIE_SECURE || 'true').toLowerCase() !== 'false',
        sameSite: (process.env.SESSION_COOKIE_SAMESITE as any) || 'lax',
        signed: true,
        maxAge: 30 * 24 * 60 * 60 * 1000,
        path: '/',
      });
      // Ensure backend tenant exists and set cookies
      let tenantId: string = orgId;
      try {
        const userCtx = extractUserContext(req);
        const token = generateGatewayJWT(userCtx);
        const resp = await fetch(`${process.env.PS_GATEWAY_URL || 'http://localhost:8098'}/v1/admin/tenants`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
            ...(process.env.PS_ADMIN_TOKEN ? { 'X-PS-Admin-Token': process.env.PS_ADMIN_TOKEN } : {}),
          },
          body: JSON.stringify({ name: orgName }),
        });
        if (resp.ok) {
          const tenant = await resp.json(); tenantId = tenant.id;
        }
      } catch {}
      // Set tenant cookies and mark admin role
      res.cookie('ps_tenant_id', tenantId, {
        httpOnly: true,
        secure: (process.env.SESSION_COOKIE_SECURE || 'true').toLowerCase() !== 'false',
        sameSite: (process.env.SESSION_COOKIE_SAMESITE as any) || 'lax',
        signed: true,
        maxAge: 30 * 24 * 60 * 60 * 1000,
        path: '/',
      });
      res.cookie('ps_tenant_role', 'admin', {
        httpOnly: true,
        secure: (process.env.SESSION_COOKIE_SECURE || 'true').toLowerCase() !== 'false',
        sameSite: (process.env.SESSION_COOKIE_SAMESITE as any) || 'lax',
        signed: true,
        maxAge: 30 * 24 * 60 * 60 * 1000,
        path: '/',
      });
      // Upsert backend membership as owner/admin
      try {
        const userCtx = extractUserContext(req);
        const token = generateGatewayJWT(userCtx);
        await fetch(`${process.env.PS_GATEWAY_URL || 'http://localhost:8098'}/v1/tenants/${tenantId}/memberships/self`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
          },
        });
      } catch {}
      res.status(201).json({ id: orgId, name: orgName, tenant_id: tenantId });
    } catch (e: any) {
      res.status(500).json({ message: 'failed to create organization', details: String(e?.message || e) });
    }
  });

  // Issue invitation to join an organization (Clerk)
  // Basic rate-limit for invites: per-user window
  const inviteCounts: Map<string, { count: number; resetAt: number }> = new Map();
  const maxInvites = parseInt(process.env.PS_INVITE_RATE_LIMIT || '5', 10);
  const windowMs = parseInt(process.env.PS_INVITE_WINDOW_MS || String(10 * 60 * 1000), 10); // 10m default
  const rateLimitInvite = (userId: string) => {
    const now = Date.now();
    const e = inviteCounts.get(userId);
    if (!e || now >= e.resetAt) {
      inviteCounts.set(userId, { count: 1, resetAt: now + windowMs });
      return { allowed: true, retryAfter: 0 };
    }
    if (e.count < maxInvites) {
      e.count += 1; inviteCounts.set(userId, e); return { allowed: true, retryAfter: 0 };
    }
    return { allowed: false, retryAfter: Math.ceil((e.resetAt - now) / 1000) };
  };

  const verifyCaptcha = (req: any): boolean => {
    const tok = (req.headers['x-ps-captcha'] as string) || (req.body && req.body.captchaToken);
    const secret = process.env.PS_CAPTCHA_TOKEN || '';
    const devBypass = isDevBypassEnv();
    if (secret) return tok === secret;
    if (devBypass) return tok === 'approved';
    return false; // require secret in non-dev
  };

  app.post('/api/orgs/invite', isAuthenticated, async (req: any, res: Response) => {
    try {
      const { orgId, email, role } = req.body || {};
      if (!orgId || !email) return res.status(400).json({ message: 'orgId and email required' });
      // CAPTCHA / anti-abuse
      if (!verifyCaptcha(req)) {
        return res.status(403).json({ code: 'CAPTCHA_REQUIRED', message: 'captcha verification failed' });
      }
      // Rate limit per user
      const userId = req.auth?.userId as string;
      const rl = rateLimitInvite(userId || 'anon');
      if (!rl.allowed) {
        res.setHeader('Retry-After', String(rl.retryAfter));
        return res.status(429).json({ code: 'RATE_LIMIT', message: 'Too many invites, slow down' });
      }
      const devBypass = isDevBypassEnv();
      if (devBypass) {
        // No-op in dev bypass
        return res.status(200).json({ invited: true, email, orgId, role: role || 'basic_member' });
      }
      if (!process.env.CLERK_SECRET_KEY) return res.status(500).json({ message: 'clerk not configured' });
      const clerk = createClerkClient({ secretKey: process.env.CLERK_SECRET_KEY! });
      // Authorization: caller must be admin/owner of org
      const mems: any = await (clerk as any).users.getOrganizationMembershipList({ userId });
      const memRec = (mems?.data || mems || []).find((m: any) => m.organization.id === orgId);
      if (!memRec) return res.status(403).json({ message: 'Not a member of this organization' });
      const roleStr = String(memRec?.role || '').toLowerCase();
      if (!(roleStr === 'admin' || roleStr === 'owner')) {
        return res.status(403).json({ message: 'Only organization admins can invite' });
      }
      // Create invitation (role defaults)
      const invRole = role || 'basic_member';
      const inv = await (clerk as any).organizations.createOrganizationInvitation({ organizationId: orgId, inviterUserId: userId, emailAddress: email, role: invRole });
      res.status(200).json({ invited: true, invitation: inv });
    } catch (e: any) {
      res.status(500).json({ message: 'failed to invite', details: String(e?.message || e) });
    }
  });

  // List organization members (requires membership)
  app.get('/api/orgs/members', isAuthenticated, async (req: any, res: Response) => {
    try {
      const devBypass = isDevBypassEnv();
      const orgId = (req.signedCookies && req.signedCookies.ps_org_id) || req.cookies?.ps_org_id;
      if (!orgId) return res.status(400).json({ message: 'No organization selected' });
      if (devBypass) {
        return res.json({ data: [{ id: 'dev-user', email: 'dev@example.com', role: 'owner' }] });
      }
      if (!process.env.CLERK_SECRET_KEY) return res.status(500).json({ message: 'clerk not configured' });
      const clerk = createClerkClient({ secretKey: process.env.CLERK_SECRET_KEY! });
      // We attempt org membership list; fallback to empty
      try {
        const list: any = await (clerk as any).organizations.getOrganizationMembershipList({ organizationId: orgId });
        const members = (list?.data || list || []).map((m: any) => ({ id: m.public_user_data?.user_id || m.id, email: m.public_user_data?.identifier || m.email_address, role: m.role }));
        return res.json({ data: members });
      } catch (_) {
        return res.json({ data: [] });
      }
    } catch (e: any) {
      res.status(500).json({ message: 'failed to load members', details: String(e?.message || e) });
    }
  });

  // List organization invitations (admin only)
  app.get('/api/orgs/invitations', isAuthenticated, async (req: any, res: Response) => {
    try {
      const devBypass = isDevBypassEnv();
      const orgId = (req.signedCookies && req.signedCookies.ps_org_id) || req.cookies?.ps_org_id;
      if (!orgId) return res.status(400).json({ message: 'No organization selected' });
      if (devBypass) {
        return res.json({ data: [] });
      }
      if (!process.env.CLERK_SECRET_KEY) return res.status(500).json({ message: 'clerk not configured' });
      const clerk = createClerkClient({ secretKey: process.env.CLERK_SECRET_KEY! });
      // Ensure caller is admin
      const userId = req.auth?.userId as string;
      const mems: any = await (clerk as any).users.getOrganizationMembershipList({ userId });
      const memRec = (mems?.data || mems || []).find((m: any) => m.organization.id === orgId);
      const roleStr = String(memRec?.role || '').toLowerCase();
      if (!(roleStr === 'admin' || roleStr === 'owner')) {
        return res.status(403).json({ message: 'Only organization admins can view invites' });
      }
      try {
        const invs: any = await (clerk as any).organizations.getOrganizationInvitationList({ organizationId: orgId });
        const items = (invs?.data || invs || []).map((v: any) => ({ id: v.id, email: v.email_address, role: v.role, status: v.status, created_at: v.created_at }));
        return res.json({ data: items });
      } catch (_) {
        return res.json({ data: [] });
      }
    } catch (e: any) {
      res.status(500).json({ message: 'failed to load invitations', details: String(e?.message || e) });
    }
  });

  // Select an organization: ensure backend tenant exists and persist tenant cookie
  app.post('/api/orgs/select', isAuthenticated, async (req: any, res: Response) => {
    try {
      const { orgId } = req.body || {};
      if (!orgId) return res.status(400).json({ message: 'orgId required' });
      const devBypass = isDevBypassEnv();
      let orgName: string | undefined = undefined;
      let membershipRole: string | undefined = undefined;
      if (!devBypass) {
        if (!process.env.CLERK_SECRET_KEY) return res.status(500).json({ message: 'clerk not configured' });
        const clerk = createClerkClient({ secretKey: process.env.CLERK_SECRET_KEY! });
        // Verify user is a member of this Clerk organization
        const userId = req.auth?.userId as string;
        const mems: any = await (clerk as any).users.getOrganizationMembershipList({ userId });
        const memberRec = (mems?.data || mems || []).find((m: any) => m.organization.id === orgId);
        const member = !!memberRec;
        if (!member) {
          return res.status(403).json({ message: 'Not a member of this organization' });
        }
        membershipRole = (memberRec?.role || '').toString();
        const org = await clerk.organizations.getOrganization({ organizationId: orgId });
        orgName = org.name;
      }

      // Ensure a backend tenant exists and link mapping (idempotent)
      let tenantId: string | null = null;
      if (!devBypass) {
        const userCtx = extractUserContext(req);
        const token = generateGatewayJWT(userCtx);
        const resp = await fetch(`${process.env.PS_GATEWAY_URL || 'http://localhost:8098'}/v1/tenants/resolve`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
          },
          body: JSON.stringify({ provider: 'clerk', external_org_id: orgId, fallback_name: orgName || orgId })
        });
        if (resp.ok) {
          const data = await resp.json();
          tenantId = data?.tenant_id || null;
        }
      }

      // Persist selection cookie from BFF side regardless
      if (!tenantId) {
        // Fallback to set cookie with an org-derived token to avoid blocking; frontend will continue
        // In production, add a /v1/tenants/find?name= endpoint to resolve ID here
      }
      res.cookie('ps_tenant_id', tenantId || orgId || '', {
        httpOnly: true,
        secure: (process.env.SESSION_COOKIE_SECURE || 'true').toLowerCase() !== 'false',
        sameSite: (process.env.SESSION_COOKIE_SAMESITE as any) || 'lax',
        signed: true,
        maxAge: 30 * 24 * 60 * 60 * 1000,
        path: '/',
      });
      // Persist Clerk org id cookie
      res.cookie('ps_org_id', orgId, {
        httpOnly: true,
        secure: (process.env.SESSION_COOKIE_SECURE || 'true').toLowerCase() !== 'false',
        sameSite: (process.env.SESSION_COOKIE_SAMESITE as any) || 'lax',
        signed: true,
        maxAge: 30 * 24 * 60 * 60 * 1000,
        path: '/',
      });
      // Persist tenant role (admin vs member) as a signed cookie for JWT claims
      const roleVal = (membershipRole || '').toLowerCase();
      const isAdminRole = roleVal === 'admin' || roleVal === 'owner';
      res.cookie('ps_tenant_role', isAdminRole ? 'admin' : 'member', {
        httpOnly: true,
        secure: (process.env.SESSION_COOKIE_SECURE || 'true').toLowerCase() !== 'false',
        sameSite: (process.env.SESSION_COOKIE_SAMESITE as any) || 'lax',
        signed: true,
        maxAge: 30 * 24 * 60 * 60 * 1000,
        path: '/',
      });
      // Upsert backend membership for the current user (self-membership)
      try {
        const userCtx = extractUserContext(req);
        const token = generateGatewayJWT(userCtx);
        if (tenantId) {
          await fetch(`${process.env.PS_GATEWAY_URL || 'http://localhost:8098'}/v1/tenants/${tenantId}/memberships/self`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${token}`,
            },
          });
        }
      } catch (_) { /* ignore */ }
      res.status(204).end();
    } catch (e: any) {
      res.status(500).json({ message: 'failed to select organization', details: String(e?.message || e) });
    }
  });

  // Preset import: baseline compliance coverage
  app.post('/api/presets/import/baseline', isAuthenticated, async (req: any, res: Response) => {
    try {
      const fs = await import('fs');
      const path = await import('path');
      const filePath = path.resolve(process.cwd(), 'presets', 'preset-compliance-coverage.yaml');
      if (!fs.existsSync(filePath)) {
        return res.status(404).json({ error: { code: 'PRESET_NOT_FOUND', message: 'Preset file not found' } });
      }
      const yaml = fs.readFileSync(filePath, 'utf8');

      const gw = process.env.PS_GATEWAY_URL || 'http://localhost:8098';
      const userCtx = extractUserContext(req);
      const token = generateGatewayJWT(userCtx);
      const tenantId = (req.signedCookies && req.signedCookies.ps_tenant_id) || req.cookies?.ps_tenant_id || userCtx.tenantId;
      const headers: any = {
        'Content-Type': 'text/yaml',
        'Authorization': `Bearer ${token}`,
        ...(tenantId ? { 'X-PS-Tenant-ID': tenantId } : {}),
      };

      const postTo = async (p: string) => {
        const r = await fetch(`${gw}${p}`, { method: 'POST', headers, body: yaml as any });
        return r;
      };

      let resp = await postTo('/rulepacks?activate=true');
      if (resp.status === 404) resp = await postTo('/v1/rulepacks?activate=true');

      const contentType = resp.headers.get('content-type') || '';
      const data = contentType.includes('application/json') ? await resp.json() : await resp.text();
      return res.status(resp.status).send(data);
    } catch (e: any) {
      return res.status(500).json({ error: { code: 'PRESET_IMPORT_FAILED', message: 'Failed to import preset', details: String(e?.message || e) } });
    }
  });

  // Back-compat for any callers using /api/proxy/*
  app.use('/api/proxy', createGatewayProxy('/api/proxy'));

  // Tool execution: proxy to Go endpoint for centralized execution + audit
  app.post('/api/tools/exec', isAuthenticated, async (req: any, res: Response) => {
    try {
      const gw = process.env.PS_GATEWAY_URL || 'http://localhost:8098';
      const userCtx = extractUserContext(req);
      const token = generateGatewayJWT(userCtx);

      // Normalize request body to gateway shape
      const b = req.body || {};
      const payload = {
        tool_id: String(b.tool_id || b.tool || '').trim(),
        args: b.args ?? {},
        conversation_id: b.conversation_id,
        request_id: b.request_id,
        endpoint: b.endpoint,
        method: b.method,
        lane: b.lane,
        plan_hash: b.plan_hash,
        plan_step: b.plan_step,
        timeout_ms: b.timeout_ms,
      };

      // Prefer tenant from signed cookie
      const tenantId = (req.signedCookies && req.signedCookies.ps_tenant_id) || req.cookies?.ps_tenant_id || userCtx.tenantId;
      const commonHeaders: any = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`,
        ...(tenantId ? { 'X-PS-Tenant-ID': tenantId } : {}),
        // Forward correlation headers when present
        ...(req.headers['x-request-id'] ? { 'X-Request-ID': req.headers['x-request-id'] as string } : {}),
        ...(req.headers['x-ps-endpoint'] ? { 'X-PS-Endpoint': req.headers['x-ps-endpoint'] as string } : {}),
        ...(req.headers['x-ps-method'] ? { 'X-PS-Method': req.headers['x-ps-method'] as string } : {}),
        ...(req.headers['x-ps-lane'] ? { 'X-PS-Lane': req.headers['x-ps-lane'] as string } : {}),
        ...(req.headers['x-ps-plan-hash'] ? { 'X-PS-Plan-Hash': req.headers['x-ps-plan-hash'] as string } : {}),
        ...(req.headers['x-ps-plan-step'] ? { 'X-PS-Plan-Step': req.headers['x-ps-plan-step'] as string } : {}),
        ...(req.headers['x-ps-approval-token'] ? { 'X-PS-Approval-Token': req.headers['x-ps-approval-token'] as string } : {}),
      };

      // Try root-mounted API first, then fallback to /v1 prefix
      const tryExec = async (path: string) => {
        const resp = await fetch(`${gw}${path}`, {
          method: 'POST',
          headers: commonHeaders,
          body: JSON.stringify(payload),
        });
        return resp;
      };

      let resp = await tryExec('/api/tools/exec');
      if (resp.status === 404) {
        resp = await tryExec('/v1/api/tools/exec');
      }

      const contentType = resp.headers.get('content-type') || '';
      const data = contentType.includes('application/json') ? await resp.json() : await resp.text();
      res.status(resp.status).send(data);
    } catch (e: any) {
      return res.status(500).json({ message: 'tool exec proxy failed', details: String(e?.message || e) });
    }
  });

  // Finally, mount catch-all gateway proxy for remaining /api/* requests
  app.use('/api', createGatewayProxy('/api'));

  const httpServer = createServer(app);
  return httpServer;
}
