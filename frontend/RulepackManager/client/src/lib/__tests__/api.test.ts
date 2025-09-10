import { describe, it, expect, vi, beforeEach } from 'vitest';
import { rulePackApi, policyAssignmentApi, auditApi } from '../api';
import { mockFetchOnce, mockFetchRejectOnce } from '@/test/utils/test-utils';

// Ensure fetch exists for tests
if (!(global as any).fetch) {
  (global as any).fetch = vi.fn();
}

describe('API Client', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset localStorage
    localStorage.clear();
    localStorage.setItem('promptshield_tenant_id', '6f4d338d-f0c0-4091-b54e-f71752c8f568');
  });

  describe('rulePackApi', () => {
    it('getAll sends correct headers and returns data', async () => {
      const mockResponse = [
        { id: '1', name: 'Test RulePack', version: 'v1', active: true, source: 'user' },
      ];
      mockFetchOnce(mockResponse, { ok: true, status: 200 } as any);

      const result = await rulePackApi.getAll();

      expect(fetch).toHaveBeenCalledWith('/api/rulepacks', expect.objectContaining({
        method: 'GET',
        headers: expect.objectContaining({ 'Content-Type': 'application/json' }),
        credentials: 'include',
      }));
      expect(result.data[0].id).toBe('1');
      expect(result.data[0].name).toBe('Test RulePack');
    });

    it('create sends correct payload and headers', async () => {
      const mockResponse = { id: 'new-id', name: 'New RulePack', version: 'v1', active: true };
      mockFetchOnce(mockResponse, { ok: true, status: 201 } as any);

      const newRulePack = {
        name: 'New RulePack',
        description: 'Test description',
        content: 'rules: []',
      };

      const result = await rulePackApi.create(newRulePack);

      expect(fetch).toHaveBeenCalledWith('/api/rulepacks?activate=true', expect.objectContaining({
        method: 'POST',
        headers: expect.any(Object),
        credentials: 'include',
      }));
      expect(result.data.id).toBe('new-id');
    });

    it('handles 404 errors gracefully', async () => {
      mockFetchOnce('Not Found', { ok: false, status: 404, statusText: 'Not Found' } as any);
      await expect(rulePackApi.getAll()).rejects.toThrow('404: Not Found');
    });
  });

  describe('policyAssignmentApi', () => {
    it('getAll sends correct headers and returns data', async () => {
      const mockResponse = { assignments: [{ id: '1', rulepack_id: 'rp1', target_scope: '/api/test' }], count: 1 };
      mockFetchOnce(mockResponse, { ok: true, status: 200 } as any);

      const result = await policyAssignmentApi.getAll();

      expect(fetch).toHaveBeenCalledWith('/api/admin/tenants/6f4d338d-f0c0-4091-b54e-f71752c8f568/assignments', expect.objectContaining({
        method: 'GET',
        headers: expect.any(Object),
        credentials: 'include',
      }));
      expect(result.data.length).toBe(1);
    });

    it('create sends correct payload and headers', async () => {
      const mockResponse = { id: 'new-id' };
      mockFetchOnce(mockResponse, { ok: true, status: 201 } as any);

      const newAssignment = {
        rulepackId: 'rp1',
        endpoints: ['/api/test'],
        priority: 'high' as const,
      };

      const result = await policyAssignmentApi.create(newAssignment);

      expect(fetch).toHaveBeenCalledWith('/api/admin/tenants/6f4d338d-f0c0-4091-b54e-f71752c8f568/assignments', expect.objectContaining({
        method: 'POST',
        headers: expect.any(Object),
        credentials: 'include',
      }));
      expect(result.id).toBe('new-id');
    });

    it('handles 404 errors gracefully', async () => {
      mockFetchOnce('Not Found', { ok: false, status: 404, statusText: 'Not Found' } as any);
      const res = await policyAssignmentApi.getAll();
      expect(res).toEqual({ success: true, data: [] });
    });
  });

  describe('auditApi', () => {
    it('getAll sends POST /api/admin/audits/search', async () => {
      const mockResponse = { events: [{ id: '1', action: 'violation', timestamp: '2024-01-01T00:00:00Z' }], total_count: 1 } as any;
      mockFetchOnce(mockResponse, { ok: true, status: 200 } as any);

      const params = { timeRange: '24h', limit: 50, offset: 0 };
      const result = await auditApi.getAll(params);

      expect(fetch).toHaveBeenCalledWith('/api/admin/audits/search', expect.objectContaining({
        method: 'POST',
        headers: expect.any(Object),
        credentials: 'include',
      }));
      expect(result.total).toBe(1);
    });

    it('search sends correct payload and headers', async () => {
      const mockResponse = { events: [{ id: '1', action: 'violation' }], total: 1 };
      mockFetchOnce(mockResponse, { ok: true, status: 200 } as any);

      const searchParams = {
        timeRange: '24h',
        limit: 50,
        offset: 0,
        action: 'violation',
      } as any;

      const result = await auditApi.search(searchParams);

      expect(fetch).toHaveBeenCalledWith('/api/v1/audit/search', expect.objectContaining({
        method: 'POST',
        headers: expect.any(Object),
        credentials: 'include',
      }));
      expect(result.total).toBeDefined();
    });

    it('handles 404 errors gracefully', async () => {
      mockFetchOnce('Not Found', { ok: false, status: 404, statusText: 'Not Found' } as any);
      const result = await auditApi.getAll({ timeRange: '24h' });
      expect(result).toEqual({ data: [], total: 0 });
    });
  });

  describe('error handling', () => {
    it('throws error for non-404 errors', async () => {
      mockFetchOnce({ error: 'Internal error' }, { ok: false, status: 500, statusText: 'Internal Server Error' } as any);
      await expect(rulePackApi.getAll()).rejects.toThrow('500: Internal Server Error');
    });

    it('handles network errors', async () => {
      mockFetchRejectOnce(new Error('Network error'));
      await expect(rulePackApi.getAll()).rejects.toThrow('Network error');
    });
  });
});
