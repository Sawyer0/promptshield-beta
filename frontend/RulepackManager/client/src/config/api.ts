// API Configuration
export const API_CONFIG = {
  // Backend ps-gateway via Express proxy (same-origin)
  BASE_URL: '', // Same-origin requests via Express proxy

  // Authentication
  FRONTEND_AUTH_TOKEN: 'verified',

  // Default headers
  getHeaders: (userContext?: { userId?: string; userName?: string; tenantId?: string }) => ({
    'Content-Type': 'application/json',
    'X-PS-Frontend-Auth': 'verified',
    'X-Tenant-ID': userContext?.tenantId || '6f4d338d-f0c0-4091-b54e-f71752c8f568',
    'X-PS-User-ID': userContext?.userId || '',
    'X-PS-User-Name': userContext?.userName || '',
  })
};