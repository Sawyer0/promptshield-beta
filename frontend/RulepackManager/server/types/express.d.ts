// Express Request augmentation for auth injected by Clerk/dev bypass
// Ensures `req.auth` is recognized by TypeScript in route handlers and tests
import 'express-serve-static-core';

declare module 'express-serve-static-core' {
  interface Request {
    auth?: {
      userId?: string | null;
      sessionClaims?: {
        name?: string;
        email?: string;
        org_roles?: string[];
        org_admin?: boolean;
        [key: string]: unknown;
      } | null;
    } | null;
  }
}

export {};
