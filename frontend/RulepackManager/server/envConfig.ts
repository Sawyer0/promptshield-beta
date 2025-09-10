/**
 * Environment configuration validation and documentation
 */

export interface EnvironmentConfig {
  // Authentication Configuration
  clerkSecretKey?: string;
  clerkPublishableKey?: string;
  
  // JWT Configuration
  jwtPrivateKey?: string;
  jwtPublicKey?: string;
  jwtIssuer: string;
  jwtAudience: string;
  jwtTtl: number;
  
  // Development Bypass Configuration
  devBypassAuth: boolean;
  devUserId: string;
  devUserName: string;
  devUserEmail: string;
  devTenantId?: string;
  devRoles?: string[];
  devIsAdmin: boolean;
  
  // Session Configuration
  sessionSecret: string;
  sessionCookieSecure: boolean;
  sessionCookieSameSite: 'strict' | 'lax' | 'none';
  
  // Gateway Configuration
  gatewayUrl: string;
  adminToken?: string;
  
  // Feature Flags
  allowSelfTenantSignup: boolean;
  enableDebugEndpoints: boolean;
  
  // Rate Limiting
  inviteRateLimit: number;
  inviteWindowMs: number;
  
  // Security
  captchaToken?: string;
  corsAllowedOrigins?: string[];
}

/**
 * Load and validate environment configuration
 */
export function loadEnvironmentConfig(): EnvironmentConfig {
  const config: EnvironmentConfig = {
    // Authentication
    clerkSecretKey: process.env.CLERK_SECRET_KEY,
    clerkPublishableKey: process.env.VITE_CLERK_PUBLISHABLE_KEY,
    
    // JWT Configuration
    jwtPrivateKey: process.env.PS_BFF_JWT_PRIVATE_KEY,
    jwtPublicKey: process.env.PS_BFF_JWT_PUBLIC_KEY,
    jwtIssuer: process.env.PS_BFF_JWT_ISSUER || 'promptshield-bff-dev',
    jwtAudience: process.env.PS_BFF_JWT_AUDIENCE || 'promptshield-gateway-dev',
    jwtTtl: parseInt(process.env.PS_BFF_JWT_TTL || '120', 10),
    
    // Development Bypass
    devBypassAuth: (process.env.PS_DEV_BYPASS_AUTH || '').toLowerCase() === 'true',
    devUserId: process.env.PS_DEV_USER_ID || 'dev-user',
    devUserName: process.env.PS_DEV_USER_NAME || 'Dev User',
    devUserEmail: process.env.PS_DEV_USER_EMAIL || 'dev@example.com',
    devTenantId: process.env.PS_DEV_TENANT_ID,
    devRoles: process.env.PS_DEV_ROLES ? process.env.PS_DEV_ROLES.split(',').map(r => r.trim()) : undefined,
    devIsAdmin: (process.env.PS_DEV_IS_ADMIN || '').toLowerCase() === 'true',
    
    // Session Configuration
    sessionSecret: process.env.SESSION_SECRET || 'ps-dev-secret',
    sessionCookieSecure: (process.env.SESSION_COOKIE_SECURE || 'true').toLowerCase() !== 'false',
    sessionCookieSameSite: (process.env.SESSION_COOKIE_SAMESITE as any) || 'lax',
    
    // Gateway Configuration
    gatewayUrl: process.env.PS_GATEWAY_URL || 'http://localhost:8098',
    adminToken: process.env.PS_ADMIN_TOKEN,
    
    // Feature Flags
    allowSelfTenantSignup: (process.env.PS_ALLOW_SELF_TENANT_SIGNUP || '').toLowerCase() === 'true',
    enableDebugEndpoints: (process.env.PS_ENABLE_DEBUG_ENDPOINTS || '').toLowerCase() === 'true',
    
    // Rate Limiting
    inviteRateLimit: parseInt(process.env.PS_INVITE_RATE_LIMIT || '5', 10),
    inviteWindowMs: parseInt(process.env.PS_INVITE_WINDOW_MS || String(10 * 60 * 1000), 10),
    
    // Security
    captchaToken: process.env.PS_CAPTCHA_TOKEN,
    corsAllowedOrigins: process.env.PS_CORS_ALLOWED_ORIGINS ? 
      process.env.PS_CORS_ALLOWED_ORIGINS.split(',').map(o => o.trim()) : undefined,
  };
  
  return config;
}

/**
 * Validate environment configuration and return validation errors
 */
export function validateEnvironmentConfig(config: EnvironmentConfig): string[] {
  const errors: string[] = [];
  
  // Production validation
  const isProduction = (process.env.NODE_ENV || '').toLowerCase() === 'production';
  
  if (isProduction && config.devBypassAuth) {
    errors.push('PS_DEV_BYPASS_AUTH should not be enabled in production');
  }
  
  if (isProduction && config.sessionSecret === 'ps-dev-secret') {
    errors.push('SESSION_SECRET must be set to a secure value in production');
  }
  
  // Authentication validation
  if (!config.devBypassAuth && !config.clerkSecretKey) {
    errors.push('CLERK_SECRET_KEY is required when dev bypass is disabled');
  }
  
  // JWT validation
  if (!config.jwtPrivateKey) {
    errors.push('PS_BFF_JWT_PRIVATE_KEY is required for JWT token generation');
  }
  
  if (config.jwtTtl <= 0 || config.jwtTtl > 3600) {
    errors.push('PS_BFF_JWT_TTL must be between 1 and 3600 seconds');
  }
  
  // Gateway validation
  if (!config.gatewayUrl) {
    errors.push('PS_GATEWAY_URL is required');
  } else {
    try {
      new URL(config.gatewayUrl);
    } catch {
      errors.push('PS_GATEWAY_URL must be a valid URL');
    }
  }
  
  // Rate limiting validation
  if (config.inviteRateLimit <= 0) {
    errors.push('PS_INVITE_RATE_LIMIT must be a positive number');
  }
  
  if (config.inviteWindowMs <= 0) {
    errors.push('PS_INVITE_WINDOW_MS must be a positive number');
  }
  
  return errors;
}

/**
 * Log environment configuration status
 */
export function logEnvironmentConfig(config: EnvironmentConfig): void {
  const isProduction = (process.env.NODE_ENV || '').toLowerCase() === 'production';
  
  console.log('🔧 Environment Configuration:', {
    environment: process.env.NODE_ENV || 'development',
    devBypassAuth: config.devBypassAuth,
    hasClerkSecret: !!config.clerkSecretKey,
    hasJWTPrivateKey: !!config.jwtPrivateKey,
    jwtIssuer: config.jwtIssuer,
    jwtAudience: config.jwtAudience,
    jwtTtl: config.jwtTtl,
    gatewayUrl: config.gatewayUrl,
    allowSelfTenantSignup: config.allowSelfTenantSignup,
    enableDebugEndpoints: config.enableDebugEndpoints,
    sessionCookieSecure: config.sessionCookieSecure,
  });
  
  if (config.devBypassAuth) {
    console.log('🔓 Development bypass enabled:', {
      devUserId: config.devUserId,
      devUserName: config.devUserName,
      devTenantId: config.devTenantId,
      devIsAdmin: config.devIsAdmin,
    });
  }
  
  if (!isProduction && config.enableDebugEndpoints) {
    console.log('🐛 Debug endpoints enabled at /api/debug/*');
  }
}

/**
 * Get environment configuration with validation
 */
export function getValidatedEnvironmentConfig(): EnvironmentConfig {
  const config = loadEnvironmentConfig();
  const errors = validateEnvironmentConfig(config);
  
  if (errors.length > 0) {
    console.error('❌ Environment configuration errors:');
    errors.forEach(error => console.error(`  - ${error}`));
    throw new Error(`Environment configuration validation failed: ${errors.join(', ')}`);
  }
  
  return config;
}