import express, { type Request, Response, NextFunction } from "express";
import cookieParser from "cookie-parser";
import { registerRoutes } from "./routes";
import { serveStatic } from "./static";
import { log } from "./logger";
import { setupAuth } from "./clerkAuth";
import { initializeJWTAuth } from "./jwtAuth";
import { getValidatedEnvironmentConfig, logEnvironmentConfig } from "./envConfig";
// Clerk-backed sessions; keep readyz minimal without store health

// Load env from local files for the Node server (not handled by Vite)
import fs from 'fs';
import path from 'path';
import dotenv from 'dotenv';
(() => {
  const envCandidates = [
    '.env.dev',
    '.env.local',
    '.env.development',
    '.env',
  ];
  for (const name of envCandidates) {
    const p = path.resolve(process.cwd(), name);
    if (fs.existsSync(p)) {
      dotenv.config({ path: p, override: true });
    }
  }
  // Bridge browser var to server var for Clerk if needed
  if (!process.env.CLERK_PUBLISHABLE_KEY && process.env.VITE_CLERK_PUBLISHABLE_KEY) {
    process.env.CLERK_PUBLISHABLE_KEY = process.env.VITE_CLERK_PUBLISHABLE_KEY;
  }
})();

const app = express();

// Validate and log environment configuration
let envConfig;
try {
  envConfig = getValidatedEnvironmentConfig();
  logEnvironmentConfig(envConfig);
} catch (error) {
  console.error('Environment configuration validation failed:', error);
  process.exit(1);
}

// Initialize JWT configuration early to catch configuration errors
try {
  initializeJWTAuth();
} catch (error) {
  console.error('JWT initialization failed:', error);
  process.exit(1);
}

// IMPORTANT: mount Clerk first so req.auth is available to all downstream handlers
// setupAuth is synchronous for middleware registration; no await required
setupAuth(app);

// Body parsers and signed cookies (tenant selection stored server-side)
app.use(express.json());
app.use(express.urlencoded({ extended: false }));
app.use(cookieParser(envConfig.sessionSecret));

// Add auth logging middleware
import { authLoggingMiddleware } from "./authLogger";
app.use(authLoggingMiddleware);

// Simple health endpoints for containers and load balancers
app.get('/healthz', (_req: Request, res: Response) => {
  res.status(200).json({ status: 'ok', timestamp: new Date().toISOString() });
});
app.get('/readyz', async (_req: Request, res: Response) => {
  const now = new Date().toISOString();
  // Check gateway readiness only (Clerk manages sessions)
  const gw = process.env.PS_GATEWAY_URL || 'http://localhost:8098';
  let gatewayOk = false;
  try {
    const controller = new AbortController();
    const t = setTimeout(() => controller.abort(), 2000);
    const r = await fetch(`${gw}/readyz`, { signal: controller.signal });
    clearTimeout(t);
    gatewayOk = r.ok;
  } catch {
    gatewayOk = false;
  }
  if (!gatewayOk) {
    return res.status(503).json({ status: 'not_ready', gateway: gatewayOk, timestamp: now });
  }
  res.status(200).json({ status: 'ready', gateway: gatewayOk, timestamp: now });
});

app.use((req, res, next) => {
  const start = Date.now();
  const path = req.path;
  let capturedJsonResponse: Record<string, any> | undefined = undefined;

  const originalResJson = res.json;
  res.json = function (bodyJson, ...args) {
    capturedJsonResponse = bodyJson;
    return originalResJson.apply(res, [bodyJson, ...args]);
  };

  res.on("finish", () => {
    const duration = Date.now() - start;
    if (path.startsWith("/api")) {
      let logLine = `${req.method} ${path} ${res.statusCode} in ${duration}ms`;
      if (capturedJsonResponse) {
        logLine += ` :: ${JSON.stringify(capturedJsonResponse)}`;
      }

      if (logLine.length > 80) {
        logLine = logLine.slice(0, 79) + "…";
      }

      log(logLine);
    }
  });

  next();
});

(async () => {
  if ((process.env.NODE_ENV || '').toLowerCase() === 'test') {
    console.log('SERVER_SETUP_START');
  }
  const server = await registerRoutes(app);
  if ((process.env.NODE_ENV || '').toLowerCase() === 'test') {
    console.log('SERVER_SETUP_ROUTES_REGISTERED');
  }

  app.use((err: any, _req: Request, res: Response, _next: NextFunction) => {
    const status = err.status || err.statusCode || 500;
    const message = err.message || "Internal Server Error";

    if ((process.env.NODE_ENV || '').toLowerCase() === 'test') {
      console.error('GLOBAL_ERROR_HANDLER', { 
        status, 
        message, 
        stack: err.stack,
        path: _req.path,
        method: _req.method 
      });
    }

    res.status(status).json({ message });
    throw err;
  });

  // Enable Vite dev middleware whenever not explicitly production
  if ((process.env.NODE_ENV || '').toLowerCase() !== "production" || (process.env.PS_USE_VITE_DEV || '').toLowerCase() === 'true') {
    const { setupVite } = await import("./vite.js");
    await setupVite(app, server);
  } else {
    serveStatic(app);
  }

  // ALWAYS serve the app on the port specified in the environment variable PORT
  // Other ports are firewalled. Default to 5000 if not specified.
  // this serves both the API and the client.
  // It is the only port that is not firewalled.
  const port = parseInt(process.env.PORT || '8096', 10);
  server.listen({
    port,
    host: "0.0.0.0",
    reusePort: true,
  }, () => {
    log(`serving on port ${port}`);
  });
})();
