import { http, HttpResponse } from 'msw';

// Mock data
const mockTenants = [
  {
    id: '6f4d338d-f0c0-4091-b54e-f71752c8f568',
    name: 'Dev Tenant',
    status: 'active',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

const mockRulePacks = [
  {
    id: '550e8400-e29b-41d4-a716-446655440001',
    name: 'Security Policy',
    description: 'Basic security rules',
    currentVersionId: 'v1',
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  },
];

const mockAssignments = [
  {
    id: '123e4567-e89b-12d3-a456-426614174000',
    rulepackId: '550e8400-e29b-41d4-a716-446655440001',
    targetScope: '/api/v1/users',
    endpoints: ['/api/v1/users', '/api/v1/admin'],
    priority: 'high',
    enabled: true,
    createdAt: '2024-01-01T00:00:00Z',
    updatedAt: '2024-01-01T00:00:00Z',
  },
];

const mockAuditEvents = [
  {
    id: 'audit-1',
    action: 'violation',
    objectType: 'request',
    objectId: 'req-123',
    metadata: {
      decision: 'deny',
      violations: ['injection_detected'],
      path: '/api/v1/users',
      status: 'blocked',
    },
    timestamp: '2024-01-01T00:00:00Z',
  },
  {
    id: 'audit-2',
    action: 'request',
    objectType: 'request',
    objectId: 'req-124',
    metadata: {
      decision: 'allow',
      path: '/api/v1/health',
      status: 'allowed',
    },
    timestamp: '2024-01-01T00:01:00Z',
  },
];

export const handlers = [
  // Allow local supertest/Express requests to pass through in server tests
  http.all('*', ({ request }) => {
    try {
      const url = new URL(request.url);
      const host = (url.hostname || '').toLowerCase();
      if (host === '127.0.0.1' || host === 'localhost') {
        return HttpResponse.passthrough();
      }
    } catch {}
    // Fallthrough to specific mocks below; the catch-all at the bottom will handle others
    return HttpResponse.passthrough();
  }),
  // Health endpoints
  http.get('/api/healthz', () => {
    // Some environments return plain text; emulate object for tests
    return HttpResponse.json({ status: 'ok', timestamp: new Date().toISOString() });
  }),

  http.get('/api/readyz', () => {
    return HttpResponse.json({ status: 'ready', checks: { database: true, auth: true, storage: true }, gateway: true, timestamp: new Date().toISOString() });
  }),

  // Auth endpoints
  http.get('/api/auth/user', () => {
    return HttpResponse.json({
      id: 'dev-user',
      email: 'dev@example.com',
      name: 'Dev User',
      systemRole: 'user',
    });
  }),

  // Tenant endpoints
  http.get('/api/v1/tenants/my', () => {
    return HttpResponse.json({
      tenants: mockTenants,
      count: mockTenants.length,
    });
  }),

  http.post('/api/v1/admin/tenants', async ({ request }) => {
    const body: any = await request.json().catch(() => ({}));
    if (!body?.name) {
      return HttpResponse.json(
        { error: 'INVALID_ARGUMENT', message: 'Tenant name is required' },
        { status: 400 }
      );
    }
    
    const newTenant = {
      id: 'new-tenant-id',
      name: body?.name,
      status: 'active',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };
    
    return HttpResponse.json(newTenant, { status: 201 });
  }),

  // RulePack endpoints
  http.get('/api/rulepacks', () => {
    return HttpResponse.json({
      data: mockRulePacks,
      total: mockRulePacks.length,
    });
  }),

  http.post('/api/rulepacks', async ({ request }) => {
    const body: any = await request.json().catch(() => ({}));
    if (!body?.name) {
      return HttpResponse.json(
        { error: 'INVALID_ARGUMENT', message: 'RulePack name is required' },
        { status: 400 }
      );
    }
    
    const newRulePack = {
      id: 'new-rulepack-id',
      name: body?.name,
      description: body?.description || '',
      currentVersionId: 'v1',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    
    return HttpResponse.json(newRulePack, { status: 201 });
  }),

  // Assignment endpoints
  http.get('/api/v1/policy-assignments', () => {
    return HttpResponse.json({
      data: mockAssignments,
      total: mockAssignments.length,
    });
  }),

  http.post('/api/v1/policy-assignments', async ({ request }) => {
    const body: any = await request.json().catch(() => ({}));
    if (!body?.rulepackId || !body?.endpoints || !body?.priority) {
      return HttpResponse.json(
        { error: 'INVALID_ARGUMENT', message: 'Missing required fields' },
        { status: 400 }
      );
    }
    
    const newAssignment = {
      id: 'new-assignment-id',
      rulepackId: body?.rulepackId,
      targetScope: Array.isArray(body?.endpoints) ? body.endpoints[0] : '*',
      endpoints: Array.isArray(body?.endpoints) ? body.endpoints : ['*'],
      priority: body?.priority,
      enabled: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
    
    return HttpResponse.json(newAssignment, { status: 201 });
  }),

  // Audit endpoints
  http.get('/api/v1/audit/violations', ({ request }) => {
    const url = new URL(request.url);
    const timeRange = url.searchParams.get('timeRange') || '24h';
    const limit = parseInt(url.searchParams.get('limit') || '100');
    const offset = parseInt(url.searchParams.get('offset') || '0');
    
    const filteredEvents = mockAuditEvents.slice(offset, offset + limit);
    
    return HttpResponse.json({
      data: filteredEvents,
      total: mockAuditEvents.length,
      timeRange,
      limit,
      offset,
    });
  }),

  http.post('/api/admin/audits/search', async ({ request }) => {
    const body: any = await request.json().catch(() => ({}));
    const { timeRange = '24h', limit = 100, offset = 0 } = body || {};
    
    const filteredEvents = mockAuditEvents.slice(offset, offset + limit);
    
    return HttpResponse.json({
      data: filteredEvents,
      total: mockAuditEvents.length,
      timeRange,
      limit,
      offset,
    });
  }),

  // Session endpoints
  http.get('/api/session/tenant', () => {
    return HttpResponse.json({ tenantId: '6f4d338d-f0c0-4091-b54e-f71752c8f568' });
  }),

  http.post('/api/session/tenant', async ({ request }) => {
    const body: any = await request.json().catch(() => ({}));
    if (!body?.tenantId) {
      return HttpResponse.json(
        { error: 'INVALID_ARGUMENT', message: 'tenantId required' },
        { status: 400 }
      );
    }
    return new HttpResponse(null, { status: 204 });
  }),

  // Catch-all for unhandled requests
  http.all('*', ({ request }) => {
    console.warn(`Unhandled request: ${request.method} ${request.url}`);
    return HttpResponse.json(
      { error: 'NOT_FOUND', message: 'Endpoint not found' },
      { status: 404 }
    );
  }),
];
