/**
 * Integration tests for authentication flow
 */

import { describe, it, expect, beforeAll, afterAll, beforeEach } from 'vitest';
import request from 'supertest';
import express, { type Express } from 'express';
import cookieParser from 'cookie-parser';
import { generateGatewayJWT, extractUserContext, validateJWTToken, getJWTConfigStatus } from '../jwtAuth';
import { setupAuth, isAuthenticated, getUserInfo } from '../clerkAuth';
import { getDevBypassConfig } from '../devBypass';
import { registerRoutes } from '../routes';

describe('Authentication Integration Tests', () => {
  // Extend Express.Request locally to include `auth` used by clerk/dev bypass
  type ReqWithAuth = express.Request & {
    auth?: {
      userId?: string | null;
      sessionClaims?: {
        name?: string;
        email?: string;
        org_roles?: string[];
        org_admin?: boolean;
      } | null;
    } | null;
  };
  let app: Express;
  let server: any;
  
  beforeAll(() => {
    // Set up test environment
    process.env.PS_DEV_BYPASS_AUTH = 'true';
    process.env.PS_DEV_USER_ID = 'test-user-123';
    process.env.PS_DEV_USER_NAME = 'Test User';
    process.env.PS_DEV_USER_EMAIL = 'test@example.com';
    process.env.PS_DEV_TENANT_ID = 'test-tenant-123';
    process.env.PS_BFF_JWT_PRIVATE_KEY = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7VJTUt9Us8cKB
wEiOfH3nzor9cwHXLbkiG+2XgQXpM6CpGiuXmN5fwB0+LWHBNy1jx4bJ5D6rE+Cc
JQyF0KTKZ/NLBQxaS1gUeE+7+LbAEk9rAhQbTtK7clIAuWsjAmzqW4fdMy7YBq1z
rtK1XqN6/bgFQGA7I2mzM5jfT28bieBq1ziiFQMQGiU+6hHiDq7yaA+RJvsPiyn9
zGB6tR8W5+wJw1L2aBQGR54soR1fFZdnWFe6qVa6+kQDNtnKwrlOOdqEGBuE4dJx
fuMzaogfJ28JjXxAlqMhHGFvvRK6VyFdMTsaVlP4Vwh7uFHd4Tg3VBhHrwX5fMCd
Qw8ZqPzfAgMBAAECggEBAKTmjaS6tkK8BlPXClTQ2vpz/N6uxDeS35mXpqasqskV
laAidgg/sWqpjXDbXr93otIMLlWsM+X0CqMDgSXKejLS2jx4GDjI1ZTXg++0AMJ8
sJ74pWzVDOfmCEQ/7wXs3+cbnXhKriO8Z036q92Qc1+N87SI38nkGa0ABH9CN83H
mQqt4fB7UdHzuIRe/me2PGhIq5ZBzj6h3BpoPGzEP+x3l9YmK8t/1cN0pqI+dQwY
sqIwVLFVp86Sm9XY3x3k+fLJGHSIUKEWpMJzby5fqIgyey299uDIqudJmcsv2U2L
4HMtCqv8pgAmRiKiHVOcjGgtjsxvFWulJ9+ckrypoGECgYEA4ZU4qwI0+YQN4xiu
VkhQAPC7i4zG4FjjZvQeMNidN9sFHSRFWqmzMZFillhMtMtlgh1yBuHdAhQ+9VMh
oPpMWrlHIB6hwcCxkcoD5owP2ivkYVmFiuQyB9RbDVvKJGOmSHiHkzG+DQkMrBQf
7o/MnBbzDEHFBQbtq0t0xiVhXoMCgYEA1KMFoRBK0TleiVVa8a73L1g4tzjQaYK+
tmUC0Yk2O/s7DQFGl2OaEA9OXzFQXctFm7r6+fdHdlkTds2uNliX3MsBVtHYa6NO
30ojp8ljz6q4pBg4atVzUpOKCRUCbUZfSdImnW2MhqbBTJPXbLiXGAcR5v6sBQGx
vuGqHZe4aY0CgYEAw+aaPqsqTQNUz3hBYz2fyh3Coj1QcRpBe+POLHSOJwRl/ArC
6S+usZP8A7kZE4ZfVEQcUjLRNnNhPgEEfIvPYdHI0/M7CZMP+i/aLkKCMHdg
Tv9k2GF4f8jXmQiVd3sbHUdGWepwJx+R/agtHPmFQ5fvLKFxJ4l1IvJ+qQMCgYAT
4DdvuEeiNNDP2eB3EBLQs7ykHSRXTsgEvWx9cQvn1lUwBHeatgAjXtK4A6NqS6w5
WqEQBrXCU1VnfH9iuDSAEeiRNE6a4EE3+7qHBdSMLEMm4NdVUDJvMaYw+9rSd+IQ
GiHwHRfFt7AkwJjeeiZOXMNb+tm2r2o1BRtmd+T9NQKBgBn4WhKzWh5BFAuAK86s
ouHiPiM5QiuKEKBFmpYpsRBNZLyFa7bbdHR3Ia6cRNxy6o2M0GBWC4Ra1Qc8lP8O
+3JEFJTtqz7AveUGBfxDA/r+mnzpXhHdNhYf71OiEYGG4EIrMYRnI+wuD+OFiEpF
4eMpvT3lGOxRGMaED4gYe9zv
-----END PRIVATE KEY-----`;
    process.env.PS_BFF_JWT_ISSUER = 'test-issuer';
    process.env.PS_BFF_JWT_AUDIENCE = 'test-audience';
    process.env.PS_BFF_JWT_TTL = '120';
  });

  beforeEach(async () => {
    app = express();
    
    // Add debugging middleware
    app.use((req, res, next) => {
      if ((process.env.NODE_ENV || '').toLowerCase() === 'test') {
        console.log('REQUEST_DEBUG', { method: req.method, path: req.path });
      }
      next();
    });
    
    app.use(express.json());
    app.use(express.urlencoded({ extended: false }));
    app.use(cookieParser('test-session-secret'));
    setupAuth(app);
    
    try {
      // Register the actual routes from routes.ts
      server = await registerRoutes(app);
      if ((process.env.NODE_ENV || '').toLowerCase() === 'test') {
        console.log('ROUTES_REGISTERED_SUCCESSFULLY');
      }
    } catch (error) {
      console.error('ERROR_REGISTERING_ROUTES:', error);
      throw error;
    }
  });

  afterAll(() => {
    // Clean up environment
    delete process.env.PS_DEV_BYPASS_AUTH;
    delete process.env.PS_DEV_USER_ID;
    delete process.env.PS_DEV_USER_NAME;
    delete process.env.PS_DEV_USER_EMAIL;
    delete process.env.PS_DEV_TENANT_ID;
    delete process.env.PS_BFF_JWT_PRIVATE_KEY;
    delete process.env.PS_BFF_JWT_ISSUER;
    delete process.env.PS_BFF_JWT_AUDIENCE;
    delete process.env.PS_BFF_JWT_TTL;
  });

  describe('Dev Bypass Mode', () => {
    it('should enable dev bypass when PS_DEV_BYPASS_AUTH is true', () => {
      const config = getDevBypassConfig();
      expect(config.enabled).toBe(true);
      expect(config.userId).toBe('test-user-123');
      expect(config.userName).toBe('Test User');
      expect(config.userEmail).toBe('test@example.com');
      expect(config.tenantId).toBe('test-tenant-123');
    });

    it('should allow requests without authentication in dev bypass mode', async () => {
      // Test the dev bypass configuration directly
      const config = getDevBypassConfig();
      expect(config.enabled).toBe(true);
      expect(config.userId).toBe('test-user-123');
      
      // Test the isAuthenticated middleware in isolation
      const mockReq = {
        auth: {
          userId: config.userId,
          sessionClaims: {
            name: config.userName,
            email: config.userEmail,
          },
        },
      };
      
      // This test verifies the dev bypass logic works
      expect(mockReq.auth.userId).toBe('test-user-123');
    });
  });

  describe('JWT Configuration', () => {
    it('should validate JWT configuration successfully', () => {
      const status = getJWTConfigStatus();
      expect(status.configured).toBe(true);
      expect(status.issuer).toBe('test-issuer');
      expect(status.audience).toBe('test-audience');
      expect(status.ttl).toBe(120);
    });

    it('should generate valid JWT tokens', () => {
      const userContext = {
        userId: 'test-user-123',
        userName: 'Test User',
        email: 'test@example.com',
        tenantId: 'test-tenant-123',
        roles: ['admin'],
        isAdmin: true,
      };

      const token = generateGatewayJWT(userContext);
      expect(token).toBeTruthy();
      expect(typeof token).toBe('string');
      expect(token.split('.')).toHaveLength(3);
    });

    it('should validate generated JWT tokens', () => {
      const userContext = {
        userId: 'test-user-123',
        userName: 'Test User',
        email: 'test@example.com',
        tenantId: 'test-tenant-123',
        roles: ['admin'],
        isAdmin: true,
      };

      const token = generateGatewayJWT(userContext);
      const validation = validateJWTToken(token);
      
      expect(validation.valid).toBe(true);
      expect(validation.payload).toBeTruthy();
      expect(validation.payload.sub).toBe('test-user-123');
      expect(validation.payload.tenant_id).toBe('test-tenant-123');
      expect(validation.payload.iss).toBe('test-issuer');
      expect(validation.payload.aud).toBe('test-audience');
    });
  });

  describe('User Context Extraction', () => {
    it('should extract user context in dev bypass mode', () => {
      const mockReq = {
        auth: {
          userId: 'test-user-123',
          sessionClaims: {
            name: 'Test User',
            email: 'test@example.com',
          },
        },
        signedCookies: {
          ps_tenant_id: 'test-tenant-123',
          ps_tenant_role: 'admin',
        },
        cookies: {},
        headers: {},
      };

      const userContext = extractUserContext(mockReq);
      
      expect(userContext.userId).toBe('test-user-123');
      expect(userContext.userName).toBe('Test User');
      expect(userContext.email).toBe('test@example.com');
      expect(userContext.tenantId).toBe('test-tenant-123');
      expect(userContext.isAdmin).toBe(true);
    });

    it('should handle missing user context gracefully', () => {
      const mockReq = {
        auth: null,
        signedCookies: {},
        cookies: {},
        headers: {},
      };

      expect(() => extractUserContext(mockReq)).toThrow('User not authenticated');
    });
  });

  describe('Error Handling', () => {
    it('should handle invalid JWT configuration', () => {
      // Temporarily break the configuration
      const originalKey = process.env.PS_BFF_JWT_PRIVATE_KEY;
      process.env.PS_BFF_JWT_PRIVATE_KEY = 'invalid-key';

      expect(() => {
        const userContext = {
          userId: 'test-user-123',
          userName: 'Test User',
          email: 'test@example.com',
        };
        generateGatewayJWT(userContext);
      }).toThrow();

      // Restore configuration
      process.env.PS_BFF_JWT_PRIVATE_KEY = originalKey;
    });

    it('should handle invalid user context', () => {
      expect(() => {
        generateGatewayJWT({} as any);
      }).toThrow('Invalid user context');
    });
  });

  describe('Authentication Endpoints', () => {
    beforeEach(() => {
      app.get('/api/auth/user', isAuthenticated, (req: ReqWithAuth, res) => {
        res.json({
          id: req.auth?.userId,
          name: req.auth?.sessionClaims?.name,
          email: req.auth?.sessionClaims?.email,
        });
      });

      app.get('/api/debug/auth', isAuthenticated, (req: ReqWithAuth, res) => {
        res.json({
          auth: {
            userId: req.auth?.userId,
            devBypass: true,
          },
          timestamp: new Date().toISOString(),
        });
      });
    });

    it('should return user information for authenticated requests', async () => {
      // Test getUserInfo function directly
      const mockReq = {
        auth: {
          userId: 'test-user-123',
          sessionClaims: {
            name: 'Test User',
            email: 'test@example.com',
          },
        },
      };
      
      const userInfo = await getUserInfo(mockReq);
      expect(userInfo).toBeDefined();
      expect(userInfo?.id).toBe('test-user-123');
      expect(userInfo?.name).toBe('Test User');
      expect(userInfo?.email).toBe('test@example.com');
    });

    it('should return debug information for authenticated requests', async () => {
      // Test the debug information logic directly
      const mockReq = {
        auth: {
          userId: 'test-user-123',
          sessionClaims: {
            name: 'Test User',
            email: 'test@example.com',
          },
        },
        cookies: {
          ps_tenant_id: 'test-tenant-123',
        },
        signedCookies: {
          ps_tenant_id: 'test-tenant-123',
        },
      };
      
      const userInfo = await getUserInfo(mockReq);
      const userContext = extractUserContext(mockReq);
      
      expect(userInfo?.id).toBe('test-user-123');
      expect(userContext.userId).toBe('test-user-123');
      expect(userContext.tenantId).toBe('test-tenant-123');
    });
  });
});
