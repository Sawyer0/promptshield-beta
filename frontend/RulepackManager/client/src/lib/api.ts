import { queryClient } from "@/lib/queryClient";
import { apiRequest } from "@/lib/queryClient";
import { useToast } from "@/hooks/use-toast";
import { isUnauthorizedError } from "@/lib/authUtils";
import { API_CONFIG } from "@/config/api";
import type {
  RulePack,
  InsertRulePack,
  Tenant,
  InsertTenant,
  PolicyAssignment,
  InsertPolicyAssignment,
  APIResponse,
} from "@shared/apiTypes";

// Audit event interface matching your backend structure
interface AuditEvent {
  id: string;
  tenant_id: string;
  action: string;
  object_type: string;
  object_id: string;
  actor_id: string;
  actor_email: string;
  metadata: {
    path?: string;
    decision?: 'allow' | 'deny' | 'quarantine';
    reason?: string;
    violations?: number;
    status?: number;
    [key: string]: any;
  };
  created_at: string;
}

// API base configuration - use config for proper port
const API_BASE = API_CONFIG.BASE_URL;

// Headers for your backend with frontend bypass authentication
const getHeaders = (userContext?: { userId?: string; userName?: string; tenantId?: string }) => ({
  'Content-Type': 'application/json',
  'X-PS-Frontend-Auth': API_CONFIG.FRONTEND_AUTH_TOKEN,
  'X-Tenant-ID': userContext?.tenantId || '6f4d338d-f0c0-4091-b54e-f71752c8f568',
  'X-PS-User-ID': userContext?.userId || '',
  'X-PS-User-Name': userContext?.userName || '',
});

// Error handling helper
export const handleApiError = (error: Error, toast: ReturnType<typeof useToast>['toast']) => {
  if (isUnauthorizedError(error)) {
    toast({
      title: "Unauthorized",
      description: "You are logged out. Logging in again...",
      variant: "destructive",
    });
    setTimeout(() => {
      window.location.href = "/landing";
    }, 500);
    return;
  }

  toast({
    title: "Error",
    description: error.message || "An unexpected error occurred",
    variant: "destructive",
  });
};

// Response handling helper
const handleResponse = async (response: Response) => {
  if (!response.ok) {
    throw new Error(`${response.status}: ${response.statusText}`);
  }
  return response.json();
};

// Utility: derive tenant id from localStorage or env fallback
const getActiveTenantId = (): string | null => {
  let id: string | null = null;
  if (typeof document !== 'undefined') {
    try {
      const m = document.cookie.match(/(?:^|; )ps_tenant_id=([^;]+)/);
      if (m && m[1]) {
        id = decodeURIComponent(m[1]);
      }
    } catch {}
  }
  if (!id && typeof window !== 'undefined') {
    id = localStorage.getItem('promptshield_tenant_id') || localStorage.getItem('ps_dev_tenant') || localStorage.getItem('tenant_id');
  }
  // Validate UUID v4-ish; if not valid, use known dev UUID so gateway endpoints accept format
  const uuidRe = /^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$/;
  if (!id || !uuidRe.test(id)) {
    return '6f4d338d-f0c0-4091-b54e-f71752c8f568';
  }
  return id;
};

// RulePack API functions - normalized to shared RulePack shape
export const rulePackApi = {
  // Core CRUD Operations
  getAll: async (_userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<RulePack[]>> => {
    // BFF proxy to gateway: GET /api/rulepacks -> returns array or { data: [] }
    const res = await apiRequest('GET', `/api/rulepacks`);
    const json = await res.json();
    const list = Array.isArray(json) ? json : (Array.isArray(json?.data) ? json.data : []);
    const data: RulePack[] = (list || []).map((m: any) => ({
      id: m.id,
      name: m.name,
      description: m.description ?? '',
      currentVersionId: m.version ?? m.currentVersionId ?? null,
      isActive: !!(m.active ?? m.isActive),
      metadata: m.metadata ?? { source: m.source },
      rules: m.rules,
      tenantId: m.tenantId ?? null,
      createdAt: m.createdAt ?? m.created_at,
      updatedAt: m.updatedAt ?? m.updated_at,
    }));
    return { success: true, data } as any;
  },

  get: async (id: string, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<RulePack>> => {
    const response = await fetch(`${API_BASE}/rulepacks/${id}`, {
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    const m = await response.json();
    const data: RulePack = {
      id: m.id,
      name: m.name,
      description: m.description ?? '',
      currentVersionId: m.version ?? m.currentVersionId ?? null,
      isActive: !!(m.active ?? m.isActive),
      metadata: m.metadata ?? { source: m.source },
      rules: m.rules,
      tenantId: m.tenantId ?? null,
      createdAt: m.createdAt ?? m.created_at,
      updatedAt: m.updatedAt ?? m.updated_at,
    } as any;
    return { success: true, data } as any;
  },

  create: async (dsl: any, _userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<RulePack>> => {
    // Create + activate via gateway through BFF: POST /api/rulepacks?activate=true -> gateway /rulepacks?activate=true
    const response = await apiRequest('POST', `/api/rulepacks?activate=true`, dsl);
    const meta = await response.json();
    const data: RulePack = {
      id: meta.id,
      name: meta.name,
      description: meta.description ?? '',
      currentVersionId: meta.version ?? meta.currentVersionId ?? null,
      isActive: !!(meta.active ?? meta.isActive),
      metadata: meta.metadata ?? { source: meta.source },
    } as any;
    return { success: true, data } as any;
  },

  delete: async (id: string, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<void>> => {
    const response = await fetch(`${API_BASE}/rulepacks/${id}`, {
      method: 'DELETE',
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  // Version Management
  createVersion: async (id: string, data: { dsl: string; status: 'draft' | 'approved' }, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<any>> => {
    const response = await fetch(`${API_BASE}/rulepacks/${id}/versions`, {
      method: 'POST',
      headers: getHeaders(userContext),
      credentials: 'include',
      body: JSON.stringify(data)
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  getVersion: async (id: string, versionNumber: string, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<any>> => {
    const response = await fetch(`${API_BASE}/rulepacks/${id}/versions/${versionNumber}`, {
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  listVersions: async (id: string, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<any[]>> => {
    const response = await fetch(`${API_BASE}/rulepacks/${id}/versions`, {
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  // Activation Control
  activateVersion: async (id: string, versionId: string, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<any>> => {
    const response = await fetch(`${API_BASE}/rulepacks/${id}/versions/${versionId}/activate`, {
      method: 'POST',
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  activateLatest: async (id: string, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<any>> => {
    const response = await fetch(`${API_BASE}/rulepacks/${id}/activate-latest`, {
      method: 'POST',
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  deactivate: async (id: string, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<any>> => {
    const response = await fetch(`${API_BASE}/rulepacks/${id}/deactivate`, {
      method: 'POST',
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  // Management Operations
  purgeVersions: async (id: string, keep: number, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<any>> => {
    const response = await fetch(`${API_BASE}/rulepacks/${id}/purge-versions`, {
      method: 'POST',
      headers: getHeaders(userContext),
      credentials: 'include',
      body: JSON.stringify({ keep })
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },
};


// Tools Registry API
export const toolsApi = {
  list: async (params?: { offset?: number; limit?: number }): Promise<{ data: any[]; total: number; offset: number; limit: number }> => {
    const qs = new URLSearchParams();
    if (params?.offset != null) qs.set('offset', String(params.offset));
    if (params?.limit != null) qs.set('limit', String(params.limit));
    const res = await apiRequest('GET', `/api/tools${qs.toString() ? `?${qs.toString()}` : ''}`);
    return res.json();
  },
  create: async (tool: {
    tool_id: string;
    name: string;
    description?: string;
    capability_tags?: string[];
    data_domains?: string[];
    side_effect?: 'none'|'reversible'|'irreversible';
    auth_scope?: 'user-delegated'|'service-account';
    arg_schema?: any;
    risk_score?: number;
  }): Promise<{ id: string }> => {
    const res = await apiRequest('POST', `/api/tools`, tool);
    return res.json();
  },
  get: async (id: string): Promise<any> => {
    const res = await apiRequest('GET', `/api/tools/${id}`);
    return res.json();
  },
  update: async (id: string, patch: Partial<any>): Promise<void> => {
    const res = await apiRequest('PUT', `/api/tools/${id}`, patch);
    if (!res.ok) throw new Error(`${res.status}`);
  },
  remove: async (id: string): Promise<void> => {
    const res = await apiRequest('DELETE', `/api/tools/${id}`);
    if (!res.ok) throw new Error(`${res.status}`);
  },
};

// Tool Policies API (tenant-scoped)
export const toolPoliciesApi = {
  get: async (): Promise<{ policies: any[] }> => {
    const res = await apiRequest('GET', `/api/tools/policies`);
    if (!res.ok) throw new Error(`${res.status}`);
    return res.json();
  },
  save: async (payload: { policies: any[] } | any[]): Promise<void> => {
    const body = Array.isArray(payload) ? { policies: payload } : payload;
    const res = await apiRequest('PUT', `/api/tools/policies`, body);
    if (!res.ok) throw new Error(`${res.status}`);
  },
  flushCaches: async (): Promise<void> => {
    const res = await apiRequest('POST', `/api/admin/tool-policies/flush`);
    if (!res.ok) throw new Error(`${res.status}`);
  },
};

// Presets API
export const presetsApi = {
  list: async (): Promise<{ data: Array<{ id: string; name: string; description: string }> }> => {
    const res = await apiRequest('GET', `/api/presets`);
    return res.json();
  },
  get: async (id: string): Promise<any> => {
    const res = await apiRequest('GET', `/api/presets/${id}`);
    return res.json();
  },
  preview: async (id: string): Promise<{ matched: any[]; total: number }> => {
    const res = await apiRequest('GET', `/api/presets/${id}/preview`);
    return res.json();
  },
};

// Tenant API functions
export const tenantApi = {
  // Non-admin: list tenants for current user
  getMine: async (): Promise<APIResponse<Tenant[]>> => {
    const response = await apiRequest('GET', `/api/v1/tenants/my`);
    const json = await response.json();
    // Normalize to { data }
    const list = Array.isArray(json?.tenants) ? json.tenants : (Array.isArray(json?.data) ? json.data : []);
    const data: Tenant[] = (list || []).map((t: any) => ({
      id: t.id,
      name: t.name,
      enabled: t.enabled ?? (t.status ? String(t.status).toLowerCase() === 'active' : true),
      createdAt: t.createdAt ?? t.created_at,
      updatedAt: t.updatedAt ?? t.updated_at,
    }));
    return { success: true, data } as any;
  },

  get: async (id: string): Promise<APIResponse<Tenant>> => {
    const response = await apiRequest('GET', `/api/v1/admin/tenants/${id}`);
    const t = await response.json();
    const data: Tenant = {
      id: t.id,
      name: t.name,
      enabled: t.enabled ?? (t.status ? String(t.status).toLowerCase() === 'active' : true),
      createdAt: t.createdAt ?? t.created_at,
      updatedAt: t.updatedAt ?? t.updated_at,
    } as any;
    return { success: true, data } as any;
  },

  // Create tenant (requires admin via roles or token); BFF will forward appropriately
  create: async (data: InsertTenant): Promise<APIResponse<Tenant>> => {
    const response = await apiRequest('POST', `/api/v1/admin/tenants`, data);
    return response.json();
  },

  update: async (id: string, data: Partial<InsertTenant>): Promise<APIResponse<Tenant>> => {
    const response = await apiRequest('PUT', `/api/v1/admin/tenants/${id}`, data);
    return response.json();
  },

  delete: async (id: string): Promise<APIResponse<void>> => {
    const response = await apiRequest('DELETE', `/api/v1/admin/tenants/${id}`);
    return response.json();
  },
  
  // Standardized list for UI usage
  getAll: async (): Promise<APIResponse<Tenant[]>> => {
    return tenantApi.getMine();
  },
};

// Policy Assignment API functions - Updated to match v1 backend endpoints
export const policyAssignmentApi = {
  // Core Assignment CRUD Operations
  getAll: async (): Promise<APIResponse<PolicyAssignment[]>> => {
    const tenantId = getActiveTenantId();
    try {
      const response = await apiRequest('GET', `/api/admin/tenants/${tenantId}/assignments`);
      // Backend returns { assignments: [...], count, tenant_id }
      const json = await response.json();
const list = Array.isArray(json?.assignments) ? json.assignments : (Array.isArray(json?.data) ? json.data : []);
      const data: PolicyAssignment[] = (list || []).map((a: any) => ({
        id: a.id,
        tenantId: a.tenantId ?? json?.tenant_id ?? tenantId ?? undefined,
        rulepackId: a.rulepackId ?? a.rulepack_id,
        method: a.method || a.Method || '*',
        targetScope: a.targetScope ?? a.target_scope ?? '*',
        priority: Number(a.priority ?? 100),
        enabled: Boolean(a.enabled ?? true),
        createdAt: a.createdAt ?? a.created_at,
        updatedAt: a.updatedAt ?? a.updated_at,
      }));
      return { success: true, data } as any;
    } catch (e: any) {
      const msg = String(e?.message || e);
      if (msg.startsWith('404')) {
        return { success: true, data: [] } as any;
      }
      throw e;
    }
  },

  get: async (id: string): Promise<APIResponse<PolicyAssignment>> => {
    const response = await apiRequest('GET', `/api/admin/assignments/${id}`);
    return response.json();
  },

  create: async (data: InsertPolicyAssignment): Promise<APIResponse<PolicyAssignment>> => {
    const tenantId = getActiveTenantId();
    // Map UI shape to backend expected payload
const payload: any = {
      rulepack_id: (data as any).rulepackId || (data as any).rulepack_id,
      method: (data as any).method || '*',
      target_scope: (data as any).targetScope || (data as any).target_scope || (Array.isArray((data as any).endpoints) ? (data as any).endpoints[0] : undefined) || '*',
      priority: (data as any).priority || 100,
    };
    if ((data as any).enabled !== undefined) payload.enabled = !!(data as any).enabled;
    const response = await apiRequest('POST', `/api/admin/tenants/${tenantId}/assignments`, payload);
    return response.json();
  },

  update: async (id: string, data: Partial<InsertPolicyAssignment>): Promise<APIResponse<PolicyAssignment>> => {
    const payload: any = {};
    if ((data as any).priority != null) payload.priority = (data as any).priority;
    if ((data as any).enabled != null) payload.enabled = !!(data as any).enabled;
    const response = await apiRequest('PUT', `/api/admin/assignments/${id}`, payload);
    return response.json();
  },

  delete: async (id: string): Promise<APIResponse<void>> => {
    const response = await apiRequest('DELETE', `/api/admin/assignments/${id}`);
    return response.json();
  },
  
  // Client-side helpers to satisfy UI callers
  batchCreate: async (data: Array<InsertPolicyAssignment | any>): Promise<{ created: PolicyAssignment[] }> => {
    const created: PolicyAssignment[] = [];
    for (const item of data) {
      const res = await policyAssignmentApi.create(item as any);
      if ((res as any)?.data) created.push((res as any).data);
    }
    return { created };
  },
  getByEndpoint: async (): Promise<PolicyAssignment[]> => {
    const res = await policyAssignmentApi.getAll();
    return (res.data || []).slice();
  },
};

// Audit Trail & Violation Tracking API - Updated to match your backend endpoints
export const auditApi = {
  // Enhanced violations search with comprehensive filtering
  getAll: async (params?: { 
    actions?: string[];
    search?: string;
    decision?: string;
    endpoint?: string;
    reason?: string;
    status?: string;
    timeRange?: string;
    limit?: number;
    offset?: number;
    format?: 'json' | 'csv';
  }): Promise<{ data: AuditEvent[]; total: number }> => {
    // Convert timeRange like "24h" to start_time/end_time
    let start_time: string | undefined;
    let end_time: string | undefined;
    const now = new Date();
    end_time = now.toISOString();
    if (params?.timeRange) {
      const m = params.timeRange.match(/^(\d+)([hd])$/);
      if (m) {
        const amt = parseInt(m[1], 10);
        const unit = m[2];
        const d = new Date(now);
        if (unit === 'h') d.setHours(d.getHours() - amt);
        if (unit === 'd') d.setDate(d.getDate() - amt);
        start_time = d.toISOString();
      }
    }

    const body: any = {
      actions: params?.actions && params.actions.length ? params.actions : [
        'violation','risk_detected','policy_applied','rule_triggered','assignment.create','assignment.update','assignment.delete'
      ],
      start_time,
      end_time,
      limit: params?.limit ?? 100,
      offset: params?.offset ?? 0,
    };

    try {
      const response = await apiRequest('POST', `/api/admin/audits/search`, body);
      const json = await response.json();
      // Normalize to { data, total }
      let events = (json?.events || []) as any[];
      if (!events.length && Array.isArray(json?.data)) events = json.data;
      const total = (json?.total_count ?? json?.count ?? json?.total ?? events.length) as number;
      return { data: events as any, total };
    } catch (e: any) {
      const msg = String(e?.message || e);
      if (msg.startsWith('404')) {
        return { data: [], total: 0 };
      }
      throw e;
    }
  },

  // Search audit events with filters (legacy support)
  search: async (filters: {
    actions?: string[];
    start_time?: string;
    end_time?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ events: AuditEvent[]; total: number }> => {
    const response = await apiRequest('POST', '/api/v1/audit/search', filters);
    return response.json();
  },

  // Get recent violations for dashboard
  getRecentViolations: async (limit = 50, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<{ events: AuditEvent[] }>> => {
    const response = await fetch(`${API_BASE}/admin/audits/search`, {
      method: 'POST',
      headers: getHeaders(userContext),
      credentials: 'include',
      body: JSON.stringify({
        actions: ["request.decision", "scan.decision"],
        limit: limit
      })
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  // Get violation statistics for dashboard
  getViolationStats: async (startDate: Date, endDate: Date, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<{ events: AuditEvent[] }>> => {
    const response = await fetch(`${API_BASE}/admin/audits/search`, {
      method: 'POST',
      headers: getHeaders(userContext),
      credentials: 'include',
      body: JSON.stringify({
        actions: ["request.decision"],
        start_time: startDate.toISOString(),
        end_time: endDate.toISOString(),
        limit: 10000 // Get all for stats calculation
      })
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  // Export audit events (violations)
  exportViolations: async (format = 'json', userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<any> => {
    const response = await fetch(`${API_BASE}/admin/audits/export?format=${format}&limit=10000`, {
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return format === 'json' ? response.json() : response.text();
  },

  // Get audit history for specific object
  getObjectHistory: async (type: string, objectId: string, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<{ events: AuditEvent[] }>> => {
    const response = await fetch(`${API_BASE}/admin/audits/object/${type}/${objectId}`, {
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  // Calculate dashboard statistics from events
  calculateStats: (events: AuditEvent[]) => {
    const stats = {
      totalRequests: events.length,
      blocked: 0,
      quarantined: 0,
      allowed: 0,
      denied: 0,
      topRules: {} as Record<string, number>,
      hourlyBreakdown: {} as Record<number, number>,
      dailyBreakdown: {} as Record<string, number>
    };

    events.forEach(event => {
      const decision = event.metadata?.decision || 'allow';
      const reason = event.metadata?.reason;
      const hour = new Date(event.created_at).getHours();
      const day = new Date(event.created_at).toISOString().split('T')[0];

      // Count decisions
      if (decision === 'deny') stats.denied++;
      else if (decision === 'quarantine') stats.quarantined++;
      else if (decision === 'allow') stats.allowed++;

      stats.blocked = stats.denied + stats.quarantined;

      // Top triggered rules
      if (reason && reason !== 'no_signals') {
        stats.topRules[reason] = (stats.topRules[reason] || 0) + 1;
      }

      // Hourly breakdown
      stats.hourlyBreakdown[hour] = (stats.hourlyBreakdown[hour] || 0) + 1;

      // Daily breakdown
      stats.dailyBreakdown[day] = (stats.dailyBreakdown[day] || 0) + 1;
    });

    return stats;
  },
};

// System API functions
export const systemApi = {
  getInfo: async (userContext?: { userId?: string; userName?: string; tenantId?: string }) => {
    const response = await fetch(`${API_BASE}/admin/system/info`, {
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  getStats: async (userContext?: { userId?: string; userName?: string; tenantId?: string }) => {
    const response = await fetch(`${API_BASE}/admin/system/stats`, {
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  getHealth: async (userContext?: { userId?: string; userName?: string; tenantId?: string }) => {
    const response = await fetch(`${API_BASE}/healthz`, {
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    // Backend returns plain text ("ok"), not JSON
    return response.text();
  },

  getReadiness: async (userContext?: { userId?: string; userName?: string; tenantId?: string }) => {
    const response = await fetch(`${API_BASE}/readyz`, {
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    // Backend returns plain text ("ready"), not JSON
    return response.text();
  },
};

// Service Management API functions (from PromptShield API guide)
export const serviceApi = {
  getAll: async (userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<any[]>> => {
    const response = await fetch(`${API_BASE}/api/v1/services`, {
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  start: async (serviceId: string, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<any>> => {
    const response = await fetch(`${API_BASE}/api/v1/services/${serviceId}/start`, {
      method: 'POST',
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  stop: async (serviceId: string, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<any>> => {
    const response = await fetch(`${API_BASE}/api/v1/services/${serviceId}/stop`, {
      method: 'POST',
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  restart: async (serviceId: string, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<any>> => {
    const response = await fetch(`${API_BASE}/api/v1/services/${serviceId}/restart`, {
      method: 'POST',
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },

  getStatus: async (serviceId: string, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<any>> => {
    const response = await fetch(`${API_BASE}/api/v1/services/${serviceId}/status`, {
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    if (!response.ok) {
      throw new Error(`${response.status}: ${response.statusText}`);
    }
    return response.json();
  },
  // Best-effort scale (if supported by backend)
  scale: async (serviceId: string, replicas: number, userContext?: { userId?: string; userName?: string; tenantId?: string }): Promise<APIResponse<any>> => {
    // Prefer explicit scale endpoint; fallback to restart if not available
    const tryEndpoints = [`${API_BASE}/api/v1/services/${serviceId}/scale`, `${API_BASE}/api/v1/services/${serviceId}/restart`];
    let lastError: any = null;
    for (const url of tryEndpoints) {
      try {
        const response = await fetch(url, {
          method: 'POST',
          headers: getHeaders(userContext),
          credentials: 'include',
          body: JSON.stringify({ replicas }),
        });
        if (response.ok) return response.json();
        lastError = new Error(`${response.status}: ${response.statusText}`);
      } catch (e: any) {
        lastError = e;
      }
    }
    throw (lastError || new Error('scale_failed'));
  },
};

// User Management API
export const userApi = {
  async getAll(userContext?: { userId?: string; userName?: string; tenantId?: string }) {
    const response = await fetch(`${API_BASE}/admin/users`, {
      method: 'GET',
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    return handleResponse(response);
  },

  async create(userData: any, userContext?: { userId?: string; userName?: string; tenantId?: string }) {
    const response = await fetch(`${API_BASE}/admin/users`, {
      method: 'POST',
      headers: getHeaders(userContext),
      credentials: 'include',
      body: JSON.stringify(userData),
    });
    return handleResponse(response);
  },

  async update(userId: string, userData: any, userContext?: { userId?: string; userName?: string; tenantId?: string }) {
    const response = await fetch(`${API_BASE}/admin/users/${userId}`, {
      method: 'PUT',
      headers: getHeaders(userContext),
      credentials: 'include',
      body: JSON.stringify(userData),
    });
    return handleResponse(response);
  },

  async delete(userId: string, userContext?: { userId?: string; userName?: string; tenantId?: string }) {
    const response = await fetch(`${API_BASE}/admin/users/${userId}`, {
      method: 'DELETE',
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    return handleResponse(response);
  },

  async updateStatus(userId: string, status: string, userContext?: { userId?: string; userName?: string; tenantId?: string }) {
    const response = await fetch(`${API_BASE}/admin/users/${userId}/status`, {
      method: 'PUT',
      headers: getHeaders(userContext),
      credentials: 'include',
      body: JSON.stringify({ status }),
    });
    return handleResponse(response);
  },

  async getUserRole(userId: string, userContext?: { userId?: string; userName?: string; tenantId?: string }) {
    const response = await fetch(`${API_BASE}/admin/users/${userId}/role`, {
      method: 'GET',
      headers: getHeaders(userContext),
      credentials: 'include',
    });
    return handleResponse(response);
  },
};

// Usage / Billing Analytics API
export const usageApi = {
  // Summary usage for current tenant (or across system for admins)
  async getSummary(params?: { from?: string; to?: string; by?: 'day'|'hour'|'month' }): Promise<any> {
    const qs = new URLSearchParams();
    if (params?.from) qs.set('from', params.from);
    if (params?.to) qs.set('to', params.to);
    if (params?.by) qs.set('by', params.by);
    const res = await apiRequest('GET', `/api/usage${qs.toString() ? `?${qs.toString()}` : ''}`);
    return res.json();
  },
  // Breakdown by endpoint
  async getByEndpoint(params?: { from?: string; to?: string }): Promise<any> {
    const qs = new URLSearchParams();
    if (params?.from) qs.set('from', params.from);
    if (params?.to) qs.set('to', params.to);
    const res = await apiRequest('GET', `/api/usage/endpoints${qs.toString() ? `?${qs.toString()}` : ''}`);
    return res.json();
  },
  // Breakdown by tool or model
  async getByTool(params?: { from?: string; to?: string }): Promise<any> {
    const qs = new URLSearchParams();
    if (params?.from) qs.set('from', params.from);
    if (params?.to) qs.set('to', params.to);
    const res = await apiRequest('GET', `/api/usage/tools${qs.toString() ? `?${qs.toString()}` : ''}`);
    return res.json();
  },
};

// License API
export const licenseApi = {
  async getInfo(): Promise<any> {
    // Gateway typically exposes /license; via BFF use /api/license
    const res = await apiRequest('GET', `/api/license`);
    return res.json();
  },
  async update(key: string): Promise<any> {
    const res = await apiRequest('POST', `/api/license`, { key });
    return res.json();
  },
};

// Agent Management API
export const agentApi = {
  async authorize(payload: { agent_id: string; tools?: string[]; scopes?: string[]; ttl_seconds?: number }): Promise<any> {
    const res = await apiRequest('POST', `/api/agent/authorize`, payload);
    return res.json();
  },
  async validatePlan(payload: { plan: any; agent_id?: string }): Promise<any> {
    const res = await apiRequest('POST', `/api/agent/validate-plan`, payload);
    return res.json();
  },
  async listExecutions(params?: { limit?: number; offset?: number }): Promise<any> {
    const qs = new URLSearchParams();
    if (params?.limit != null) qs.set('limit', String(params.limit));
    if (params?.offset != null) qs.set('offset', String(params.offset));
    const res = await apiRequest('GET', `/api/agent/executions${qs.toString() ? `?${qs.toString()}` : ''}`);
    return res.json();
  },
};

// Billing API functions
export const billingApi = {
  // Subscription Plans
  getPlans: async (): Promise<{ plans: any[] }> => {
    const res = await apiRequest('GET', `/api/billing/plans`);
    return res.json();
  },
  
  getPlan: async (planId: string): Promise<any> => {
    const res = await apiRequest('GET', `/api/billing/plans/${planId}`);
    return res.json();
  },
  
  // Subscriptions
  getSubscription: async (): Promise<any> => {
    const res = await apiRequest('GET', `/api/billing/subscription`);
    return res.json();
  },
  
  createSubscription: async (data: {
    plan_id: string;
    billing_cycle: 'monthly' | 'yearly';
  }): Promise<any> => {
    const res = await apiRequest('POST', `/api/billing/subscription`, data);
    return res.json();
  },
  
  updateSubscription: async (subscriptionId: string, data: {
    plan_id?: string;
    billing_cycle?: 'monthly' | 'yearly';
    cancel_at_period_end?: boolean;
  }): Promise<any> => {
    const res = await apiRequest('PUT', `/api/billing/subscription/${subscriptionId}`, data);
    return res.json();
  },
  
  cancelSubscription: async (subscriptionId: string, cancelAtPeriodEnd: boolean = true): Promise<any> => {
    const res = await apiRequest('DELETE', `/api/billing/subscription/${subscriptionId}?cancel_at_period_end=${cancelAtPeriodEnd}`);
    return res.json();
  },
  
  // Usage
  getUsage: async (params?: {
    start_date?: string;
    end_date?: string;
  }): Promise<any> => {
    const qs = new URLSearchParams();
    if (params?.start_date) qs.set('start_date', params.start_date);
    if (params?.end_date) qs.set('end_date', params.end_date);
    const res = await apiRequest('GET', `/api/billing/usage${qs.toString() ? `?${qs.toString()}` : ''}`);
    return res.json();
  },
  
  recordUsage: async (data: {
    api_calls: number;
    llm_calls: number;
    violations: number;
  }): Promise<any> => {
    const res = await apiRequest('POST', `/api/billing/usage`, data);
    return res.json();
  },
  
  // Billing History
  getBillingHistory: async (): Promise<{ billing_history: any[] }> => {
    const res = await apiRequest('GET', `/api/billing/history`);
    return res.json();
  },
  
  processBilling: async (data: {
    billing_period_start: string;
    billing_period_end: string;
  }): Promise<any> => {
    const res = await apiRequest('POST', `/api/billing/process`, data);
    return res.json();
  },
  
  // Quota
  checkQuota: async (resourceType: 'api_calls' | 'llm_calls' | 'rulepacks' | 'users'): Promise<any> => {
    const res = await apiRequest('GET', `/api/billing/quota?resource_type=${resourceType}`);
    return res.json();
  },
  
  // Stripe Integration
  createStripeCustomer: async (data: { email: string }): Promise<{ customer_id: string }> => {
    const res = await apiRequest('POST', `/api/billing/stripe/customer`, data);
    return res.json();
  },
  
  createStripeSubscription: async (data: {
    customer_id: string;
    price_id: string;
  }): Promise<{ subscription_id: string }> => {
    const res = await apiRequest('POST', `/api/billing/stripe/subscription`, data);
    return res.json();
  },
};

// Invoice API functions
export const invoiceApi = {
  // List invoices
  getInvoices: async (params?: {
    status?: string;
    billing_period_start?: string;
    billing_period_end?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ invoices: any[]; count: number }> => {
    const qs = new URLSearchParams();
    if (params?.status) qs.set('status', params.status);
    if (params?.billing_period_start) qs.set('billing_period_start', params.billing_period_start);
    if (params?.billing_period_end) qs.set('billing_period_end', params.billing_period_end);
    if (params?.limit) qs.set('limit', params.limit.toString());
    if (params?.offset) qs.set('offset', params.offset.toString());
    const res = await apiRequest('GET', `/api/invoices${qs.toString() ? `?${qs.toString()}` : ''}`);
    return res.json();
  },

  // Generate invoice
  generateInvoice: async (data: {
    subscription_id: string;
    billing_period_start: string;
    billing_period_end: string;
    force_regenerate?: boolean;
  }): Promise<any> => {
    const res = await apiRequest('POST', `/api/invoices/generate`, data);
    return res.json();
  },

  // Get invoice by ID
  getInvoice: async (invoiceId: string): Promise<any> => {
    const res = await apiRequest('GET', `/api/invoices/${invoiceId}`);
    return res.json();
  },

  // Update invoice status
  updateInvoiceStatus: async (invoiceId: string, status: string): Promise<any> => {
    const res = await apiRequest('PUT', `/api/invoices/${invoiceId}/status`, { status });
    return res.json();
  },

  // Generate PDF
  generatePDF: async (invoiceId: string): Promise<{ pdf_url: string }> => {
    const res = await apiRequest('POST', `/api/invoices/${invoiceId}/pdf`);
    return res.json();
  },

  // Send invoice email
  sendEmail: async (invoiceId: string): Promise<any> => {
    const res = await apiRequest('POST', `/api/invoices/${invoiceId}/send`);
    return res.json();
  },

  // Mark as paid
  markAsPaid: async (invoiceId: string, data: {
    paid_at: string;
    stripe_invoice_id?: string;
  }): Promise<any> => {
    const res = await apiRequest('PUT', `/api/invoices/${invoiceId}/mark-paid`, data);
    return res.json();
  },

  // Get invoice summary
  getSummary: async (params?: {
    start_date?: string;
    end_date?: string;
  }): Promise<any> => {
    const qs = new URLSearchParams();
    if (params?.start_date) qs.set('start_date', params.start_date);
    if (params?.end_date) qs.set('end_date', params.end_date);
    const res = await apiRequest('GET', `/api/invoices/summary${qs.toString() ? `?${qs.toString()}` : ''}`);
    return res.json();
  },
};
