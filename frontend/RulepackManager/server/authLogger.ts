/**
 * Authentication flow logging utilities
 */

export interface AuthLogContext {
  correlationId?: string;
  userId?: string;
  tenantId?: string;
  path?: string;
  method?: string;
  userAgent?: string;
  remoteAddr?: string;
  component?: string;
}

export class AuthLogger {
  private static getTimestamp(): string {
    return new Date().toISOString();
  }

  private static formatLogMessage(level: string, message: string, context?: AuthLogContext, details?: any): string {
    const logEntry = {
      timestamp: this.getTimestamp(),
      level,
      message,
      component: 'auth',
      ...context,
      ...(details && { details }),
    };
    
    return JSON.stringify(logEntry);
  }

  static info(message: string, context?: AuthLogContext, details?: any): void {
    console.log(this.formatLogMessage('INFO', message, context, details));
  }

  static warn(message: string, context?: AuthLogContext, details?: any): void {
    console.warn(this.formatLogMessage('WARN', message, context, details));
  }

  static error(message: string, context?: AuthLogContext, details?: any): void {
    console.error(this.formatLogMessage('ERROR', message, context, details));
  }

  static debug(message: string, context?: AuthLogContext, details?: any): void {
    if (process.env.NODE_ENV !== 'production') {
      console.debug(this.formatLogMessage('DEBUG', message, context, details));
    }
  }

  static logAuthAttempt(req: any, success: boolean, reason?: string): void {
    const context: AuthLogContext = {
      correlationId: req.headers['x-correlation-id'] || `req_${Date.now()}`,
      path: req.path,
      method: req.method,
      userAgent: req.headers['user-agent'],
      remoteAddr: req.ip || req.connection?.remoteAddress,
    };

    if (success) {
      this.info('Authentication successful', context, {
        userId: req.auth?.userId,
      });
    } else {
      this.warn('Authentication failed', context, {
        reason,
        hasAuth: !!req.auth,
        hasAuthHeader: !!req.headers.authorization,
      });
    }
  }

  static logJWTGeneration(userContext: any, success: boolean, error?: any): void {
    const context: AuthLogContext = {
      userId: userContext.userId,
      tenantId: userContext.tenantId,
    };

    if (success) {
      this.info('JWT token generated successfully', context, {
        hasRoles: !!userContext.roles?.length,
        isAdmin: userContext.isAdmin,
      });
    } else {
      this.error('JWT token generation failed', context, {
        error: error instanceof Error ? error.message : String(error),
        errorType: error?.constructor?.name,
      });
    }
  }

  static logTenantAccess(tenantId: string, userId: string, success: boolean, reason?: string): void {
    const context: AuthLogContext = {
      userId,
      tenantId,
    };

    if (success) {
      this.info('Tenant access granted', context);
    } else {
      this.warn('Tenant access denied', context, { reason });
    }
  }

  static logConfigurationIssue(component: string, issue: string, details?: any): void {
    this.error('Configuration issue detected', { component }, {
      issue,
      ...details,
    });
  }

  static logSecurityEvent(event: string, context?: AuthLogContext, details?: any): void {
    this.warn(`Security event: ${event}`, context, details);
  }
}

/**
 * Express middleware to add auth logging context to requests
 */
export function authLoggingMiddleware(req: any, res: any, next: any): void {
  // Add correlation ID if not present
  if (!req.headers['x-correlation-id']) {
    req.headers['x-correlation-id'] = `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  // Log request start
  AuthLogger.debug('Request started', {
    correlationId: req.headers['x-correlation-id'],
    path: req.path,
    method: req.method,
    userAgent: req.headers['user-agent'],
    remoteAddr: req.ip || req.connection?.remoteAddress,
  });

  next();
}
