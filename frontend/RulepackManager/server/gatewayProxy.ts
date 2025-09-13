import express from 'express';
import { generateGatewayJWT, extractUserContext } from './jwtAuth.js';

const GATEWAY_URL = process.env.PS_GATEWAY_URL || 'http://localhost:8098';

/**
 * Create a proxy middleware that forwards requests to the Go gateway with JWT auth
 *
 * If mounted at `mountPath` (e.g. `/api`), the proxy strips that prefix when
 * constructing the gateway request path so `/api/v1/...` -> `/v1/...`.
 */
export function createGatewayProxy(mountPath?: string) {
  const router = express.Router();
  const devBypass = (process.env.PS_DEV_BYPASS_AUTH || '').toLowerCase() === 'true';

  // Middleware to ensure user is authenticated (Clerk)
  const requireAuth = (req: any, res: any, next: any) => {
    if (!req?.auth?.userId) {
      return res.status(401).json({ error: 'Authentication required' });
    }
    next();
  };

  // Middleware to ensure tenant context is selected for non-admin routes
  const requireTenantContext = (req: any, res: any, next: any) => {
    if (devBypass) return next();
    const path = req.originalUrl || req.path || '';
    // Allow admin ops and public health
    if (path.startsWith('/api/admin') || path.endsWith('/healthz') || path.endsWith('/readyz')) return next();
    // Allow auth/org management endpoints
    if (path.startsWith('/api/orgs') || path.startsWith('/api/session')) return next();
    // Admin role from cookie
    const role = (req.signedCookies && req.signedCookies.ps_tenant_role) || (req.cookies && req.cookies.ps_tenant_role);
    const isAdmin = String(role || '').toLowerCase() === 'admin';
    if (isAdmin) return next();
    const tenantId = (req.signedCookies && req.signedCookies.ps_tenant_id) || (req.cookies && req.cookies.ps_tenant_id);
    if (!tenantId) {
      return res.status(428).json({ code: 'TENANT_REQUIRED', message: 'Select an organization before continuing' });
    }
    next();
  };

  /**
   * Generic proxy function that forwards requests to the gateway
   */
  const proxyToGateway = async (req: any, res: any) => {
    try {
      // Always attach a gateway JWT (even in dev bypass)
      const userContext = extractUserContext(req);
      const token: string = generateGatewayJWT(userContext);

      // Build target URL
      let targetPath = req.originalUrl;
      if (mountPath && targetPath.startsWith(mountPath)) {
        targetPath = targetPath.slice(mountPath.length) || '/';
      }
      const targetUrl = `${GATEWAY_URL}${targetPath}`;

      // Build headers: strip any inbound Authorization, always send our JWT
      const fwdHeaders: Record<string, string> = {
        'Content-Type': 'application/json',
      };
      for (const [k, v] of Object.entries(req.headers)) {
        const key = k.toLowerCase();
        if (key === 'authorization') continue;
        // Do not forward any client-provided identity or tenant hints
        if (key === 'x-tenant-id' || key === 'x-ps-tenant-id' || key === 'x-ps-frontend-auth' || key === 'x-ps-user-id' || key === 'x-ps-user-name') continue;
        if (typeof v === 'string') fwdHeaders[k] = v;
      }
      // Prefer tenant from signed cookie for RLS
      try {
        const tenantId = (req.signedCookies && req.signedCookies.ps_tenant_id) || (req.cookies && req.cookies.ps_tenant_id);
        if (tenantId) fwdHeaders['X-PS-Tenant-ID'] = tenantId as string;
      } catch {}
      if (token) {
        fwdHeaders['Authorization'] = `Bearer ${token}`;
      }
      if ((process.env.PS_ENABLE_ADMIN_TOKEN_FORWARD || '').toLowerCase() === 'true' && process.env.PS_ADMIN_TOKEN) {
        if ((process.env.NODE_ENV || '').toLowerCase() === 'production') {
          console.warn('[SECURITY] Admin token forward is enabled in production. This is not recommended.');
        }
        fwdHeaders['X-PS-Admin-Token'] = process.env.PS_ADMIN_TOKEN;
      }

      const response = await fetch(targetUrl, {
        method: req.method,
        headers: fwdHeaders,
        body: req.method !== 'GET' && req.method !== 'HEAD' ? JSON.stringify(req.body) : undefined,
      });

      // Forward response status
      res.status(response.status);

      // Handle different content types
      const contentType = response.headers.get('content-type');
      if (contentType?.includes('application/json')) {
        const data = await response.json();
        res.json(data);
      } else {
        const text = await response.text();
        res.send(text);
      }
    } catch (error) {
      // Gateway proxy error handled by returning 500 response
      res.status(500).json({ 
        error: 'Gateway request failed',
        details: error instanceof Error ? error.message : 'Unknown error'
      });
    }
  };

  // Health endpoints (public, no auth required)
  router.get('/healthz', async (_req, res) => {
    try {
      const response = await fetch(`${GATEWAY_URL}/healthz`);
      const text = await response.text();
      res.status(response.status).send(text);
    } catch (error) {
      res.status(500).send('Gateway health check failed');
    }
  });

  router.get('/readyz', async (_req, res) => {
    try {
      const response = await fetch(`${GATEWAY_URL}/readyz`);
      const text = await response.text();
      res.status(response.status).send(text);
    } catch (error) {
      res.status(500).send('Gateway readiness check failed');
    }
  });

  // In dev-bypass, do not require auth/tenant context for any proxied endpoints
  if (!devBypass) {
    // Protected endpoints that require authentication
    router.use('/rulepacks', requireAuth, requireTenantContext);
    router.use('/admin', requireAuth);
    router.use('/api/v1', requireAuth, requireTenantContext);
    // Proxy all requests to the gateway with guards
    router.all('*', requireAuth, requireTenantContext, proxyToGateway);
  } else {
    // Dev-bypass: proxy everything without guards
    router.all('*', proxyToGateway);
  }

  return router;
}
