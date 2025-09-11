# Multi-Tenant Frontend Integration Guide

Complete guide for implementing multi-tenant SaaS architecture with PromptShield.

## 🏗️ Production Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Frontend Applications                      │
│  Customer A App    Customer B App    Customer C App           │
└────────┬──────────────────┬──────────────────┬───────────────┘
         │                  │                  │
         ├──────────────────┴──────────────────┤
         │           Load Balancer              │
         │     (nginx/CloudFlare/AWS ALB)       │
         ├──────────────────┬──────────────────┤
         │                  │                  │
    ┌────▼─────┐      ┌─────▼─────┐     ┌─────▼─────┐
    │   API    │      │  Envoy    │     │   Envoy   │
    │  :9090   │      │  Proxy    │     │   Proxy   │
    └────┬─────┘      └─────┬─────┘     └─────┬─────┘
         │                  │                  │
    ┌────▼──────────────────▼──────────────────▼─────┐
    │        PromptShield Cluster (3+ instances)     │
    │   Instance 1    Instance 2    Instance 3       │
    └────────────────────┬────────────────────────────┘
                         │
                    ┌────▼────┐
                    │Supabase │
                    │Database │
                    └─────────┘
```

## 🚀 Quick Start

### 1. Environment Setup

Create `.env.production` for your frontend:

```bash
# API Endpoints
REACT_APP_PROMPTSHIELD_API=https://api.promptshield.com
REACT_APP_PROMPTSHIELD_PROXY=https://proxy.promptshield.com

# Tenant Configuration
REACT_APP_TENANT_ID=your-tenant-id
REACT_APP_TENANT_NAME=YourCompany

# Optional: Override for local development
REACT_APP_DEV_API=http://localhost:9090
REACT_APP_DEV_PROXY=http://localhost:8080
```

### 2. API Service Implementation

Create `services/promptshield.ts`:

```typescript
import axios, { AxiosInstance } from 'axios';

interface PromptShieldConfig {
  apiUrl: string;
  proxyUrl: string;
  tenantId: string;
  userId?: string;
  userName?: string;
  authToken?: string;
}

export class PromptShieldService {
  private api: AxiosInstance;
  private proxy: AxiosInstance;
  private config: PromptShieldConfig;

  constructor(config: PromptShieldConfig) {
    this.config = config;
    
    // API client for management operations
    this.api = axios.create({
      baseURL: config.apiUrl,
      timeout: 30000,
      headers: this.getHeaders()
    });

    // Proxy client for LLM traffic
    this.proxy = axios.create({
      baseURL: config.proxyUrl,
      timeout: 300000, // 5 minutes for LLM calls
      headers: {
        'X-PS-Tenant-ID': config.tenantId
      }
    });

    this.setupInterceptors();
  }

  private getHeaders() {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'X-PS-Tenant-ID': this.config.tenantId
    };

    // Authentication strategy based on environment
    if (this.config.authToken) {
      // Strategy 1: Admin token for direct API access
      headers['Authorization'] = `Bearer ${this.config.authToken}`;
    } else {
      // Strategy 2: Frontend bypass with user context
      headers['X-PS-Frontend-Auth'] = 'verified';
      if (this.config.userId) {
        headers['X-PS-User-ID'] = this.config.userId;
      }
      if (this.config.userName) {
        headers['X-PS-User-Name'] = this.config.userName;
      }
    }

    return headers;
  }

  private setupInterceptors() {
    // Request interceptor for auth refresh
    this.api.interceptors.request.use(
      (config) => {
        // Update headers with current user context
        const currentUser = this.getCurrentUser();
        if (currentUser) {
          config.headers['X-PS-User-ID'] = currentUser.id;
          config.headers['X-PS-User-Name'] = currentUser.name;
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor for error handling
    this.api.interceptors.response.use(
      (response) => response,
      async (error) => {
        if (error.response?.status === 401) {
          // Handle authentication failure
          await this.refreshAuth();
          return this.api.request(error.config);
        }
        return Promise.reject(error);
      }
    );
  }

  private getCurrentUser() {
    // Get current user from your auth system
    return {
      id: localStorage.getItem('userId') || 'anonymous',
      name: localStorage.getItem('userName') || 'Anonymous User'
    };
  }

  private async refreshAuth() {
    // Implement auth refresh logic
    // This could involve refreshing JWT tokens, re-authenticating, etc.
  }

  // ============= RulePack Management =============

  async listRulePacks(filters?: { enabled?: boolean }) {
    const params = new URLSearchParams(filters as any);
    const response = await this.api.get(`/rulepacks?${params}`);
    return response.data;
  }

  async getRulePack(id: string) {
    const response = await this.api.get(`/rulepacks/${id}`);
    return response.data;
  }

  async createRulePack(rulepack: any) {
    const response = await this.api.post('/rulepacks', rulepack);
    return response.data;
  }

  async updateRulePack(id: string, updates: any) {
    const response = await this.api.put(`/rulepacks/${id}`, updates);
    return response.data;
  }

  async deleteRulePack(id: string) {
    const response = await this.api.delete(`/rulepacks/${id}`);
    return response.data;
  }

  async activateRulePack(id: string) {
    const response = await this.api.post(`/rulepacks/${id}/activate`);
    return response.data;
  }

  async deactivateRulePack(id: string) {
    const response = await this.api.post(`/rulepacks/${id}/deactivate`);
    return response.data;
  }

  // ============= Enforcement =============

  async checkPrompt(content: string, context?: Record<string, any>) {
    const response = await this.api.post('/check', content, {
      headers: {
        'Content-Type': 'text/plain',
        ...(context ? { 'X-PS-Context': JSON.stringify(context) } : {})
      }
    });
    return response.data;
  }

  async scanContent(content: string, options?: { async?: boolean }) {
    const response = await this.api.post('/scan', { content, options });
    return response.data;
  }

  // ============= LLM Proxy =============

  async callOpenAI(messages: any[], options?: any) {
    const response = await this.proxy.post('/v1/chat/completions', {
      model: options?.model || 'gpt-4',
      messages,
      ...options
    }, {
      headers: {
        'Authorization': `Bearer ${process.env.REACT_APP_OPENAI_KEY}`,
        'X-PS-Tenant-ID': this.config.tenantId
      }
    });
    return response.data;
  }

  async callAnthropic(messages: any[], options?: any) {
    const response = await this.proxy.post('/v1/messages', {
      model: options?.model || 'claude-3-opus-20240229',
      messages,
      max_tokens: options?.max_tokens || 1024,
      ...options
    }, {
      headers: {
        'x-api-key': process.env.REACT_APP_ANTHROPIC_KEY,
        'anthropic-version': '2023-06-01',
        'X-PS-Tenant-ID': this.config.tenantId
      }
    });
    return response.data;
  }

  // ============= Service Management =============

  async listServices() {
    const response = await this.api.get('/api/v1/services');
    return response.data;
  }

  async startService(serviceId: string) {
    const response = await this.api.post(`/api/v1/services/${serviceId}/start`);
    return response.data;
  }

  async stopService(serviceId: string) {
    const response = await this.api.post(`/api/v1/services/${serviceId}/stop`);
    return response.data;
  }

  async restartService(serviceId: string) {
    const response = await this.api.post(`/api/v1/services/${serviceId}/restart`);
    return response.data;
  }

  async getServiceStatus(serviceId: string) {
    const response = await this.api.get(`/api/v1/services/${serviceId}/status`);
    return response.data;
  }

  // ============= Tenant Management =============

  async getTenantInfo() {
    const response = await this.api.get(`/tenants/${this.config.tenantId}`);
    return response.data;
  }

  async listTenantAssignments() {
    const response = await this.api.get(`/assignments?tenant_id=${this.config.tenantId}`);
    return response.data;
  }

  async getTenantMetrics() {
    const response = await this.api.get(`/tenants/${this.config.tenantId}/metrics`);
    return response.data;
  }

  // ============= Audit & Monitoring =============

  async getAuditEvents(filters?: {
    event_type?: string;
    resource_type?: string;
    limit?: number;
    offset?: number;
  }) {
    const params = new URLSearchParams({
      tenant_id: this.config.tenantId,
      ...(filters as any)
    });
    const response = await this.api.get(`/audit?${params}`);
    return response.data;
  }

  async getHealthStatus() {
    const response = await this.api.get('/healthz');
    return response.data;
  }

  async getSystemInfo() {
    const response = await this.api.get('/system/info');
    return response.data;
  }

  async getMetrics() {
    const response = await this.api.get('/metrics', {
      responseType: 'text'
    });
    return response.data;
  }
}

// Singleton instance
let instance: PromptShieldService | null = null;

export function initializePromptShield(config?: Partial<PromptShieldConfig>) {
  const defaultConfig: PromptShieldConfig = {
    apiUrl: process.env.NODE_ENV === 'development' 
      ? process.env.REACT_APP_DEV_API || 'http://localhost:9090'
      : process.env.REACT_APP_PROMPTSHIELD_API || 'https://api.promptshield.com',
    proxyUrl: process.env.NODE_ENV === 'development'
      ? process.env.REACT_APP_DEV_PROXY || 'http://localhost:8080'  
      : process.env.REACT_APP_PROMPTSHIELD_PROXY || 'https://proxy.promptshield.com',
    tenantId: process.env.REACT_APP_TENANT_ID || 'default',
    ...config
  };

  instance = new PromptShieldService(defaultConfig);
  return instance;
}

export function getPromptShield(): PromptShieldService {
  if (!instance) {
    throw new Error('PromptShield not initialized. Call initializePromptShield first.');
  }
  return instance;
}
```

### 3. React Context Provider

Create `contexts/PromptShieldContext.tsx`:

```tsx
import React, { createContext, useContext, useEffect, useState } from 'react';
import { initializePromptShield, PromptShieldService } from '../services/promptshield';

interface PromptShieldContextType {
  service: PromptShieldService;
  isReady: boolean;
  tenantId: string;
  systemInfo: any;
}

const PromptShieldContext = createContext<PromptShieldContextType | null>(null);

export const PromptShieldProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [service, setService] = useState<PromptShieldService | null>(null);
  const [isReady, setIsReady] = useState(false);
  const [systemInfo, setSystemInfo] = useState(null);

  useEffect(() => {
    const initService = async () => {
      try {
        // Initialize with user context from your auth system
        const currentUser = await getCurrentUser(); // Your auth logic
        
        const ps = initializePromptShield({
          tenantId: currentUser.tenantId || process.env.REACT_APP_TENANT_ID,
          userId: currentUser.id,
          userName: currentUser.name
        });

        // Verify connection
        const info = await ps.getSystemInfo();
        setSystemInfo(info.data);
        
        setService(ps);
        setIsReady(true);
      } catch (error) {
        console.error('Failed to initialize PromptShield:', error);
        // Handle initialization failure
      }
    };

    initService();
  }, []);

  if (!service) {
    return <div>Initializing security layer...</div>;
  }

  return (
    <PromptShieldContext.Provider 
      value={{ 
        service, 
        isReady, 
        tenantId: process.env.REACT_APP_TENANT_ID || 'default',
        systemInfo 
      }}
    >
      {children}
    </PromptShieldContext.Provider>
  );
};

export const usePromptShield = () => {
  const context = useContext(PromptShieldContext);
  if (!context) {
    throw new Error('usePromptShield must be used within PromptShieldProvider');
  }
  return context;
};
```

### 4. Dashboard Component

Create `components/SecurityDashboard.tsx`:

```tsx
import React, { useState, useEffect } from 'react';
import { usePromptShield } from '../contexts/PromptShieldContext';

export const SecurityDashboard: React.FC = () => {
  const { service, tenantId, systemInfo } = usePromptShield();
  const [rulepacks, setRulepacks] = useState([]);
  const [services, setServices] = useState([]);
  const [metrics, setMetrics] = useState<any>({});
  const [testResult, setTestResult] = useState<any>(null);
  const [testPrompt, setTestPrompt] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    loadDashboardData();
    const interval = setInterval(loadMetrics, 30000); // Poll metrics every 30s
    return () => clearInterval(interval);
  }, []);

  const loadDashboardData = async () => {
    try {
      setLoading(true);
      const [rulepacksRes, servicesRes, metricsRes] = await Promise.all([
        service.listRulePacks({ enabled: true }),
        service.listServices(),
        service.getTenantMetrics()
      ]);
      
      setRulepacks(rulepacksRes.data || []);
      setServices(servicesRes.data || []);
      setMetrics(metricsRes.data || {});
    } catch (error) {
      console.error('Failed to load dashboard:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadMetrics = async () => {
    try {
      const metricsRes = await service.getTenantMetrics();
      setMetrics(metricsRes.data || {});
    } catch (error) {
      console.error('Failed to load metrics:', error);
    }
  };

  const toggleRulePack = async (id: string, enabled: boolean) => {
    try {
      if (enabled) {
        await service.activateRulePack(id);
      } else {
        await service.deactivateRulePack(id);
      }
      await loadDashboardData();
    } catch (error) {
      console.error('Failed to toggle rulepack:', error);
    }
  };

  const testPromptSecurity = async () => {
    if (!testPrompt.trim()) return;
    
    try {
      setLoading(true);
      const result = await service.checkPrompt(testPrompt);
      setTestResult(result);
    } catch (error) {
      console.error('Failed to test prompt:', error);
      setTestResult({ error: 'Failed to test prompt' });
    } finally {
      setLoading(false);
    }
  };

  const callLLMWithSecurity = async () => {
    try {
      setLoading(true);
      // This call goes through the security proxy
      const response = await service.callOpenAI([
        { role: 'user', content: testPrompt }
      ]);
      console.log('LLM Response:', response);
      setTestResult({ 
        allowed: true, 
        llmResponse: response.choices[0].message.content 
      });
    } catch (error: any) {
      if (error.response?.status === 403) {
        setTestResult({ 
          allowed: false, 
          violations: error.response.data.violations 
        });
      } else {
        console.error('LLM call failed:', error);
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="security-dashboard">
      {/* Header */}
      <div className="header">
        <h1>PromptShield Security Dashboard</h1>
        <div className="tenant-info">
          <span>Tenant: {tenantId}</span>
          <span>Mode: {systemInfo?.enforcement_mode || 'loading...'}</span>
          <span className="status">
            {systemInfo ? '🟢 Connected' : '🔴 Disconnected'}
          </span>
        </div>
      </div>

      {/* Metrics */}
      <div className="metrics-grid">
        <div className="metric-card">
          <h3>Requests Today</h3>
          <div className="metric-value">{metrics.requests_today || 0}</div>
        </div>
        <div className="metric-card">
          <h3>Threats Blocked</h3>
          <div className="metric-value">{metrics.threats_blocked || 0}</div>
        </div>
        <div className="metric-card">
          <h3>Active Rules</h3>
          <div className="metric-value">{metrics.active_rules || 0}</div>
        </div>
        <div className="metric-card">
          <h3>Avg Latency</h3>
          <div className="metric-value">{metrics.avg_latency_ms || 0}ms</div>
        </div>
      </div>

      {/* RulePacks */}
      <div className="rulepacks-section">
        <h2>Active Security Policies</h2>
        <div className="rulepacks-list">
          {rulepacks.map((rulepack: any) => (
            <div key={rulepack.id} className="rulepack-card">
              <div className="rulepack-header">
                <h3>{rulepack.metadata.name}</h3>
                <label className="switch">
                  <input 
                    type="checkbox" 
                    checked={rulepack.enabled}
                    onChange={(e) => toggleRulePack(rulepack.id, e.target.checked)}
                  />
                  <span className="slider"></span>
                </label>
              </div>
              <p>{rulepack.metadata.description}</p>
              <div className="rulepack-stats">
                <span>Rules: {rulepack.rules?.length || 0}</span>
                <span>Version: {rulepack.metadata.version}</span>
                <span>Updated: {new Date(rulepack.updated_at).toLocaleDateString()}</span>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Prompt Testing */}
      <div className="test-section">
        <h2>Test Security Policies</h2>
        <div className="test-input">
          <textarea
            value={testPrompt}
            onChange={(e) => setTestPrompt(e.target.value)}
            placeholder="Enter a prompt to test for security violations..."
            rows={4}
          />
          <div className="test-buttons">
            <button 
              onClick={testPromptSecurity}
              disabled={loading || !testPrompt.trim()}
              className="btn-test"
            >
              Test Security Only
            </button>
            <button 
              onClick={callLLMWithSecurity}
              disabled={loading || !testPrompt.trim()}
              className="btn-llm"
            >
              Call LLM (via Proxy)
            </button>
          </div>
        </div>

        {testResult && (
          <div className={`test-result ${testResult.allowed ? 'allowed' : 'blocked'}`}>
            <h3>
              {testResult.allowed ? '✅ ALLOWED' : '🚫 BLOCKED'}
            </h3>
            
            {testResult.violations && testResult.violations.length > 0 && (
              <div className="violations">
                <h4>Security Violations:</h4>
                {testResult.violations.map((v: any, i: number) => (
                  <div key={i} className="violation">
                    <strong>{v.rule_name}</strong>
                    <span className={`severity severity-${v.severity}`}>
                      {v.severity}
                    </span>
                    <p>{v.message}</p>
                    <small>Confidence: {v.confidence}%</small>
                  </div>
                ))}
              </div>
            )}

            {testResult.llmResponse && (
              <div className="llm-response">
                <h4>LLM Response:</h4>
                <p>{testResult.llmResponse}</p>
              </div>
            )}

            {testResult.scan_info && (
              <div className="scan-info">
                <small>
                  Scan ID: {testResult.scan_info.scan_id} | 
                  Processing: {testResult.scan_info.processing_time_ms}ms | 
                  Rules: {testResult.scan_info.rules_evaluated}
                </small>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
```

### 5. Deployment Scripts

Create `deploy.sh`:

```bash
#!/bin/bash

# Deploy to Docker Compose (Development/Staging)
deploy_docker() {
    echo "🚀 Deploying to Docker Compose..."
    
    # Build and start services
    docker compose -f docker-compose.production.yaml up -d --build
    
    # Wait for health checks
    echo "⏳ Waiting for services to be healthy..."
    sleep 10
    
    # Verify deployment
    curl -sf http://localhost:9090/healthz && echo "✅ API is healthy"
    curl -sf http://localhost:8080/ && echo "✅ Proxy is healthy"
}

# Deploy to Kubernetes (Production)
deploy_kubernetes() {
    echo "🚀 Deploying to Kubernetes..."
    
    # Apply all manifests
    kubectl apply -f deploy/k8s/namespace.yaml
    kubectl apply -f deploy/k8s/secret.yaml
    kubectl apply -f deploy/k8s/configmap.yaml
    kubectl apply -f deploy/k8s/rbac.yaml
    kubectl apply -f deploy/k8s/deployment.yaml
    kubectl apply -f deploy/k8s/service.yaml
    kubectl apply -f deploy/k8s/hpa.yaml
    kubectl apply -f deploy/k8s/ingress.yaml
    
    # Wait for rollout
    kubectl rollout status deployment/promptshield -n promptshield
    kubectl rollout status deployment/envoy-proxy -n promptshield
    
    # Get load balancer IPs
    echo "📡 Service endpoints:"
    kubectl get ingress -n promptshield
}

# Main deployment logic
case "$1" in
    docker)
        deploy_docker
        ;;
    k8s|kubernetes)
        deploy_kubernetes
        ;;
    *)
        echo "Usage: $0 {docker|kubernetes}"
        exit 1
        ;;
esac
```

## 🔐 Security Best Practices

### 1. Tenant Isolation

```typescript
// Always include tenant context in requests
const headers = {
  'X-PS-Tenant-ID': tenantId,
  'X-PS-User-ID': userId,
  // Add request signing for extra security
  'X-PS-Request-Signature': await signRequest(payload)
};
```

### 2. Rate Limiting

```typescript
// Implement client-side rate limiting
import { RateLimiter } from 'limiter';

const limiter = new RateLimiter({
  tokensPerInterval: 100,
  interval: 'minute',
  fireImmediately: true
});

async function rateLimitedCall(fn: Function) {
  const remainingRequests = await limiter.removeTokens(1);
  if (remainingRequests < 0) {
    throw new Error('Rate limit exceeded');
  }
  return fn();
}
```

### 3. Connection Resilience

```typescript
// Implement retry logic with exponential backoff
async function resilientCall(fn: Function, maxRetries = 3) {
  let lastError;
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn();
    } catch (error: any) {
      lastError = error;
      if (error.response?.status >= 500) {
        // Server error - retry with backoff
        await new Promise(r => setTimeout(r, Math.pow(2, i) * 1000));
      } else {
        // Client error - don't retry
        throw error;
      }
    }
  }
  throw lastError;
}
```

## 📊 Monitoring & Analytics

### Real-time Metrics Dashboard

```typescript
// Create a metrics polling service
class MetricsService {
  private ws: WebSocket | null = null;
  
  connectWebSocket(tenantId: string) {
    this.ws = new WebSocket(`wss://api.promptshield.com/ws?tenant=${tenantId}`);
    
    this.ws.onmessage = (event) => {
      const metrics = JSON.parse(event.data);
      this.updateDashboard(metrics);
    };
  }
  
  private updateDashboard(metrics: any) {
    // Update your dashboard UI with real-time metrics
    document.dispatchEvent(new CustomEvent('metrics-update', { detail: metrics }));
  }
}
```

## 🎯 Deployment Checklist

- [ ] Configure environment variables for production
- [ ] Set up SSL certificates for HTTPS
- [ ] Configure DNS for api.promptshield.com and proxy.promptshield.com
- [ ] Deploy using Docker Compose or Kubernetes
- [ ] Verify health checks are passing
- [ ] Test tenant isolation
- [ ] Set up monitoring and alerting
- [ ] Configure backup strategy for database
- [ ] Test disaster recovery procedures
- [ ] Document API keys and access controls

The multi-tenant SaaS architecture is now ready for production deployment!