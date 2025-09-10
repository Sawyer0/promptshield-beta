import { useMemo } from 'react';
import { useAuth } from './useAuth';
import { rulePackApi, tenantApi, policyAssignmentApi, auditApi, userApi, serviceApi, systemApi } from '@/lib/api';

/**
 * Hook that provides API functions pre-configured with the current user's context
 * All API calls will automatically include proper authentication headers
 */
export function useAuthenticatedApi() {
  const { user } = useAuth();
  
  const userContext = useMemo(() => ({
    userId: user?.id || '',
    userName: user?.name || user?.firstName || '',
    tenantId: user?.tenants?.[0]?.tenant_id || '6f4d338d-f0c0-4091-b54e-f71752c8f568',
  }), [user]);

  // Return API functions that automatically include user context
  return useMemo(() => ({
    rulePacks: {
      getAll: () => rulePackApi.getAll(userContext),
      get: (id: string) => rulePackApi.get(id, userContext),
      create: (data: any) => rulePackApi.create(data, userContext),
      delete: (id: string) => rulePackApi.delete(id, userContext),
      createVersion: (id: string, data: any) => rulePackApi.createVersion(id, data, userContext),
      getVersion: (id: string, versionNumber: string) => rulePackApi.getVersion(id, versionNumber, userContext),
      listVersions: (id: string) => rulePackApi.listVersions(id, userContext),
      activateVersion: (id: string, versionId: string) => rulePackApi.activateVersion(id, versionId, userContext),
      activateLatest: (id: string) => rulePackApi.activateLatest(id, userContext),
      deactivate: (id: string) => rulePackApi.deactivate(id, userContext),
      purgeVersions: (id: string, keep: number) => rulePackApi.purgeVersions(id, keep, userContext),
    },
    tenants: {
      getAll: () => tenantApi.getAll(),
      get: (id: string) => tenantApi.get(id),
      create: (data: any) => tenantApi.create(data),
      update: (id: string, data: any) => tenantApi.update(id, data),
      delete: (id: string) => tenantApi.delete(id),
    },
    policyAssignments: {
      getAll: () => policyAssignmentApi.getAll(),
      get: (id: string) => policyAssignmentApi.get(id),
      create: (data: any) => policyAssignmentApi.create(data),
      update: (id: string, data: any) => policyAssignmentApi.update(id, data),
      delete: (id: string) => policyAssignmentApi.delete(id),
      batchCreate: (data: any[]) => policyAssignmentApi.batchCreate(data),
      getByEndpoint: () => policyAssignmentApi.getByEndpoint(),
    },
    audit: {
      getAll: (options?: any) => auditApi.getAll(options),
      getViolationStats: (startDate: Date, endDate: Date) => auditApi.getViolationStats(startDate, endDate, userContext),
      getRecentViolations: (limit: number) => auditApi.getRecentViolations(limit, userContext),
      exportViolations: (format?: string) => auditApi.exportViolations(format, userContext),
      search: (filters: any) => auditApi.search(filters),
      getObjectHistory: (type: string, objectId: string) => auditApi.getObjectHistory(type, objectId, userContext),
      calculateStats: auditApi.calculateStats, // This doesn't need user context
    },
    users: {
      getAll: () => userApi.getAll(userContext),
      create: (data: any) => userApi.create(data, userContext),
      update: (userId: string, data: any) => userApi.update(userId, data, userContext),
      delete: (userId: string) => userApi.delete(userId, userContext),
      updateStatus: (userId: string, status: string) => userApi.updateStatus(userId, status, userContext),
      getUserRole: (userId: string) => userApi.getUserRole(userId, userContext),
    },
    services: {
      getAll: () => serviceApi.getAll(userContext),
      start: (serviceId: string) => serviceApi.start(serviceId, userContext),
      stop: (serviceId: string) => serviceApi.stop(serviceId, userContext),
      restart: (serviceId: string) => serviceApi.restart(serviceId, userContext),
      getStatus: (serviceId: string) => serviceApi.getStatus(serviceId, userContext),
    },
    system: {
      getInfo: () => systemApi.getInfo(userContext),
      getStats: () => systemApi.getStats(userContext),
      getHealth: () => systemApi.getHealth(userContext),
      getReadiness: () => systemApi.getReadiness(userContext),
    },
  }), [userContext]);
}
