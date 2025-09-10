import type { Express, RequestHandler } from 'express';
import { clerkMiddleware, getAuth } from '@clerk/express';
import { createClerkClient } from '@clerk/backend';
import { getDevBypassConfig, logDevBypassStatus } from './devBypass';
import { AuthLogger } from './authLogger';

export function setupAuth(app: Express) {
  const devConfig = getDevBypassConfig();
  const secret = process.env.CLERK_SECRET_KEY; // server-only requirement

  // Log auth mode for debugging
  logDevBypassStatus();

  if ((process.env.NODE_ENV || '').toLowerCase() === 'test') {
    console.log('SETUP_AUTH_CALLED', { devBypass: devConfig.enabled });
  }

  app.set('trust proxy', 1);
  
  if (devConfig.enabled) {
    // No global middleware; dev identity is applied per-route by isAuthenticated
  } else {
    if (!secret) {
      throw new Error('CLERK_SECRET_KEY must be set when dev bypass is disabled');
    }
    app.use(clerkMiddleware());
  }
}

export const isAuthenticated: RequestHandler = async (req: any, res, next) => {
  const devConfig = getDevBypassConfig();
  if ((process.env.NODE_ENV || '').toLowerCase() === 'test') {
    console.log('IS_AUTH_ENTER', req.method, req.path, { devBypass: devConfig.enabled });
  }
  
  try {
    if (devConfig.enabled) {
      // Apply dev identity for this request and allow through
      req.auth = {
        userId: devConfig.userId,
        sessionClaims: {
          name: devConfig.userName,
          email: devConfig.userEmail,
        },
      };
      if ((process.env.NODE_ENV || '').toLowerCase() === 'test') {
        console.log('IS_AUTH_DEV_BYPASS', { userId: req.auth.userId, path: req.path });
      }
      return next();
    }
    
    const auth = getAuth(req);
    if (auth && auth.userId) {
      AuthLogger.logAuthAttempt(req, true);
      return next();
    }

    // Fallback: support Authorization: Bearer <Clerk JWT> from the browser
    const secret = process.env.CLERK_SECRET_KEY;
    if (!secret) throw new Error('missing secret');
    const header = (req.headers['authorization'] || req.headers['Authorization']) as string | undefined;
    if (header && header.toLowerCase().startsWith('bearer ')) {
      const token = header.slice(7).trim();
      if (token) {
        const clerk = createClerkClient({ secretKey: secret });
        const verified: any = await (clerk as any).verifyToken(token);
        if (verified?.sub) {
          req.auth = { userId: verified.sub };
          AuthLogger.logAuthAttempt(req, true, 'clerk_jwt_fallback');
          return next();
        }
      }
    }
  } catch (error) {
    AuthLogger.warn('Clerk token verification failed', {
      correlationId: req.headers['x-correlation-id'],
      path: req.path,
      method: req.method,
    }, {
      error: error instanceof Error ? error.message : String(error),
    });
  }

  AuthLogger.logAuthAttempt(req, false, 'no_valid_auth');
  return res.status(401).json({ 
    error: {
      code: 'UNAUTHORIZED',
      message: 'Authentication required',
      details: { reason: 'invalid_or_missing_auth' }
    }
  });
};

export async function authHealth(): Promise<{ backend: string; ok: boolean; details?: string }>{
  if (process.env.CLERK_SECRET_KEY) {
    return { backend: 'clerk', ok: true };
  }
  return { backend: 'clerk', ok: false, details: 'missing CLERK_SECRET_KEY' };
}

export async function getUserInfo(req: any): Promise<{ id: string; email?: string; firstName?: string; lastName?: string; name?: string } | null> {
  const devConfig = getDevBypassConfig();
  
  if (devConfig.enabled) {
    return {
      id: devConfig.userId,
      email: devConfig.userEmail,
      name: devConfig.userName,
    };
  }
  
  let auth = getAuth(req);
  if (auth?.userId) {
    return { id: auth.userId };
  }
  
  // Fallback header verification for callers that send Bearer token
  try {
    const header = (req.headers['authorization'] || req.headers['Authorization']) as string | undefined;
    if (header && header.toLowerCase().startsWith('bearer ')) {
      const token = header.slice(7).trim();
      if (token) {
        const clerk = createClerkClient({ secretKey: process.env.CLERK_SECRET_KEY! });
        const verified: any = await (clerk as any).verifyToken(token);
        if (verified?.sub) {
          return { id: verified.sub };
        }
      }
    }
  } catch (error) {
    console.warn('Clerk token verification failed:', error instanceof Error ? error.message : String(error));
  }
  
  return null;
}
