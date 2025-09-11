/**
 * Development bypass utilities for consistent dev mode handling
 */

export interface DevConfig {
  enabled: boolean;
  userId: string;
  userName: string;
  userEmail: string;
  tenantId?: string;
  roles?: string[];
  isAdmin: boolean;
}

/**
 * Get standardized dev bypass configuration
 */
export function getDevBypassConfig(): DevConfig {
  const enabled = (process.env.PS_DEV_BYPASS_AUTH || '').toLowerCase() === 'true';
  
  return {
    enabled,
    userId: process.env.PS_DEV_USER_ID || 'dev-user',
    userName: process.env.PS_DEV_USER_NAME || 'Dev User',
    userEmail: process.env.PS_DEV_USER_EMAIL || 'dev@example.com',
    tenantId: process.env.PS_DEV_TENANT_ID,
    roles: process.env.PS_DEV_ROLES ? process.env.PS_DEV_ROLES.split(',').map(r => r.trim()) : undefined,
    isAdmin: (process.env.PS_DEV_IS_ADMIN || '').toLowerCase() === 'true',
  };
}

/**
 * Check if dev bypass mode is enabled
 */
export function isDevBypassEnabled(): boolean {
  return getDevBypassConfig().enabled;
}

/**
 * Log dev bypass status for debugging
 */
export function logDevBypassStatus(): void {
  const config = getDevBypassConfig();
  
  if (config.enabled) {
    console.log('🔧 Development bypass mode enabled', {
      userId: config.userId,
      userName: config.userName,
      userEmail: config.userEmail,
      tenantId: config.tenantId,
      roles: config.roles,
      isAdmin: config.isAdmin,
    });
  } else {
    console.log('🔒 Production authentication mode enabled');
  }
}

/**
 * Create dev user context for JWT generation
 */
export function createDevUserContext() {
  const config = getDevBypassConfig();
  
  if (!config.enabled) {
    throw new Error('Dev bypass is not enabled');
  }
  
  return {
    userId: config.userId,
    userName: config.userName,
    email: config.userEmail,
    tenantId: config.tenantId,
    roles: config.roles,
    isAdmin: config.isAdmin,
  };
}