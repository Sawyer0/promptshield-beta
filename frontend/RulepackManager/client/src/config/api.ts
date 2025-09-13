// API Configuration
export const API_CONFIG = {
  // Backend ps-gateway via Express proxy (same-origin)
  BASE_URL: '', // Same-origin requests via Express proxy

  // Authentication
  FRONTEND_AUTH_TOKEN: '',

  // Default headers
  getHeaders: (userContext?: { userId?: string; userName?: string; tenantId?: string }) => ({
    'Content-Type': 'application/json'
  })
};
