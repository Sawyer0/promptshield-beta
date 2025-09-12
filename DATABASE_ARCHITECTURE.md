# PromptShield Database Architecture

## 🎯 Overview

This document defines the comprehensive database architecture for PromptShield, following our **hybrid database strategy** that scales from development to production while supporting our multi-tenant, compliance-focused AI security platform.

## 🏗️ Hybrid Database Strategy

### **Progressive Database Architecture**

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Development   │    │     Staging     │    │   Production    │
│                 │    │                 │    │                 │
│ Local PostgreSQL│───▶│    Supabase     │───▶│ Aurora PostgreSQL│
│                 │    │                 │    │                 │
│ • Fast iteration│    │ • Shared access │    │ • Enterprise    │
│ • No costs      │    │ • Free tier     │    │ • High availability│
│ • Full control  │    │ • Easy setup    │    │ • Auto-scaling  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### **Environment-Specific Optimizations**

#### **Development: Local PostgreSQL**
- **Purpose**: Fast iteration and development
- **Features**: Full PostgreSQL features, local control
- **Migration Path**: `migrations/` directory
- **Benefits**: Zero cost, offline development, full schema control

#### **Staging: Supabase (PostgreSQL)**
- **Purpose**: Team testing and cloud validation
- **Features**: Managed PostgreSQL with free tier
- **Migration Path**: `migrations_consolidated/` directory
- **Benefits**: Shared access, real cloud testing, built-in backups

#### **Production: Aurora PostgreSQL**
- **Purpose**: Enterprise-scale production deployment
- **Features**: Aurora-optimized with enterprise features
- **Migration Path**: `migrations_aurora/` directory
- **Benefits**: High availability, auto-scaling, performance insights

## 🏗️ Architecture Principles

### 1. **Production: Aurora-Optimized Design**
- **Read Replicas**: Distribute read load across multiple replicas
- **Aurora Serverless**: Auto-scaling for variable workloads
- **Aurora Global Database**: Multi-region disaster recovery
- **Performance Insights**: Built-in monitoring and optimization

### 2. **Multi-Tenant Isolation**
- **Row Level Security (RLS)**: Complete tenant data isolation
- **Tenant Context Functions**: Automatic query filtering
- **Audit Trail Isolation**: Separate compliance evidence per tenant

### 3. **Compliance-First Storage**
- **Immutable Audit Logs**: Tamper-evident compliance evidence
- **Data Retention Policies**: Automated compliance data lifecycle
- **Evidence Generation**: Automated compliance report generation

### 4. **Performance Optimization**
- **Partitioning**: Time-based partitioning for high-volume tables
- **Indexing Strategy**: Optimized for Aurora's query patterns
- **Caching Layers**: Multi-level caching for hot data

## 🔄 Migration Strategy

### **Environment-Specific Migration Paths**

#### **Development Migrations (`migrations/`)**
```bash
# Local PostgreSQL development
export PS_PG_DSN="postgres://promptshield:example@localhost:5432/promptshield_dev?sslmode=disable"
go run scripts/run-migrations.go
```

#### **Staging Migrations (`migrations_consolidated/`)**
```bash
# Supabase staging environment
export PS_PG_DSN="postgres://postgres:[password]@db.[project].supabase.co:5432/postgres?sslmode=require"
go run scripts/run-consolidated-migrations.go
```

#### **Production Migrations (`migrations_aurora/`)**
```bash
# Aurora PostgreSQL production
export AURORA_PG_DSN="postgres://promptshield:[password]@[aurora-cluster-endpoint]:5432/promptshield?sslmode=require"
go run scripts/run-aurora-migrations.go
```

### **Migration Best Practices**

1. **Test in Development First**: Always validate migrations locally
2. **Staging Validation**: Test consolidated migrations in Supabase
3. **Production Deployment**: Use Aurora-optimized migrations
4. **Rollback Capability**: Maintain rollback scripts for each environment

## 🔧 Environment Configuration

### **Development Environment**
```yaml
# docker-compose.dev.yml
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: promptshield_dev
      POSTGRES_USER: promptshield
      POSTGRES_PASSWORD: example
    ports:
      - "5432:5432"

  promptshield:
    environment:
      PS_PG_DSN: postgres://promptshield:example@postgres:5432/promptshield_dev?sslmode=disable
      PS_DEV_BYPASS_AUTH: "true"
```

### **Staging Environment**
```bash
# .env.staging
PS_PG_DSN=postgres://postgres:[password]@db.[project].supabase.co:5432/postgres?sslmode=require
PS_TENANT_ID=staging-tenant-uuid
PS_ENVIRONMENT=staging
```

### **Production Environment**
```bash
# .env.production
PS_PG_DSN=postgres://promptshield:[password]@[aurora-cluster-endpoint]:5432/promptshield?sslmode=require
AURORA_PROXY_ENDPOINT=[rds-proxy-endpoint]
AURORA_DB_NAME=promptshield
PS_TENANT_ID=production-tenant-uuid
PS_ENVIRONMENT=production
```

## 💰 Cost Analysis & Benefits

### **Development Environment**
- **Cost**: $0 (local PostgreSQL)
- **Benefits**: Fast iteration, full control, offline development
- **Limitations**: Single developer, no sharing

### **Staging Environment**
- **Cost**: $0 (Supabase free tier)
- **Benefits**: Team access, cloud testing, built-in backups
- **Limitations**: 500MB storage, 2GB bandwidth/month

### **Production Environment**
- **Cost**: $200-500/month (Aurora)
- **Benefits**: Enterprise features, high availability, auto-scaling
- **Limitations**: Higher cost, more complex setup

### **Strategic Benefits**

1. **Cost Optimization**: Free development and staging, pay only for production
2. **Team Collaboration**: Shared staging environment for testing
3. **Risk Mitigation**: Gradual progression from simple to complex
4. **Scalability**: Start simple, scale to enterprise production

## 📊 Data Categories & Storage Strategy

### **Category 1: Core Business Data (Hot Storage)**
**Purpose**: Active business operations, real-time access required
**Retention**: Indefinite (with archival)
**Aurora Optimization**: Primary tables with read replicas

#### Tables:
- `tenants` - Master tenant registry
- `users` - User accounts and authentication
- `rulepacks` - Security policy containers ⚠️ **HIGH IMPACT**
- `rulepack_versions` - Policy version history ⚠️ **HIGH IMPACT**
- `assignments` - Policy-to-endpoint assignments
- `tools` - Tool registry and metadata ⚠️ **MEDIUM IMPACT**
- `tenant_settings` - Tool policies (allow/deny) ⚠️ **HIGH IMPACT**
- `tenant_services` - Service registrations
- `platform_users` - Platform administration

### **Category 2: Real-Time Operations (Hot Storage)**
**Purpose**: Live scanning, enforcement decisions, real-time monitoring
**Retention**: 30 days (hot), 1 year (warm), 7 years (cold)
**Aurora Optimization**: Partitioned tables with Aurora Serverless

#### Tables:
- `violations` - Security violations (partitioned by date)
- `scan_results` - Scan operation results (partitioned by date)
- `enforcement_decisions` - Real-time enforcement decisions
- `threat_intelligence` - Active threat data
- `alerts` - Security alerts and notifications

### **Category 3: Compliance & Audit (Warm Storage)**
**Purpose**: Compliance evidence, audit trails, regulatory reporting
**Retention**: 7 years (immutable)
**Aurora Optimization**: Read-optimized with compression

#### Tables:
- `audits` - Immutable audit trail (partitioned by date)
- `compliance_evidence` - Generated compliance reports
- `compliance_violations` - Compliance requirement violations
- `data_retention_logs` - Data lifecycle tracking

### **Category 4: Analytics & Metrics (Warm Storage)**
**Purpose**: Performance metrics, usage analytics, business intelligence
**Retention**: 2 years (detailed), 7 years (aggregated)
**Aurora Optimization**: Aggregated tables with materialized views

#### Tables:
- `usage_metrics` - Usage statistics (partitioned by date)
- `performance_metrics` - System performance data
- `scan_metrics` - Scanning performance analytics
- `tenant_metrics` - Per-tenant usage analytics

### **Category 5: Configuration & Metadata (Hot Storage)**
**Purpose**: System configuration, feature flags, tenant settings
**Retention**: Indefinite
**Aurora Optimization**: Small tables with frequent access

#### Tables:
- `platform_settings` - Global platform configuration
- `tenant_settings` - Per-tenant configuration
- `feature_flags` - Feature toggle management
- `tools` - Tool registry and policies
- `webhooks` - Webhook configurations

## 🗄️ Table Design Specifications

### **Core Business Tables**

```sql
-- TENANTS: Master tenant registry
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    status VARCHAR(50) DEFAULT 'active',
    plan_type VARCHAR(50) DEFAULT 'free',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- RULEPACKS: Security policy containers (HIGH PERFORMANCE IMPACT)
CREATE TABLE rulepacks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    
    -- DUAL STORAGE STRATEGY: Both YAML and JSONB for performance
    yaml_content TEXT,                    -- Raw YAML from UI (up to 1MB)
    rules JSONB,                          -- Parsed/compiled rules (indexed)
    current_version_id UUID,
    
    -- Management fields  
    is_active BOOLEAN DEFAULT true,
    status TEXT DEFAULT 'active' CHECK (status IN ('draft', 'active', 'archived')),
    enforcement_mode TEXT DEFAULT 'enforce' CHECK (enforcement_mode IN ('monitor', 'enforce', 'redact')),
    fail_on_severity TEXT DEFAULT 'HIGH' CHECK (fail_on_severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    priority INTEGER DEFAULT 100,
    metadata JSONB DEFAULT '{}',
    
    -- Performance tracking
    rule_count INTEGER DEFAULT 0,        -- Cached rule count for quick queries
    content_size_bytes INTEGER DEFAULT 0, -- Cached content size
    last_compiled_at TIMESTAMPTZ,        -- When rules were last compiled
    
    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- RULEPACK VERSIONS: Version history with audit trail (HIGH STORAGE IMPACT)
CREATE TABLE rulepack_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rulepack_id UUID NOT NULL REFERENCES rulepacks(id) ON DELETE CASCADE,
    version INT NOT NULL,
    
    -- DUAL STORAGE: Raw YAML + Parsed JSONB
    yaml_content TEXT NOT NULL,           -- Raw YAML from UI (up to 1MB per version)
    dsl JSONB NOT NULL,                   -- Parsed/compiled rules (indexed)
    
    -- Version management
    status TEXT NOT NULL CHECK (status IN ('draft','approved','active','archived')),
    created_by UUID,
    approval_workflow JSONB,
    
    -- Performance tracking
    rule_count INTEGER DEFAULT 0,        -- Cached rule count
    content_size_bytes INTEGER DEFAULT 0, -- Cached content size
    compilation_time_ms INTEGER,         -- How long compilation took
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(rulepack_id, version)
);
```

### **Real-Time Operations Tables (Partitioned)**

```sql
-- VIOLATIONS: Security violations (partitioned by date)
CREATE TABLE violations (
    id UUID DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    rulepack_id UUID NOT NULL REFERENCES rulepacks(id) ON DELETE CASCADE,
    rule_id TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    message TEXT NOT NULL,
    input_hash TEXT NOT NULL, -- For deduplication
    scan_result JSONB,
    enforcement_action TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create monthly partitions
CREATE TABLE violations_2025_01 PARTITION OF violations
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- SCAN RESULTS: Scan operation results (partitioned by date)
CREATE TABLE scan_results (
    id UUID DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    request_id TEXT NOT NULL,
    endpoint TEXT NOT NULL,
    method TEXT NOT NULL,
    scan_status TEXT NOT NULL,
    violations_count INTEGER DEFAULT 0,
    processing_time_ms INTEGER,
    decision JSONB,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);
```

### **Compliance & Audit Tables (Immutable)**

```sql
-- AUDITS: Immutable audit trail (partitioned by date)
CREATE TABLE audits (
    id UUID DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_id UUID,
    actor_email TEXT,
    action TEXT NOT NULL,
    object_type TEXT NOT NULL,
    object_id UUID NOT NULL,
    diff JSONB,
    metadata JSONB,
    integrity_hash TEXT NOT NULL, -- Tamper detection
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- COMPLIANCE EVIDENCE: Generated compliance reports
CREATE TABLE compliance_evidence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    standard TEXT NOT NULL, -- SOC2, HIPAA, GDPR, etc.
    requirement_id TEXT NOT NULL,
    evidence_type TEXT NOT NULL,
    time_range_start TIMESTAMPTZ NOT NULL,
    time_range_end TIMESTAMPTZ NOT NULL,
    event_count INTEGER NOT NULL,
    evidence_data JSONB NOT NULL,
    integrity_hash TEXT NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    generated_by TEXT NOT NULL
);
```

### **Analytics & Metrics Tables (Aggregated)**

```sql
-- USAGE METRICS: Usage statistics (partitioned by date)
CREATE TABLE usage_metrics (
    id UUID DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    metric_type TEXT NOT NULL,
    metric_value NUMERIC NOT NULL,
    dimensions JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- PERFORMANCE METRICS: System performance data
CREATE TABLE performance_metrics (
    id UUID DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    operation TEXT NOT NULL,
    latency_ms INTEGER NOT NULL,
    throughput_rps NUMERIC,
    error_rate NUMERIC,
    resource_usage JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);
```

## ⚠️ RulePack Performance & Storage Impact

### **UI-Created RulePack Storage Challenges**

When users create rulepacks through the UI, they have significant database performance and storage implications:

#### **1. Storage Size Impact**
```sql
-- Current limits from codebase analysis:
-- PS_MAX_RULEPACK_KB = 1024 KB (1MB) per rulepack
-- PS_MAX_RULES = 1000 rules per rulepack
-- PS_MAX_RULEPACK_BODY_BYTES = 1MB for HTTP requests

-- Real-world examples:
-- comprehensive.yaml: 8,551 bytes (8.5KB)
-- essentials.yaml: ~3-5KB
-- Large enterprise rulepack: 50-200KB
-- Maximum theoretical: 1MB per rulepack
```

#### **2. Dual Storage Strategy**
```sql
-- Why we store both YAML and JSONB:
-- yaml_content TEXT: Raw YAML from UI (for editing, versioning, compliance)
-- rules JSONB: Parsed/compiled rules (for fast scanning, indexing)

-- Storage multiplication effect:
-- 1 rulepack = 2x storage (YAML + JSONB)
-- 10 versions = 20x storage per rulepack
-- 100 tenants × 10 rulepacks × 10 versions = 20,000 storage units
```

#### **3. Performance Impact Analysis**

| **Operation** | **Impact** | **Mitigation** |
|---------------|------------|----------------|
| **UI Creation** | High I/O, JSONB parsing | Async processing, caching |
| **Version History** | Exponential storage growth | Automated archival |
| **Rule Compilation** | CPU intensive | Background compilation |
| **Scanning Engine** | Memory intensive | Lazy loading, streaming |
| **Multi-tenant Queries** | RLS overhead | Optimized indexes |

#### **4. Aurora Optimization for RulePacks**

```sql
-- Specialized indexes for rulepack performance
CREATE INDEX idx_rulepacks_active_tenant ON rulepacks(tenant_id, is_active) 
WHERE is_active = true;

CREATE INDEX idx_rulepacks_rules_gin ON rulepacks USING GIN (rules);
CREATE INDEX idx_rulepacks_metadata_gin ON rulepacks USING GIN (metadata);

-- Partial indexes for common queries
CREATE INDEX idx_rulepacks_active_priority ON rulepacks(priority, tenant_id) 
WHERE is_active = true AND status = 'active';

-- Rulepack version performance
CREATE INDEX idx_rulepack_versions_active ON rulepack_versions(rulepack_id, status) 
WHERE status = 'active';

CREATE INDEX idx_rulepack_versions_dsl_gin ON rulepack_versions USING GIN (dsl);
```

#### **5. Storage Growth Projections**

```sql
-- Conservative estimates per tenant:
-- 5 active rulepacks × 1MB = 5MB
-- 10 versions per rulepack × 1MB = 50MB
-- 100 tenants × 50MB = 5GB
-- 1000 tenants × 50MB = 50GB
-- 10,000 tenants × 50MB = 500GB

-- With Aurora compression (70%):
-- 500GB → 150GB actual storage
-- Still significant for hot storage
```

#### **6. Performance Monitoring**

```sql
-- RulePack performance monitoring
CREATE VIEW rulepack_performance_summary AS
SELECT 
    tenant_id,
    COUNT(*) as total_rulepacks,
    COUNT(*) FILTER (WHERE is_active = true) as active_rulepacks,
    AVG(content_size_bytes) as avg_size_bytes,
    MAX(content_size_bytes) as max_size_bytes,
    SUM(content_size_bytes) as total_size_bytes,
    AVG(rule_count) as avg_rules_per_pack,
    MAX(rule_count) as max_rules_per_pack
FROM rulepacks
GROUP BY tenant_id;

-- Version history impact
CREATE VIEW rulepack_version_impact AS
SELECT 
    r.tenant_id,
    r.name as rulepack_name,
    COUNT(rv.id) as version_count,
    SUM(rv.content_size_bytes) as total_version_size,
    MAX(rv.created_at) as last_version_date
FROM rulepacks r
LEFT JOIN rulepack_versions rv ON r.id = rv.rulepack_id
GROUP BY r.tenant_id, r.id, r.name;
```

#### **7. Automated Cleanup Strategy**

```sql
-- Automated rulepack version cleanup
CREATE OR REPLACE FUNCTION cleanup_old_rulepack_versions()
RETURNS VOID AS $$
BEGIN
    -- Archive versions older than 1 year (keep last 10 versions)
    WITH old_versions AS (
        SELECT rv.id, rv.rulepack_id, rv.version
        FROM rulepack_versions rv
        WHERE rv.created_at < NOW() - INTERVAL '1 year'
        AND rv.status != 'active'
        AND rv.version < (
            SELECT MAX(version) - 10 
            FROM rulepack_versions rv2 
            WHERE rv2.rulepack_id = rv.rulepack_id
        )
    )
    INSERT INTO rulepack_versions_archive 
    SELECT rv.* FROM rulepack_versions rv
    INNER JOIN old_versions ov ON rv.id = ov.id;
    
    -- Delete archived versions
    DELETE FROM rulepack_versions 
    WHERE id IN (
        SELECT id FROM old_versions
    );
END;
$$ LANGUAGE plpgsql;

-- Schedule cleanup monthly
SELECT cron.schedule('rulepack-cleanup', '0 3 1 * *', 
    'SELECT cleanup_old_rulepack_versions();');
```

#### **8. UI Performance Considerations**

```typescript
// Frontend optimizations for large rulepacks
interface RulePackPerformance {
  // Lazy load rule content
  loadRules: (id: string) => Promise<Rule[]>;
  
  // Stream large rulepacks
  streamRulePack: (id: string) => AsyncIterable<Rule>;
  
  // Cache compiled rules
  cacheCompiledRules: (rules: Rule[]) => void;
  
  // Paginate rule lists
  getRulesPaginated: (page: number, limit: number) => Promise<Rule[]>;
}

// Backend optimizations
const rulePackOptimizations = {
  // Async rule compilation
  compileRulesAsync: true,
  
  // Background processing
  processInBackground: true,
  
  // Streaming responses
  streamLargeResponses: true,
  
  // Compression
  compressResponses: true,
  
  // Caching
  cacheCompiledRules: true
};
```

## ⚡ Real-Time Rule Application by Endpoint

### **Critical Performance Path: Endpoint → RulePack Resolution**

When a request hits an endpoint, the system must resolve which rulepacks apply in real-time. This is the **hottest path** in your database:

#### **1. Real-Time Resolution Flow**

```go
// Every request triggers this flow:
func resolveApplicableRulepacks(ctx context.Context, endpoint string, method string) ([]rules.RulePack, error) {
    // 1. Get tenant ID from context
    tenantID := getTenantID(ctx)
    
    // 2. Query assignments for tenant (HOT PATH)
    assignments, err := assignmentRepo.ListByTenant(ctx, tenantID)
    
    // 3. Filter by endpoint pattern matching
    for _, assignment := range assignments {
        if matchesEndpointPattern(endpoint, assignment.TargetScope) {
            // 4. Fetch rulepack content (HOT PATH)
            dsl, err := rulepackService.GetActive(ctx, assignment.RulepackID)
            
            // 5. Parse YAML to RulePack struct
            var rulePack rules.RulePack
            yaml.Unmarshal(dsl, &rulePack)
        }
    }
}
```

#### **2. Database Query Performance Impact**

```sql
-- HOT PATH QUERY 1: List assignments by tenant
-- Called for EVERY request to ANY endpoint
SELECT id, tenant_id, rulepack_id, target_scope, method, priority, enabled, created_at, updated_at 
FROM rulepack_assignments 
WHERE tenant_id = $1 
ORDER BY priority DESC, created_at ASC;

-- HOT PATH QUERY 2: Get active rulepack content
-- Called for each matching assignment
SELECT yaml_content, rules, current_version_id
FROM rulepacks 
WHERE id = $1 AND is_active = true;

-- HOT PATH QUERY 3: Get rulepack version content
-- Called if current_version_id exists
SELECT dsl, yaml_content, rules
FROM rulepack_versions 
WHERE id = $1 AND status = 'active';
```

#### **3. Performance Bottlenecks Analysis**

| **Operation** | **Frequency** | **Impact** | **Mitigation** |
|---------------|---------------|------------|----------------|
| **ListByTenant** | Every request | High I/O | Redis caching |
| **GetActive** | Per matching assignment | High I/O | Redis caching |
| **YAML Parsing** | Per rulepack | CPU intensive | Background compilation |
| **Endpoint Matching** | Per assignment | CPU intensive | Optimized patterns |
| **Rule Compilation** | Per rulepack | CPU intensive | Cached compilation |

#### **4. Aurora Optimization for Real-Time Resolution**

```sql
-- Critical indexes for real-time performance
CREATE INDEX idx_rulepack_assignments_tenant_priority 
ON rulepack_assignments(tenant_id, priority DESC, enabled) 
WHERE enabled = true;

CREATE INDEX idx_rulepack_assignments_tenant_scope 
ON rulepack_assignments(tenant_id, target_scope, enabled) 
WHERE enabled = true;

-- Partial index for active rulepacks only
CREATE INDEX idx_rulepacks_active_tenant 
ON rulepacks(tenant_id, id) 
WHERE is_active = true;

-- GIN index for JSONB rule content
CREATE INDEX idx_rulepacks_rules_gin 
ON rulepacks USING GIN (rules);

-- Covering index for assignment resolution
CREATE INDEX idx_assignments_covering 
ON rulepack_assignments(tenant_id, target_scope, method, rulepack_id, priority, enabled);
```

#### **5. Redis Caching Strategy**

```go
// Multi-level caching for real-time performance
type RulePackCache struct {
    // L1: Assignment cache (per tenant)
    assignmentCache map[uuid.UUID][]*RulepackAssignment
    
    // L2: RulePack content cache (per rulepack)
    rulepackCache map[uuid.UUID]*RulePack
    
    // L3: Compiled rules cache (per rulepack)
    compiledCache map[uuid.UUID][]compiledRule
}

// Cache keys for Redis
const (
    AssignmentKey = "assignments:tenant:%s"
    RulePackKey   = "rulepack:content:%s"
    CompiledKey   = "rulepack:compiled:%s"
)

// Cache TTL strategy
const (
    AssignmentTTL = 5 * time.Minute   // Short TTL for assignments
    RulePackTTL   = 30 * time.Minute  // Medium TTL for rulepack content
    CompiledTTL   = 1 * time.Hour     // Long TTL for compiled rules
)
```

#### **6. Endpoint Pattern Matching Optimization**

```go
// Optimized endpoint pattern matching
func matchesEndpointPattern(requestPath, pattern string) bool {
    // Fast path for common patterns
    switch pattern {
    case "/", "*", "":
        return true
    case requestPath:
        return true
    }
    
    // Wildcard pattern optimization
    if strings.HasSuffix(pattern, "/*") {
        base := strings.TrimSuffix(pattern, "/*")
        return strings.HasPrefix(requestPath, base+"/") || requestPath == base
    }
    
    return requestPath == pattern
}

// Pre-compiled pattern matcher for high-traffic endpoints
type PatternMatcher struct {
    exactMatches   map[string]bool
    prefixMatches  []string
    wildcardMatches []string
}

func (pm *PatternMatcher) Match(path string) bool {
    // O(1) exact match
    if pm.exactMatches[path] {
        return true
    }
    
    // O(n) prefix match
    for _, prefix := range pm.prefixMatches {
        if strings.HasPrefix(path, prefix) {
            return true
        }
    }
    
    return false
}
```

#### **7. Real-Time Performance Monitoring**

```sql
-- Real-time performance monitoring
CREATE VIEW endpoint_resolution_performance AS
SELECT 
    DATE_TRUNC('minute', created_at) as minute,
    COUNT(*) as total_requests,
    AVG(resolution_time_ms) as avg_resolution_time,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY resolution_time_ms) as p95_resolution_time,
    COUNT(*) FILTER (WHERE resolution_time_ms > 100) as slow_resolutions
FROM scan_results
WHERE created_at > NOW() - INTERVAL '1 hour'
GROUP BY DATE_TRUNC('minute', created_at)
ORDER BY minute DESC;

-- Assignment resolution performance
CREATE VIEW assignment_resolution_stats AS
SELECT 
    tenant_id,
    COUNT(*) as total_assignments,
    AVG(rulepack_count) as avg_rulepacks_per_request,
    MAX(rulepack_count) as max_rulepacks_per_request,
    AVG(resolution_time_ms) as avg_resolution_time
FROM (
    SELECT 
        tenant_id,
        COUNT(DISTINCT rulepack_id) as rulepack_count,
        resolution_time_ms
    FROM scan_results
    WHERE created_at > NOW() - INTERVAL '1 day'
    GROUP BY tenant_id, request_id, resolution_time_ms
) t
GROUP BY tenant_id;
```

#### **8. Performance Targets for Real-Time Resolution**

```yaml
# Real-time performance targets
PerformanceTargets:
  AssignmentResolution:
    P50: "< 5ms"
    P95: "< 15ms"
    P99: "< 50ms"
  
  RulePackLoading:
    P50: "< 10ms"
    P95: "< 30ms"
    P99: "< 100ms"
  
  TotalResolution:
    P50: "< 20ms"
    P95: "< 50ms"
    P99: "< 150ms"
  
  CacheHitRate:
    AssignmentCache: "> 95%"
    RulePackCache: "> 90%"
    CompiledCache: "> 85%"
```

#### **9. Scaling Strategies**

```go
// Horizontal scaling for assignment resolution
type AssignmentResolver struct {
    // Shard assignments by tenant
    shards map[int]*AssignmentShard
    
    // Load balancer for high-traffic tenants
    loadBalancer *LoadBalancer
    
    // Circuit breaker for database protection
    circuitBreaker *CircuitBreaker
}

// Tenant-based sharding
func (ar *AssignmentResolver) getShard(tenantID uuid.UUID) *AssignmentShard {
    shardID := int(tenantID[0]) % len(ar.shards)
    return ar.shards[shardID]
}

// Circuit breaker for database protection
func (ar *AssignmentResolver) resolveWithCircuitBreaker(ctx context.Context, tenantID uuid.UUID) ([]*RulepackAssignment, error) {
    return ar.circuitBreaker.Execute(func() ([]*RulepackAssignment, error) {
        return ar.assignmentRepo.ListByTenant(ctx, tenantID)
    })
}
```

#### **10. Real-Time Optimization Checklist**

```sql
-- Database optimizations
CREATE INDEX CONCURRENTLY idx_assignments_tenant_priority_enabled 
ON rulepack_assignments(tenant_id, priority DESC) 
WHERE enabled = true;

CREATE INDEX CONCURRENTLY idx_rulepacks_active_tenant_id 
ON rulepacks(tenant_id, id) 
WHERE is_active = true;

-- Redis cache warming
INSERT INTO cache_warming_jobs (tenant_id, job_type, status) 
SELECT DISTINCT tenant_id, 'assignments', 'pending' 
FROM rulepack_assignments 
WHERE enabled = true;

-- Performance monitoring
SELECT 
    schemaname,
    tablename,
    attname,
    n_distinct,
    correlation
FROM pg_stats 
WHERE tablename IN ('rulepack_assignments', 'rulepacks', 'rulepack_versions')
ORDER BY n_distinct DESC;
```

## 🛠️ User-Defined Tools (Allow/Deny Lists) Performance Impact

### **Tool Policy Architecture Overview**

Users define tool allow/deny lists through the UI, which are stored as JSONB policies in `tenant_settings` and applied in real-time during tool invocation requests.

#### **1. Tool Policy Storage Structure**

```sql
-- TOOLS: Tool registry and metadata
CREATE TABLE tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    tool_id TEXT NOT NULL,                    -- Unique tool identifier
    name TEXT NOT NULL,
    description TEXT DEFAULT '',
    capability_tags JSONB NOT NULL DEFAULT '[]'::jsonb,  -- ["read", "write", "payment"]
    data_domains JSONB NOT NULL DEFAULT '[]'::jsonb,     -- ["pii", "financial", "health"]
    side_effect TEXT NOT NULL DEFAULT 'none',            -- none | reversible | irreversible
    auth_scope TEXT NOT NULL DEFAULT 'user-delegated',   -- user-delegated | service-account
    arg_schema JSONB DEFAULT NULL,                       -- Tool argument schema
    risk_score INT,                                      -- 1-10 risk rating
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, tool_id)
);

-- TENANT_SETTINGS: Tool policies (allow/deny lists)
CREATE TABLE tenant_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key TEXT NOT NULL,                                   -- 'tool_policies'
    value JSONB NOT NULL,                                -- Tool policy JSON
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(tenant_id, key)
);
```

#### **2. Tool Policy JSON Structure**

```json
{
  "policies": [
    {
      "scope": "/api/chat/completions",
      "methods": ["POST"],
      "allowed_tools": ["web_search", "calculator", "file_reader"],
      "require_approval": ["payment_processor", "email_sender"],
      "timeout_ms": 30000,
      "egress_allowlist": {
        "schemes": ["https"],
        "hosts": ["api.openai.com", "api.anthropic.com"],
        "paths": ["/v1/chat/completions"]
      },
      "require_roles": ["user", "admin"],
      "require_headers": {
        "X-API-Key": ["required"],
        "Authorization": ["Bearer *"]
      }
    }
  ]
}
```

#### **3. Real-Time Tool Policy Resolution Flow**

```go
// Every tool invocation request triggers this flow:
func loadMatchedToolPolicy(opt Options, r *http.Request, endpoint, method string) *toolPolicySpec {
    // 1. Get tenant ID from context
    tenantID := getTenantID(r.Context())
    
    // 2. Load tool policies from cache or database (HOT PATH)
    raw := loadPoliciesJSONFromCacheOrDB(opt, r, tenantID)
    
    // 3. Parse JSON policies
    var root map[string]any
    json.Unmarshal([]byte(raw), &root)
    
    // 4. Find best matching policy by endpoint scope
    policies := root["policies"].([]any)
    var best *toolPolicySpec
    bestLen := -1
    
    for _, policy := range policies {
        if scopeMatch(policy.Scope, endpoint) {
            if len(policy.Scope) > bestLen {
                best = policy
                bestLen = len(policy.Scope)
            }
        }
    }
    
    return best
}
```

#### **4. Database Query Performance Impact**

```sql
-- HOT PATH QUERY 1: Load tool policies (called for every tool request)
SELECT value FROM tenant_settings 
WHERE key='tool_policies' AND tenant_id=$1 
LIMIT 1;

-- HOT PATH QUERY 2: Get tool metadata (called for tool validation)
SELECT id, tenant_id, tool_id, name, capability_tags, data_domains, 
       side_effect, auth_scope, arg_schema, risk_score
FROM tools 
WHERE tenant_id=$1 AND tool_id=$2;

-- HOT PATH QUERY 3: Update tool policies (called when policies change)
INSERT INTO tenant_settings (tenant_id, key, value)
VALUES ($1, 'tool_policies', $2::jsonb)
ON CONFLICT (tenant_id, key)
DO UPDATE SET value=EXCLUDED.value, updated_at=NOW();
```

#### **5. Performance Bottlenecks Analysis**

| **Operation** | **Frequency** | **Impact** | **Current Mitigation** |
|---------------|---------------|------------|----------------------|
| **Load Tool Policies** | Every tool request | High I/O | In-memory cache (60s TTL) |
| **JSON Parsing** | Every tool request | CPU intensive | Cached parsing |
| **Policy Matching** | Every tool request | CPU intensive | Optimized scope matching |
| **Tool Metadata Lookup** | Per tool validation | Medium I/O | No caching (needs optimization) |
| **Policy Updates** | Policy changes | Medium I/O | Cache invalidation |

#### **6. Aurora Optimization for Tool Policies**

```sql
-- Critical indexes for tool policy performance
CREATE INDEX idx_tenant_settings_tool_policies 
ON tenant_settings(tenant_id, key) 
WHERE key = 'tool_policies';

CREATE INDEX idx_tools_tenant_toolid 
ON tools(tenant_id, tool_id);

CREATE INDEX idx_tools_capability_tags_gin 
ON tools USING GIN (capability_tags);

CREATE INDEX idx_tools_data_domains_gin 
ON tools USING GIN (data_domains);

-- Partial index for active tools only
CREATE INDEX idx_tools_active_tenant 
ON tools(tenant_id, tool_id, risk_score) 
WHERE risk_score IS NOT NULL;
```

#### **7. Caching Strategy for Tool Policies**

```go
// Multi-level caching for tool policies
type ToolPolicyCache struct {
    // L1: In-memory cache (per tenant)
    policyCache map[string]struct {
        policies *toolPolicySpec
        raw      string
        at       time.Time
    }
    
    // L2: Redis cache (shared across instances)
    redisCache *redis.Client
    
    // L3: Database (source of truth)
    db *sql.DB
}

// Cache TTL strategy
const (
    ToolPolicyTTL = 60 * time.Second    // Short TTL for policy changes
    ToolMetadataTTL = 5 * time.Minute   // Medium TTL for tool metadata
    PolicyEpochTTL = 1 * time.Hour      // Long TTL for policy epochs
)

// Cache invalidation on policy updates
func (tpc *ToolPolicyCache) InvalidatePolicy(tenantID uuid.UUID) {
    // Invalidate in-memory cache
    delete(tpc.policyCache, tenantID.String())
    
    // Invalidate Redis cache
    tpc.redisCache.Del(ctx, fmt.Sprintf("tool_policies:%s", tenantID))
    
    // Update policy epoch for global invalidation
    tpc.updatePolicyEpoch()
}
```

#### **8. Tool Policy Performance Monitoring**

```sql
-- Tool policy resolution performance
CREATE VIEW tool_policy_performance AS
SELECT 
    DATE_TRUNC('minute', created_at) as minute,
    COUNT(*) as total_tool_requests,
    AVG(resolution_time_ms) as avg_resolution_time,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY resolution_time_ms) as p95_resolution_time,
    COUNT(*) FILTER (WHERE resolution_time_ms > 50) as slow_resolutions
FROM tool_requests
WHERE created_at > NOW() - INTERVAL '1 hour'
GROUP BY DATE_TRUNC('minute', created_at)
ORDER BY minute DESC;

-- Tool policy cache effectiveness
CREATE VIEW tool_policy_cache_stats AS
SELECT 
    tenant_id,
    COUNT(*) as total_requests,
    COUNT(*) FILTER (WHERE cache_hit = true) as cache_hits,
    ROUND(COUNT(*) FILTER (WHERE cache_hit = true) * 100.0 / COUNT(*), 2) as cache_hit_rate,
    AVG(resolution_time_ms) as avg_resolution_time
FROM tool_requests
WHERE created_at > NOW() - INTERVAL '1 day'
GROUP BY tenant_id;
```

#### **9. Performance Targets for Tool Policies**

```yaml
# Tool policy performance targets
ToolPolicyTargets:
  PolicyResolution:
    P50: "< 5ms"
    P95: "< 15ms"
    P99: "< 30ms"
  
  ToolMetadataLookup:
    P50: "< 2ms"
    P95: "< 5ms"
    P99: "< 10ms"
  
  TotalToolRequest:
    P50: "< 10ms"
    P95: "< 25ms"
    P99: "< 50ms"
  
  CacheHitRate:
    PolicyCache: "> 95%"
    ToolMetadataCache: "> 90%"
```

#### **10. Tool Policy Storage Growth Projections**

```sql
-- Conservative estimates per tenant:
-- 10 tools × 1KB metadata = 10KB
-- 5 policy rules × 2KB JSON = 10KB
-- 100 tenants × 20KB = 2MB
-- 1000 tenants × 20KB = 20MB
-- 10,000 tenants × 20KB = 200MB

-- With Aurora compression (70%):
-- 200MB → 60MB actual storage
-- Minimal impact compared to rulepacks
```

#### **11. Tool Policy Optimization Checklist**

```sql
-- Database optimizations
CREATE INDEX CONCURRENTLY idx_tenant_settings_tool_policies_covering 
ON tenant_settings(tenant_id, key, value) 
WHERE key = 'tool_policies';

CREATE INDEX CONCURRENTLY idx_tools_tenant_capabilities 
ON tools(tenant_id, capability_tags, risk_score);

-- Cache warming for high-traffic tenants
INSERT INTO cache_warming_jobs (tenant_id, job_type, status) 
SELECT DISTINCT tenant_id, 'tool_policies', 'pending' 
FROM tenant_settings 
WHERE key = 'tool_policies';

-- Performance monitoring
SELECT 
    schemaname,
    tablename,
    attname,
    n_distinct,
    correlation
FROM pg_stats 
WHERE tablename IN ('tools', 'tenant_settings')
ORDER BY n_distinct DESC;
```

#### **12. Real-Time Tool Enforcement Flow**

```go
// Complete tool enforcement flow
func enforceToolPolicy(ctx context.Context, req *ToolRequest) (*ToolDecision, error) {
    // 1. Load tool policies (cached)
    policies := loadMatchedToolPolicy(ctx, req.Endpoint, req.Method)
    
    // 2. Check tool allowlist
    if policies != nil && len(policies.AllowedTools) > 0 {
        if !contains(policies.AllowedTools, req.ToolID) {
            return &ToolDecision{Allow: false, Reason: "tool_not_in_allowlist"}, nil
        }
    }
    
    // 3. Check approval requirements
    if policies != nil && contains(policies.RequireApproval, req.ToolID) {
        if !hasApproval(ctx, req.ToolID, req.UserID) {
            return &ToolDecision{Allow: false, Reason: "approval_required"}, nil
        }
    }
    
    // 4. Load tool metadata
    tool, err := getToolMetadata(ctx, req.TenantID, req.ToolID)
    if err != nil {
        return &ToolDecision{Allow: false, Reason: "tool_not_found"}, nil
    }
    
    // 5. Check capability restrictions
    if hasRestrictedCapabilities(tool, policies) {
        return &ToolDecision{Allow: false, Reason: "capability_restricted"}, nil
    }
    
    // 6. Check egress allowlist
    if policies != nil && !isEgressAllowed(req.TargetURL, policies.EgressAllowlist) {
        return &ToolDecision{Allow: false, Reason: "egress_not_allowed"}, nil
    }
    
    return &ToolDecision{Allow: true, Reason: "ok"}, nil
}
```

## 🎯 Priority & Progression in Enforcement

### **Multi-Layer Priority System Overview**

Your enforcement system uses a sophisticated multi-layer priority and progression system that determines how rulepacks and tool policies are applied and how decisions are made.

#### **1. RulePack Assignment Priority**

```sql
-- RulePack assignments are ordered by priority (higher priority first)
SELECT id, tenant_id, rulepack_id, target_scope, method, priority, enabled, created_at, updated_at 
FROM rulepack_assignments 
WHERE tenant_id = $1 
ORDER BY priority DESC, created_at ASC;  -- Higher priority wins, then creation order
```

**Priority Resolution Logic:**
- **Higher priority number = Higher precedence**
- **Same priority = First created wins** (deterministic fallback)
- **Enabled assignments only** (disabled assignments are skipped)
- **Endpoint pattern matching** (exact match > wildcard match)

#### **2. RulePack Composition Priority**

```go
// RulePack composition strategies
type CompositionStrategy string

const (
    CompositionFirstMatch    = "first_match"     // First matching rule wins
    CompositionPriorityOrder = "priority_order"  // Priority-based composition
)

// Priority-based rule merging
func MergePacksPriorityOrder(packs []RulePack) []Rule {
    // Sort packs by priority (higher priority first)
    sort.Slice(sorted, func(i, j int) bool {
        pi := getPriority(sorted[i])
        pj := getPriority(sorted[j])
        if pi != pj {
            return pi > pj // Higher priority first
        }
        return sorted[i].Metadata.Name < sorted[j].Metadata.Name // Deterministic fallback
    })
    
    // First-wins: keep first encountered rule ID
    // Later duplicates are ignored
}
```

#### **3. Enforcement Mode Progression**

```go
// Enforcement modes in order of severity
type EnforcementMode string

const (
    EnforcementModeObserve    = "observe"    // Log violations but allow
    EnforcementModeRedact     = "redact"     // Redact sensitive content
    EnforcementModeQuarantine = "quarantine" // Block with review option
    EnforcementModeEnforce    = "enforce"    // Block immediately
)
```

**Enforcement Mode Hierarchy:**
1. **Observe** → Log only, never block
2. **Redact** → Allow but mutate content
3. **Quarantine** → Block with review option
4. **Enforce** → Block immediately

#### **4. Decision Progression Flow**

```go
// Complete decision progression logic
func makeEnforcementDecision(res *ScanResult, enforcementMode EnforcementMode) (string, string) {
    decision := "allow"
    reason := "no_signals"
    
    // 1. Check for explicit deny/block actions
    anyDeny := false
    anyQuarantine := false
    firstRule := ""
    
    for _, v := range res.Violations {
        if firstRule == "" {
            firstRule = v.RuleID
        }
        switch v.ResponseAction {
        case "deny", "block":
            anyDeny = true
        case "quarantine":
            anyQuarantine = true
        }
    }
    
    // 2. Apply rule-based decisions (highest priority)
    if anyDeny {
        decision = "deny"
        reason = firstRule
    } else if anyQuarantine {
        decision = "quarantine"
        reason = firstRule
    } else {
        // 3. Apply threshold-based decisions
        finalScore, _, _ := computeFinalScore(res)
        threshold := parseFloatEnv("PS_BLOCK_THRESHOLD", 0.75)
        if finalScore >= threshold {
            decision = "quarantine"
            reason = "policy_bridge_threshold"
        }
    }
    
    // 4. Apply enforcement mode overrides
    switch enforcementMode {
    case "observe":
        decision = "allow" // Never block in observe mode
    case "redact":
        if decision == "quarantine" {
            decision = "redact" // Redact instead of quarantine
        }
    case "enforce":
        if decision == "quarantine" {
            decision = "deny" // Block immediately
        }
    }
    
    return decision, reason
}
```

#### **5. Tool Policy Priority System**

```go
// Tool policy resolution with priority
func loadMatchedToolPolicy(opt Options, r *http.Request, endpoint, method string) *toolPolicySpec {
    // Load all policies for tenant
    policies := loadPoliciesJSONFromCacheOrDB(opt, r, tenantID)
    
    var best *toolPolicySpec
    bestLen := -1
    
    // Find best matching policy by scope specificity
    for _, policy := range policies {
        if scopeMatch(policy.Scope, endpoint) {
            // More specific scope wins (longer scope = more specific)
            if len(policy.Scope) > bestLen {
                best = policy
                bestLen = len(policy.Scope)
            }
        }
    }
    
    return best
}
```

**Tool Policy Priority Rules:**
1. **Scope Specificity** → More specific scope wins (`/api/chat/completions` > `/api/*`)
2. **Method Matching** → Exact method match > wildcard (`POST` > `*`)
3. **Policy Order** → First matching policy wins

#### **6. Multi-Level Decision Matrix**

| **Level** | **Priority** | **Decision Logic** | **Override Capability** |
|-----------|--------------|-------------------|------------------------|
| **Rule Response Actions** | Highest | `deny` > `quarantine` > `allow` | Cannot be overridden |
| **Threshold Scores** | High | Score ≥ threshold → `quarantine` | Can be overridden by enforcement mode |
| **Enforcement Mode** | Medium | `enforce` > `quarantine` > `redact` > `observe` | Can override threshold decisions |
| **Tool Policies** | Medium | Allowlist > Approval > Capability checks | Can override rule decisions |
| **Global Settings** | Low | Environment variables, system defaults | Can override all above |

#### **7. Priority Resolution Examples**

```yaml
# Example 1: RulePack Assignment Priority
assignments:
  - rulepack_id: "high-security"
    target_scope: "/api/payments/*"
    priority: 1000        # Highest priority
    enabled: true
  
  - rulepack_id: "general-security"
    target_scope: "/api/*"
    priority: 100         # Lower priority
    enabled: true

# Result: /api/payments/charge → high-security rules apply
#         /api/users/profile → general-security rules apply

# Example 2: Rule Composition Priority
rulepacks:
  - name: "enterprise-security"
    composition:
      strategy: "priority_order"
      priority: 1000
    rules:
      - id: "pii-detection"
        response:
          action: "quarantine"
  
  - name: "basic-security"
    composition:
      strategy: "priority_order"
      priority: 100
    rules:
      - id: "pii-detection"
        response:
          action: "redact"

# Result: enterprise-security rules win (higher priority)
#         pii-detection → quarantine (not redact)

# Example 3: Enforcement Mode Override
rulepack:
  enforcement_mode: "observe"
  rules:
    - id: "malware-detection"
      response:
        action: "deny"

# Result: malware-detection → allow (observe mode overrides deny)
```

#### **8. Database Optimization for Priority Resolution**

```sql
-- Optimized indexes for priority-based queries
CREATE INDEX idx_rulepack_assignments_priority_scope 
ON rulepack_assignments(tenant_id, priority DESC, target_scope, enabled) 
WHERE enabled = true;

-- Covering index for assignment resolution
CREATE INDEX idx_assignments_covering_priority 
ON rulepack_assignments(tenant_id, priority DESC, target_scope, method, rulepack_id, enabled);

-- Partial index for high-priority assignments
CREATE INDEX idx_high_priority_assignments 
ON rulepack_assignments(tenant_id, target_scope, rulepack_id) 
WHERE priority >= 500 AND enabled = true;
```

#### **9. Performance Impact of Priority Resolution**

| **Operation** | **Complexity** | **Impact** | **Optimization** |
|---------------|----------------|------------|------------------|
| **Assignment Sorting** | O(n log n) | Medium | Pre-sorted in database |
| **Rule Composition** | O(n²) | High | Cached composition |
| **Policy Matching** | O(n) | Medium | Indexed lookups |
| **Decision Progression** | O(1) | Low | In-memory logic |

#### **10. Priority Monitoring & Debugging**

```sql
-- Priority resolution monitoring
CREATE VIEW priority_resolution_stats AS
SELECT 
    tenant_id,
    COUNT(*) as total_assignments,
    COUNT(*) FILTER (WHERE priority >= 1000) as high_priority,
    COUNT(*) FILTER (WHERE priority >= 500) as medium_priority,
    COUNT(*) FILTER (WHERE priority < 500) as low_priority,
    AVG(priority) as avg_priority,
    MAX(priority) as max_priority
FROM rulepack_assignments
WHERE enabled = true
GROUP BY tenant_id;

-- Decision progression tracking
CREATE VIEW decision_progression_stats AS
SELECT 
    DATE_TRUNC('hour', created_at) as hour,
    decision,
    reason,
    COUNT(*) as count,
    AVG(resolution_time_ms) as avg_resolution_time
FROM scan_results
WHERE created_at > NOW() - INTERVAL '24 hours'
GROUP BY DATE_TRUNC('hour', created_at), decision, reason
ORDER BY hour DESC, count DESC;
```

#### **11. Priority Configuration Best Practices**

```yaml
# Recommended priority ranges
PriorityRanges:
  SystemCritical: 10000-9999    # System-level security
  Enterprise: 1000-999          # Enterprise-wide policies
  Department: 100-99            # Department-specific policies
  Project: 10-9                 # Project-specific policies
  User: 1-0                     # User-defined policies

# Enforcement mode progression
EnforcementProgression:
  Development: "observe"        # Log everything, block nothing
  Staging: "redact"            # Redact sensitive content
  Production: "quarantine"     # Block with review
  Critical: "enforce"          # Block immediately
```

#### **12. Priority Conflict Resolution**

```go
// Priority conflict resolution logic
func resolvePriorityConflicts(assignments []RulepackAssignment) []RulepackAssignment {
    // Sort by priority (highest first)
    sort.Slice(assignments, func(i, j int) bool {
        return assignments[i].Priority > assignments[j].Priority
    })
    
    // Remove duplicates (same rulepack_id)
    seen := make(map[uuid.UUID]bool)
    var result []RulepackAssignment
    
    for _, assignment := range assignments {
        if !seen[assignment.RulepackID] {
            seen[assignment.RulepackID] = true
            result = append(result, assignment)
        }
    }
    
    return result
}
```

This priority and progression system ensures that:
1. **Higher priority rulepacks** take precedence over lower priority ones
2. **More specific endpoint matches** win over general ones
3. **Enforcement modes** can override rule decisions when appropriate
4. **Tool policies** can override rule-based decisions
5. **System maintains deterministic behavior** even with complex priority hierarchies

## 🚀 High-Leverage Performance Optimizations

### **Hot Path Optimizations (Endpoint → RulePack Resolution)**

#### **1. Precompiled Scope Matcher**

```go
// Path trie for O(path) lookup instead of N pattern checks
type ScopeTrie struct {
    Children map[string]*ScopeTrie `json:"children"`
    Methods  map[string][]uuid.UUID `json:"methods"` // method -> rulepack_ids
    Wildcard []uuid.UUID           `json:"wildcard"` // catch-all rulepacks
}

// Per-tenant scope trie cache
type TenantScopeCache struct {
    TenantID    uuid.UUID           `json:"tenant_id"`
    ScopeTrie   *ScopeTrie          `json:"scope_trie"`
    CompiledAt  time.Time           `json:"compiled_at"`
    Version     int64               `json:"version"`
}

// Compile assignments into trie
func compileScopeTrie(assignments []RulepackAssignment) *ScopeTrie {
    trie := &ScopeTrie{
        Children: make(map[string]*ScopeTrie),
        Methods:  make(map[string][]uuid.UUID),
        Wildcard: []uuid.UUID{},
    }
    
    for _, assignment := range assignments {
        if !assignment.Enabled {
            continue
        }
        
        // Parse scope pattern: /api/v1/tools/* -> ["api", "v1", "tools", "*"]
        segments := parseScopeSegments(assignment.TargetScope)
        
        current := trie
        for i, segment := range segments {
            if segment == "*" {
                // Wildcard at this level
                if assignment.Method == "*" {
                    current.Wildcard = append(current.Wildcard, assignment.RulepackID)
                } else {
                    if current.Methods[assignment.Method] == nil {
                        current.Methods[assignment.Method] = []uuid.UUID{}
                    }
                    current.Methods[assignment.Method] = append(current.Methods[assignment.Method], assignment.RulepackID)
                }
                break
            }
            
            // Create child if doesn't exist
            if current.Children[segment] == nil {
                current.Children[segment] = &ScopeTrie{
                    Children: make(map[string]*ScopeTrie),
                    Methods:  make(map[string][]uuid.UUID),
                    Wildcard: []uuid.UUID{},
                }
            }
            current = current.Children[segment]
        }
    }
    
    return trie
}

// O(path) lookup instead of O(n) pattern matching
func (t *ScopeTrie) findRulepacks(path string, method string) []uuid.UUID {
    segments := strings.Split(strings.Trim(path, "/"), "/")
    return t.findRulepacksRecursive(segments, method, 0)
}

func (t *ScopeTrie) findRulepacksRecursive(segments []string, method string, depth int) []uuid.UUID {
    if depth >= len(segments) {
        // Exact match found
        var result []uuid.UUID
        if t.Methods[method] != nil {
            result = append(result, t.Methods[method]...)
        }
        if t.Methods["*"] != nil {
            result = append(result, t.Methods["*"]...)
        }
        result = append(result, t.Wildcard...)
        return result
    }
    
    segment := segments[depth]
    var result []uuid.UUID
    
    // Try exact match
    if child, exists := t.Children[segment]; exists {
        result = append(result, child.findRulepacksRecursive(segments, method, depth+1)...)
    }
    
    // Try wildcard match
    if child, exists := t.Children["*"]; exists {
        result = append(result, child.findRulepacksRecursive(segments, method, depth+1)...)
    }
    
    return result
}
```

**Performance Impact:**
- **Before**: O(n) pattern matching for each request
- **After**: O(path_length) trie traversal
- **Memory**: ~1KB per tenant (compact JSON blob)
- **Cache Hit**: 99.9% (invalidated only on assignment changes)

#### **2. Two-Stage Candidate Filter**

```go
// Rulepack fingerprint for fast pre-filtering
type RulepackFingerprint struct {
    RulepackID  uuid.UUID `json:"rulepack_id"`
    Fingerprint []byte    `json:"fingerprint"` // 64-256 bit bloom filter
    Version     int64     `json:"version"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// Compute fingerprint from compiled rules
func computeRulepackFingerprint(rules []Rule) []byte {
    hasher := sha256.New()
    
    for _, rule := range rules {
        // Hash rule patterns and keywords
        for _, pattern := range rule.Patterns {
            hasher.Write([]byte(pattern.Regex))
            hasher.Write([]byte(pattern.Keyword))
        }
        
        // Hash rule metadata
        hasher.Write([]byte(rule.ID))
        hasher.Write([]byte(rule.Severity))
    }
    
    // Create bloom filter from hash
    bloom := bloom.New(256, 3) // 256 bits, 3 hash functions
    bloom.Add(hasher.Sum(nil))
    
    return bloom.Bytes()
}

// Fast pre-filtering before DB load
func filterCandidatesByFingerprint(payload []byte, fingerprints []RulepackFingerprint) []uuid.UUID {
    payloadFingerprint := computePayloadFingerprint(payload)
    var candidates []uuid.UUID
    
    for _, fp := range fingerprints {
        if bloomFilterMatch(payloadFingerprint, fp.Fingerprint) {
            candidates = append(candidates, fp.RulepackID)
        }
    }
    
    return candidates
}
```

**Performance Impact:**
- **Filter Rate**: >90% of rulepacks eliminated before DB load
- **Memory**: ~32 bytes per rulepack fingerprint
- **CPU**: O(1) bloom filter check per rulepack

#### **3. Compiled Automata Cache**

```go
// Binary blob storage for compiled automata
type CompiledAutomata struct {
    RulepackID    uuid.UUID `json:"rulepack_id"`
    Version       int64     `json:"version"`
    BlobKey       string    `json:"blob_key"`       // S3/object storage key
    BlobSize      int64     `json:"blob_size"`
    CompiledAt    time.Time `json:"compiled_at"`
    MemorySize    int64     `json:"memory_size"`
    Checksum      string    `json:"checksum"`
}

// Store compiled automata in object storage
func storeCompiledAutomata(rulepackID uuid.UUID, version int64, automata *CompiledAutomata) error {
    // Serialize automata to binary blob
    blob, err := serializeAutomata(automata)
    if err != nil {
        return err
    }
    
    // Compress with zstd
    compressed, err := compressZstd(blob)
    if err != nil {
        return err
    }
    
    // Store in S3 with content-addressed key
    key := fmt.Sprintf("automata/%s/v%d/%s.zst", rulepackID, version, sha256.Sum256(compressed))
    
    return s3Client.PutObject(key, compressed)
}

// Load and pin in memory
func loadCompiledAutomata(rulepackID uuid.UUID, version int64) (*CompiledAutomata, error) {
    // Check memory cache first
    if cached := memoryCache.Get(rulepackID); cached != nil {
        return cached, nil
    }
    
    // Load from object storage
    blob, err := s3Client.GetObject(fmt.Sprintf("automata/%s/v%d/", rulepackID, version))
    if err != nil {
        return nil, err
    }
    
    // Decompress and deserialize
    decompressed, err := decompressZstd(blob)
    if err != nil {
        return nil, err
    }
    
    automata, err := deserializeAutomata(decompressed)
    if err != nil {
        return nil, err
    }
    
    // Pin in memory cache
    memoryCache.Set(rulepackID, automata)
    
    return automata, nil
}
```

**Performance Impact:**
- **DB Query**: Only metadata lookup (pointer to blob)
- **CPU**: Skip compilation on hot path
- **Memory**: Pin active versions only
- **Storage**: Compressed binary blobs in object storage

#### **4. Snapshot Table for Resolution**

```sql
-- Materialized endpoint template resolution
CREATE TABLE endpoint_rulepack_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    endpoint_template TEXT NOT NULL,  -- Normalized: /v1/tools/:id
    method TEXT NOT NULL,
    rulepack_ids UUID[] NOT NULL,
    priority_order INTEGER[] NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for O(1) lookup
CREATE INDEX idx_endpoint_snapshots_lookup 
ON endpoint_rulepack_snapshots(tenant_id, endpoint_template, method);

-- Materialization function
CREATE OR REPLACE FUNCTION materialize_endpoint_snapshots()
RETURNS void AS $$
BEGIN
    -- Clear existing snapshots
    DELETE FROM endpoint_rulepack_snapshots;
    
    -- Materialize new snapshots
    INSERT INTO endpoint_rulepack_snapshots (tenant_id, endpoint_template, method, rulepack_ids, priority_order)
    SELECT 
        ra.tenant_id,
        normalize_endpoint_template(ra.target_scope) as endpoint_template,
        ra.method,
        ARRAY_AGG(ra.rulepack_id ORDER BY ra.priority DESC) as rulepack_ids,
        ARRAY_AGG(ra.priority ORDER BY ra.priority DESC) as priority_order
    FROM rulepack_assignments ra
    WHERE ra.enabled = true
    GROUP BY ra.tenant_id, normalize_endpoint_template(ra.target_scope), ra.method;
END;
$$ LANGUAGE plpgsql;

-- Normalize endpoint templates
CREATE OR REPLACE FUNCTION normalize_endpoint_template(scope TEXT)
RETURNS TEXT AS $$
BEGIN
    -- Convert /v1/tools/* to /v1/tools/:id
    -- Convert /v1/users/{id} to /v1/users/:id
    RETURN regexp_replace(
        regexp_replace(scope, '\*', ':id', 'g'),
        '\{[^}]+\}', ':id', 'g'
    );
END;
$$ LANGUAGE plpgsql;
```

**Performance Impact:**
- **Query**: Single indexed lookup instead of pattern matching
- **Complexity**: O(1) instead of O(n)
- **Maintenance**: Materialized on assignment changes

### **Storage & Versioning Optimizations**

#### **5. Content-Addressed Deduplication**

```sql
-- Content-addressed storage for YAML/DSL blobs
CREATE TABLE content_blobs (
    content_hash CHAR(64) PRIMARY KEY,  -- SHA256
    content_type TEXT NOT NULL,         -- 'yaml', 'dsl', 'automata'
    blob_size BIGINT NOT NULL,
    compressed_size BIGINT NOT NULL,
    compression_type TEXT NOT NULL,     -- 'gzip', 'zstd', 'none'
    s3_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    access_count BIGINT NOT NULL DEFAULT 0,
    last_accessed TIMESTAMPTZ
);

-- Update rulepacks to reference content hashes
ALTER TABLE rulepacks ADD COLUMN content_hash CHAR(64);
ALTER TABLE rulepack_versions ADD COLUMN content_hash CHAR(64);

-- Index for content hash lookups
CREATE INDEX idx_rulepacks_content_hash ON rulepacks(content_hash);
CREATE INDEX idx_rulepack_versions_content_hash ON rulepack_versions(content_hash);
```

```go
// Content-addressed storage service
type ContentStorage struct {
    s3Client    *s3.Client
    db          *sql.DB
    compressor  *zstd.Encoder
}

func (cs *ContentStorage) StoreContent(content []byte, contentType string) (string, error) {
    // Compute content hash
    hash := sha256.Sum256(content)
    hashStr := hex.EncodeToString(hash[:])
    
    // Check if already exists
    if exists, err := cs.contentExists(hashStr); err != nil {
        return "", err
    } else if exists {
        return hashStr, nil
    }
    
    // Compress content
    compressed, err := cs.compressor.EncodeAll(content, nil)
    if err != nil {
        return "", err
    }
    
    // Store in S3
    s3Key := fmt.Sprintf("content/%s/%s.zst", contentType, hashStr)
    if err := cs.s3Client.PutObject(s3Key, compressed); err != nil {
        return "", err
    }
    
    // Store metadata in DB
    _, err = cs.db.Exec(`
        INSERT INTO content_blobs (content_hash, content_type, blob_size, compressed_size, compression_type, s3_key)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (content_hash) DO NOTHING
    `, hashStr, contentType, len(content), len(compressed), "zstd", s3Key)
    
    return hashStr, err
}
```

**Storage Savings:**
- **Deduplication**: Multiple versions pointing to same content
- **Compression**: 60-80% size reduction with zstd
- **Efficiency**: Only store unique content once

#### **6. Structural Delta Storage**

```sql
-- Delta storage for version history
CREATE TABLE rulepack_deltas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rulepack_id UUID NOT NULL,
    from_version INTEGER NOT NULL,
    to_version INTEGER NOT NULL,
    delta_type TEXT NOT NULL,  -- 'full', 'patch', 'snapshot'
    delta_data JSONB NOT NULL, -- JSON Patch (RFC 6902) or full content
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for delta retrieval
CREATE INDEX idx_rulepack_deltas_lookup 
ON rulepack_deltas(rulepack_id, from_version, to_version);

-- Full snapshots every N versions
CREATE TABLE rulepack_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rulepack_id UUID NOT NULL,
    version INTEGER NOT NULL,
    snapshot_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unique constraint for snapshots
CREATE UNIQUE INDEX idx_rulepack_snapshots_unique 
ON rulepack_snapshots(rulepack_id, version);
```

```go
// Delta storage service
type DeltaStorage struct {
    db *sql.DB
}

func (ds *DeltaStorage) StoreVersion(rulepackID uuid.UUID, version int, content []byte) error {
    // Get previous version
    prevContent, err := ds.getVersionContent(rulepackID, version-1)
    if err != nil {
        return err
    }
    
    if prevContent == nil {
        // First version - store as full snapshot
        return ds.storeFullSnapshot(rulepackID, version, content)
    }
    
    // Compute delta
    delta, err := ds.computeDelta(prevContent, content)
    if err != nil {
        return err
    }
    
    // Store delta
    _, err = ds.db.Exec(`
        INSERT INTO rulepack_deltas (rulepack_id, from_version, to_version, delta_type, delta_data)
        VALUES ($1, $2, $3, $4, $5)
    `, rulepackID, version-1, version, "patch", delta)
    
    // Store full snapshot every 10 versions
    if version%10 == 0 {
        return ds.storeFullSnapshot(rulepackID, version, content)
    }
    
    return err
}

func (ds *DeltaStorage) computeDelta(oldContent, newContent []byte) ([]byte, error) {
    // Use JSON Patch (RFC 6902) for structured deltas
    oldJSON := make(map[string]interface{})
    newJSON := make(map[string]interface{})
    
    if err := json.Unmarshal(oldContent, &oldJSON); err != nil {
        return nil, err
    }
    if err := json.Unmarshal(newContent, &newJSON); err != nil {
        return nil, err
    }
    
    // Generate JSON Patch
    patch, err := jsonpatch.CreatePatch(oldJSON, newJSON)
    if err != nil {
        return nil, err
    }
    
    return json.Marshal(patch)
}
```

**Storage Savings:**
- **Delta Storage**: 80-95% reduction for incremental changes
- **Snapshot Strategy**: Full snapshots every N versions
- **Reconstruction**: On-demand for historical versions

#### **7. Generated Columns for Indexing**

```sql
-- Generated columns for computed metadata
ALTER TABLE rulepacks ADD COLUMN rule_count INTEGER 
GENERATED ALWAYS AS (jsonb_array_length(rules)) STORED;

ALTER TABLE rulepacks ADD COLUMN regex_count INTEGER 
GENERATED ALWAYS AS (
    SELECT COUNT(*) FROM jsonb_array_elements(rules) AS rule
    WHERE rule->'patterns' IS NOT NULL
) STORED;

ALTER TABLE rulepacks ADD COLUMN max_pattern_length INTEGER 
GENERATED ALWAYS AS (
    SELECT MAX(LENGTH(pattern->>'regex')) 
    FROM jsonb_array_elements(rules) AS rule,
         jsonb_array_elements(rule->'patterns') AS pattern
) STORED;

-- Partial indexes using generated columns
CREATE INDEX idx_rulepacks_large_rulesets 
ON rulepacks(tenant_id, created_at) 
WHERE rule_count > 100;

CREATE INDEX idx_rulepacks_complex_patterns 
ON rulepacks(tenant_id, created_at) 
WHERE max_pattern_length > 1000;

-- Guardrails using generated columns
ALTER TABLE rulepacks ADD CONSTRAINT chk_rule_count_limit 
CHECK (rule_count <= 1000);

ALTER TABLE rulepacks ADD CONSTRAINT chk_pattern_length_limit 
CHECK (max_pattern_length <= 10000);
```

**Performance Impact:**
- **Index Efficiency**: Partial indexes on computed values
- **Guardrails**: Automatic validation without parsing
- **Query Performance**: No JSONB parsing on read

### **Partitioning & Index Optimizations**

#### **8. 2D Sub-Partitioning**

```sql
-- 2D partitioning: tenant_hash + month
CREATE TABLE violations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    tenant_hash INTEGER NOT NULL,  -- hash(tenant_id) % 16
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- ... other columns
) PARTITION BY HASH (tenant_hash);

-- Create hash partitions
CREATE TABLE violations_hash_0 PARTITION OF violations FOR VALUES WITH (modulus 16, remainder 0);
CREATE TABLE violations_hash_1 PARTITION OF violations FOR VALUES WITH (modulus 16, remainder 1);
-- ... create all 16 hash partitions

-- Sub-partition each hash partition by month
CREATE TABLE violations_hash_0_2024_01 PARTITION OF violations_hash_0 
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE violations_hash_0_2024_02 PARTITION OF violations_hash_0 
FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
-- ... create monthly sub-partitions

-- Automatic partition creation function
CREATE OR REPLACE FUNCTION create_monthly_partitions()
RETURNS void AS $$
DECLARE
    hash_partition INTEGER;
    month_start DATE;
    month_end DATE;
    partition_name TEXT;
BEGIN
    month_start := DATE_TRUNC('month', CURRENT_DATE + INTERVAL '1 month');
    month_end := month_start + INTERVAL '1 month';
    
    FOR hash_partition IN 0..15 LOOP
        partition_name := format('violations_hash_%s_%s_%s', 
            hash_partition, 
            EXTRACT(YEAR FROM month_start), 
            LPAD(EXTRACT(MONTH FROM month_start)::TEXT, 2, '0'));
        
        EXECUTE format('CREATE TABLE IF NOT EXISTS %I PARTITION OF violations_hash_%s 
                       FOR VALUES FROM (%L) TO (%L)',
            partition_name, hash_partition, month_start, month_end);
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Schedule with pg_cron
SELECT cron.schedule('create-monthly-partitions', '0 0 1 * *', 'SELECT create_monthly_partitions();');
```

**Performance Impact:**
- **Query Pruning**: 2D partition elimination
- **Maintenance**: Fast vacuum/analyze on small partitions
- **RLS**: Works well with tenant-based partitioning

#### **9. BRIN for Big Time Ranges**

```sql
-- BRIN indexes for append-only time series data
CREATE INDEX idx_violations_created_at_brin 
ON violations USING BRIN (created_at);

CREATE INDEX idx_scan_results_created_at_brin 
ON scan_results USING BRIN (created_at);

CREATE INDEX idx_audit_logs_created_at_brin 
ON audit_logs USING BRIN (created_at);

-- BRIN with additional columns for better selectivity
CREATE INDEX idx_violations_tenant_created_brin 
ON violations USING BRIN (tenant_id, created_at);
```

**Performance Impact:**
- **Index Size**: 1000x smaller than B-tree
- **Scan Performance**: Excellent for large time ranges
- **Maintenance**: Minimal overhead

#### **10. Optimized JSONB Indexes**

```sql
-- GIN jsonb_path_ops for containment queries
CREATE INDEX idx_rulepacks_rules_containment 
ON rulepacks USING GIN (rules jsonb_path_ops);

-- Functional indexes for hot JSON keys
CREATE INDEX idx_rulepacks_rule_kind 
ON rulepacks ((rules->>'kind'));

CREATE INDEX idx_rulepacks_severity 
ON rulepacks ((rules->'meta'->>'severity'));

-- Partial indexes on JSONB conditions
CREATE INDEX idx_rulepacks_active_rules 
ON rulepacks (tenant_id, created_at) 
WHERE (rules->>'enabled')::boolean = true;

-- Composite index for common query patterns
CREATE INDEX idx_rulepacks_tenant_kind_severity 
ON rulepacks (tenant_id, (rules->>'kind'), (rules->'meta'->>'severity'));
```

**Performance Impact:**
- **Containment Queries**: 10x faster with jsonb_path_ops
- **Functional Indexes**: Direct access to JSON keys
- **Partial Indexes**: Smaller, faster indexes

#### **11. Trigram Assist for Scopes**

```sql
-- Enable trigram extension
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Add first path segment column
ALTER TABLE rulepack_assignments ADD COLUMN first_path_segment TEXT 
GENERATED ALWAYS AS (split_part(ltrim(target_scope, '/'), '/', 1)) STORED;

-- Trigram index for wildcard scope matching
CREATE INDEX idx_rulepack_assignments_scope_trgm 
ON rulepack_assignments USING GIN (target_scope gin_trgm_ops);

-- Composite index with first segment prefix
CREATE INDEX idx_rulepack_assignments_scope_prefix 
ON rulepack_assignments (tenant_id, first_path_segment, target_scope);
```

**Performance Impact:**
- **Wildcard Matching**: Fast trigram-based pattern matching
- **Prefix Pruning**: First segment eliminates most rows
- **Legacy Support**: Handles complex wildcard patterns

### **Read Replicas & Consistency**

#### **12. Read-Your-Writes Token**

```go
// LSN-based read consistency
type ReadConsistency struct {
    db *sql.DB
}

func (rc *ReadConsistency) GetWriteLSN() (string, error) {
    var lsn string
    err := rc.db.QueryRow("SELECT pg_current_wal_lsn()").Scan(&lsn)
    return lsn, err
}

func (rc *ReadConsistency) WaitForReplica(lsn string) error {
    // Wait for replica to catch up to LSN
    _, err := rc.db.Exec("SELECT pg_wal_replay_wait($1)", lsn)
    return err
}

// Context with LSN for read routing
type LSNContext struct {
    context.Context
    writeLSN string
}

func (rc *ReadConsistency) RouteRead(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
    if lsnCtx, ok := ctx.(*LSNContext); ok && lsnCtx.writeLSN != "" {
        // Route to primary until replica catches up
        return rc.db.QueryContext(ctx, query, args...)
    }
    
    // Route to replica
    return rc.replicaDB.QueryContext(ctx, query, args...)
}
```

**Consistency Benefits:**
- **Read-Your-Writes**: Users see their changes immediately
- **Performance**: Route to replicas when possible
- **Transparency**: Automatic LSN-based routing

#### **13. Replica Specialization**

```go
// Specialized replica routing
type ReplicaRouter struct {
    primaryDB    *sql.DB
    analyticsDB  *sql.DB  // Heavy analytics queries
    complianceDB *sql.DB  // Compliance evidence builds
    defaultDB    *sql.DB  // General read queries
}

func (rr *ReplicaRouter) RouteQuery(query string, queryType QueryType) *sql.DB {
    switch queryType {
    case QueryTypeAnalytics:
        return rr.analyticsDB
    case QueryTypeCompliance:
        return rr.complianceDB
    case QueryTypeGeneral:
        return rr.defaultDB
    default:
        return rr.primaryDB
    }
}

// Query type detection
func detectQueryType(query string) QueryType {
    query = strings.ToLower(query)
    
    if strings.Contains(query, "count(") || strings.Contains(query, "sum(") || 
       strings.Contains(query, "avg(") || strings.Contains(query, "group by") {
        return QueryTypeAnalytics
    }
    
    if strings.Contains(query, "audit") || strings.Contains(query, "compliance") ||
       strings.Contains(query, "evidence") {
        return QueryTypeCompliance
    }
    
    return QueryTypeGeneral
}
```

**Performance Benefits:**
- **Analytics Isolation**: Heavy queries don't affect primary
- **Compliance Isolation**: Evidence builds don't impact latency
- **Load Distribution**: Optimal resource utilization

### **Compliance & Immutability**

#### **14. Daily Merkle Roots**

```sql
-- Merkle root chain for tamper evidence
CREATE TABLE merkle_anchors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    date DATE NOT NULL,
    root_hash CHAR(64) NOT NULL,
    prev_root_hash CHAR(64),
    row_count BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    external_anchor TEXT,  -- External verifier reference
    UNIQUE(date)
);

-- Function to compute daily Merkle root
CREATE OR REPLACE FUNCTION compute_daily_merkle_root(target_date DATE)
RETURNS TEXT AS $$
DECLARE
    root_hash TEXT;
    prev_root TEXT;
    row_count BIGINT;
BEGIN
    -- Get previous day's root
    SELECT root_hash INTO prev_root 
    FROM merkle_anchors 
    WHERE date = target_date - INTERVAL '1 day';
    
    -- Compute hash chain for the day
    SELECT 
        encode(sha256(
            COALESCE(prev_root, '') || 
            string_agg(row_hash, '' ORDER BY created_at)
        ), 'hex'),
        COUNT(*)
    INTO root_hash, row_count
    FROM audit_logs 
    WHERE DATE(created_at) = target_date;
    
    -- Store or update anchor
    INSERT INTO merkle_anchors (date, root_hash, prev_root_hash, row_count)
    VALUES (target_date, root_hash, prev_root, row_count)
    ON CONFLICT (date) DO UPDATE SET
        root_hash = EXCLUDED.root_hash,
        prev_root_hash = EXCLUDED.prev_root_hash,
        row_count = EXCLUDED.row_count;
    
    RETURN root_hash;
END;
$$ LANGUAGE plpgsql;

-- Schedule daily Merkle root computation
SELECT cron.schedule('compute-merkle-roots', '0 1 * * *', 
    'SELECT compute_daily_merkle_root(CURRENT_DATE - INTERVAL ''1 day'');');
```

**Security Benefits:**
- **Tamper Evidence**: Cryptographic chain of custody
- **External Anchoring**: Optional external verifier integration
- **Audit Trail**: Immutable evidence for compliance

#### **15. Immutable Partition Locks**

```sql
-- Partition lock table
CREATE TABLE partition_locks (
    partition_name TEXT PRIMARY KEY,
    locked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_by TEXT NOT NULL,
    lock_reason TEXT,
    revocable BOOLEAN NOT NULL DEFAULT true
);

-- Function to lock partition
CREATE OR REPLACE FUNCTION lock_partition(partition_name TEXT, locked_by TEXT, reason TEXT)
RETURNS void AS $$
BEGIN
    INSERT INTO partition_locks (partition_name, locked_by, lock_reason)
    VALUES (partition_name, locked_by, reason)
    ON CONFLICT (partition_name) DO UPDATE SET
        locked_at = NOW(),
        locked_by = EXCLUDED.locked_by,
        lock_reason = EXCLUDED.lock_reason;
END;
$$ LANGUAGE plpgsql;

-- Function to check if partition is locked
CREATE OR REPLACE FUNCTION is_partition_locked(partition_name TEXT)
RETURNS BOOLEAN AS $$
DECLARE
    locked BOOLEAN;
BEGIN
    SELECT EXISTS(
        SELECT 1 FROM partition_locks 
        WHERE partition_name = $1 AND revocable = true
    ) INTO locked;
    
    RETURN locked;
END;
$$ LANGUAGE plpgsql;
```

**Compliance Benefits:**
- **Immutability**: Prevents accidental data modification
- **Audit Trail**: Track who locked what and when
- **Break-Glass**: Emergency revocation capability

### **Caching Optimizations**

#### **16. Epoch-Based Invalidation**

```go
// Epoch-based cache invalidation
type EpochCache struct {
    tenantEpochs map[uuid.UUID]int64
    mutex        sync.RWMutex
}

func (ec *EpochCache) InvalidateTenant(tenantID uuid.UUID) {
    ec.mutex.Lock()
    defer ec.mutex.Unlock()
    
    ec.tenantEpochs[tenantID]++
}

func (ec *EpochCache) GetCacheKey(tenantID uuid.UUID, baseKey string) string {
    ec.mutex.RLock()
    epoch := ec.tenantEpochs[tenantID]
    ec.mutex.RUnlock()
    
    return fmt.Sprintf("%s:tenant:%s:epoch:%d", baseKey, tenantID, epoch)
}

// Single write invalidates all cache tiers
func (ec *EpochCache) OnAssignmentChange(tenantID uuid.UUID) {
    ec.InvalidateTenant(tenantID)
    // All cache keys for this tenant are now invalid
}
```

**Performance Benefits:**
- **Single Write**: One epoch bump invalidates all tiers
- **No Fan-Out**: No need to track individual cache keys
- **Deterministic**: Same epoch = same cache state

#### **17. Cost-Aware TTLs**

```go
// Cost-aware TTL management
type CostAwareTTL struct {
    baseTTL    time.Duration
    hotTTL     time.Duration
    coldTTL    time.Duration
    hitRatios  map[string]float64
    mutex      sync.RWMutex
}

func (ct *CostAwareTTL) GetTTL(key string) time.Duration {
    ct.mutex.RLock()
    hitRatio := ct.hitRatios[key]
    ct.mutex.RUnlock()
    
    if hitRatio > 0.8 {
        return ct.hotTTL  // Extend TTL for hot keys
    } else if hitRatio < 0.2 {
        return ct.coldTTL // Shrink TTL for cold keys
    }
    
    return ct.baseTTL
}

func (ct *CostAwareTTL) UpdateHitRatio(key string, hits, misses int) {
    ct.mutex.Lock()
    defer ct.mutex.Unlock()
    
    total := hits + misses
    if total > 0 {
        ct.hitRatios[key] = float64(hits) / float64(total)
    }
}
```

**Performance Benefits:**
- **Hot Key Optimization**: Longer TTL for frequently accessed data
- **Cold Key Cleanup**: Shorter TTL for rarely accessed data
- **Adaptive**: TTL adjusts based on access patterns

### **Write Path & Background Work**

#### **18. Async Compile Pipeline**

```go
// Async compilation pipeline
type CompilePipeline struct {
    queue    chan CompileJob
    workers  int
    s3Client *s3.Client
    db       *sql.DB
}

type CompileJob struct {
    RulepackID uuid.UUID
    Version    int
    Content    []byte
    Priority   int
}

func (cp *CompilePipeline) EnqueueCompile(rulepackID uuid.UUID, version int, content []byte) error {
    job := CompileJob{
        RulepackID: rulepackID,
        Version:    version,
        Content:    content,
        Priority:   1,
    }
    
    select {
    case cp.queue <- job:
        return nil
    default:
        return errors.New("compile queue full")
    }
}

func (cp *CompilePipeline) Start() {
    for i := 0; i < cp.workers; i++ {
        go cp.worker()
    }
}

func (cp *CompilePipeline) worker() {
    for job := range cp.queue {
        cp.processCompileJob(job)
    }
}

func (cp *CompilePipeline) processCompileJob(job CompileJob) error {
    // Acquire advisory lock to prevent stampedes
    lockKey := fmt.Sprintf("compile_%s_%d", job.RulepackID, job.Version)
    
    if err := cp.acquireAdvisoryLock(lockKey); err != nil {
        return err
    }
    defer cp.releaseAdvisoryLock(lockKey)
    
    // Compile rules
    rules, err := cp.compileRules(job.Content)
    if err != nil {
        return err
    }
    
    // Compute fingerprint
    fingerprint := cp.computeFingerprint(rules)
    
    // Store compiled automata
    blobKey, err := cp.storeCompiledAutomata(job.RulepackID, job.Version, rules)
    if err != nil {
        return err
    }
    
    // Update database with compiled metadata
    return cp.updateCompiledMetadata(job.RulepackID, job.Version, fingerprint, blobKey)
}
```

**Performance Benefits:**
- **Non-Blocking**: UI returns immediately
- **Background Processing**: Compilation happens async
- **Stampede Prevention**: Advisory locks prevent duplicate work

#### **19. UNLOGGED Staging for Bursty Telemetry**

```sql
-- UNLOGGED staging table for high-rate telemetry
CREATE UNLOGGED TABLE scan_results_staging (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    endpoint TEXT NOT NULL,
    method TEXT NOT NULL,
    decision TEXT NOT NULL,
    reason TEXT,
    scan_time_ms INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Batch upsert function
CREATE OR REPLACE FUNCTION upsert_scan_results_batch()
RETURNS void AS $$
BEGIN
    -- Batch upsert from staging to partitioned table
    INSERT INTO scan_results (tenant_id, endpoint, method, decision, reason, scan_time_ms, created_at)
    SELECT tenant_id, endpoint, method, decision, reason, scan_time_ms, created_at
    FROM scan_results_staging
    ON CONFLICT (id) DO NOTHING;
    
    -- Clear staging table
    TRUNCATE scan_results_staging;
END;
$$ LANGUAGE plpgsql;

-- Schedule batch upserts
SELECT cron.schedule('upsert-scan-results', '*/5 * * * *', 'SELECT upsert_scan_results_batch();');
```

**Performance Benefits:**
- **High Throughput**: UNLOGGED tables are much faster
- **Batch Processing**: Periodic upserts to logged tables
- **Durability Trade-off**: Configurable for dev vs prod

### **Operational Guardrails**

#### **20. Enforce Single Active Version**

```sql
-- Unique partial index to prevent dual-active rulepacks
CREATE UNIQUE INDEX idx_rulepacks_single_active 
ON rulepacks (tenant_id, rulepack_id) 
WHERE status = 'active';

-- Function to safely activate version
CREATE OR REPLACE FUNCTION activate_rulepack_version(
    p_tenant_id UUID,
    p_rulepack_id UUID,
    p_version INTEGER
) RETURNS BOOLEAN AS $$
BEGIN
    -- Deactivate current active version
    UPDATE rulepacks 
    SET status = 'inactive', updated_at = NOW()
    WHERE tenant_id = p_tenant_id 
      AND rulepack_id = p_rulepack_id 
      AND status = 'active';
    
    -- Activate new version
    UPDATE rulepacks 
    SET status = 'active', updated_at = NOW()
    WHERE tenant_id = p_tenant_id 
      AND rulepack_id = p_rulepack_id 
      AND version = p_version;
    
    RETURN FOUND;
END;
$$ LANGUAGE plpgsql;
```

**Safety Benefits:**
- **Single Active**: Prevents configuration errors
- **Atomic Activation**: Safe version switching
- **Audit Trail**: Track activation history

#### **21. Auto Partition Management**

```sql
-- Automatic partition management
CREATE OR REPLACE FUNCTION manage_partitions()
RETURNS void AS $$
DECLARE
    next_month DATE;
    old_month DATE;
    partition_name TEXT;
BEGIN
    -- Create next month's partitions
    next_month := DATE_TRUNC('month', CURRENT_DATE + INTERVAL '1 month');
    PERFORM create_monthly_partitions();
    
    -- Archive old partitions (older than 12 months)
    old_month := DATE_TRUNC('month', CURRENT_DATE - INTERVAL '12 months');
    
    -- Detach old partitions
    FOR partition_name IN 
        SELECT schemaname||'.'||tablename 
        FROM pg_tables 
        WHERE tablename LIKE 'violations_hash_%' 
          AND tablename LIKE '%' || to_char(old_month, 'YYYY_MM')
    LOOP
        EXECUTE format('ALTER TABLE %I DETACH PARTITION %I', 'violations', partition_name);
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Schedule partition management
SELECT cron.schedule('manage-partitions', '0 2 1 * *', 'SELECT manage_partitions();');
```

**Operational Benefits:**
- **Automated**: No manual partition management
- **Storage Optimization**: Automatic archiving of old data
- **Performance**: Keeps catalogs lean

#### **22. pg_stat_statements SLOs**

```sql
-- SLO monitoring for hot queries
CREATE VIEW query_performance_slos AS
SELECT 
    query,
    calls,
    total_time,
    mean_time,
    stddev_time,
    rows,
    100.0 * shared_blks_hit / nullif(shared_blks_hit + shared_blks_read, 0) AS hit_percent,
    CASE 
        WHEN mean_time > 100 THEN 'CRITICAL'
        WHEN mean_time > 50 THEN 'WARNING'
        ELSE 'OK'
    END as slo_status
FROM pg_stat_statements 
WHERE query LIKE '%rulepack_assignments%' 
   OR query LIKE '%rulepacks%'
   OR query LIKE '%endpoint_rulepack_snapshots%'
ORDER BY mean_time DESC;

-- Alert on SLO violations
CREATE OR REPLACE FUNCTION check_query_slos()
RETURNS TABLE(alert_level TEXT, query TEXT, mean_time NUMERIC) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        'CRITICAL'::TEXT,
        qps.query,
        qps.mean_time
    FROM query_performance_slos qps
    WHERE qps.slo_status = 'CRITICAL';
END;
$$ LANGUAGE plpgsql;
```

**Monitoring Benefits:**
- **SLO Tracking**: Monitor performance of critical queries
- **Alerting**: Automatic alerts on performance regressions
- **Optimization**: Identify queries needing optimization

### **Implementation Priority**

1. **Immediate (Week 1-2)**:
   - Precompiled scope matcher
   - Generated columns for indexing
   - Basic partitioning

2. **Short-term (Month 1)**:
   - Content-addressed deduplication
   - Epoch-based cache invalidation
   - Async compile pipeline

3. **Medium-term (Month 2-3)**:
   - 2D sub-partitioning
   - BRIN indexes
   - Read-your-writes consistency

4. **Long-term (Month 3+)**:
   - Merkle root chains
   - Replica specialization
   - Advanced monitoring

These optimizations will transform your system from a functional prototype to a production-ready, enterprise-scale platform capable of handling thousands of tenants with sub-100ms response times.

## 🔒 Multi-Tenant Isolation Strategy

### **Row Level Security (RLS) Implementation**

```sql
-- Enable RLS on all tenant-isolated tables
ALTER TABLE rulepacks ENABLE ROW LEVEL SECURITY;
ALTER TABLE violations ENABLE ROW LEVEL SECURITY;
ALTER TABLE audits ENABLE ROW LEVEL SECURITY;
-- ... (all tenant tables)

-- Tenant isolation policy
CREATE POLICY tenant_isolation_policy ON rulepacks 
FOR ALL USING (tenant_id = get_current_tenant_id() OR is_platform_admin());

-- Platform admin override policy
CREATE POLICY platform_admin_policy ON rulepacks 
FOR ALL USING (is_platform_admin());
```

### **Tenant Context Functions**

```sql
-- Set tenant context for current session
CREATE OR REPLACE FUNCTION set_tenant_context(tenant_id UUID)
RETURNS VOID AS $$
BEGIN
    PERFORM set_config('app.current_tenant_id', tenant_id::text, true);
END;
$$ LANGUAGE plpgsql;

-- Get current tenant ID
CREATE OR REPLACE FUNCTION get_current_tenant_id()
RETURNS UUID AS $$
BEGIN
    RETURN COALESCE(
        current_setting('app.current_tenant_id', true)::UUID,
        '00000000-0000-0000-0000-000000000000'::UUID
    );
END;
$$ LANGUAGE plpgsql;
```

## 📈 Aurora Optimization Strategy

### **1. Read Replica Distribution**
```sql
-- Configure read replicas for different workloads
-- Primary: Write operations, real-time queries
-- Replica 1: Analytics and reporting queries
-- Replica 2: Compliance evidence generation
-- Replica 3: Dashboard and UI queries
```

### **2. Partitioning Strategy**
```sql
-- Time-based partitioning for high-volume tables
-- Monthly partitions for violations, scan_results, audits
-- Quarterly partitions for usage_metrics, performance_metrics
-- Automatic partition creation and cleanup
```

### **3. Indexing Strategy**
```sql
-- Composite indexes for common query patterns
CREATE INDEX idx_violations_tenant_date_severity 
ON violations (tenant_id, created_at DESC, severity);

CREATE INDEX idx_scan_results_tenant_endpoint_date 
ON scan_results (tenant_id, endpoint, created_at DESC);

-- GIN indexes for JSONB columns
CREATE INDEX idx_rulepacks_rules_gin ON rulepacks USING GIN (rules);
CREATE INDEX idx_audits_metadata_gin ON audits USING GIN (metadata);
```

### **4. Aurora Serverless Configuration**
```yaml
# Aurora Serverless v2 configuration
AuroraServerless:
  MinCapacity: 0.5 ACU
  MaxCapacity: 16 ACU
  AutoPause: true
  AutoPauseDelay: 300 seconds
  ScalingPolicy: "target-tracking"
```

## 🗃️ Data Retention & Lifecycle

### **Retention Policies**

| Data Category | Hot Storage | Warm Storage | Cold Storage | Archive |
|---------------|-------------|--------------|--------------|---------|
| **Violations** | 30 days | 1 year | 3 years | 7 years |
| **Scan Results** | 7 days | 90 days | 1 year | 3 years |
| **Audit Logs** | 90 days | 1 year | 3 years | 7 years |
| **Compliance Evidence** | 1 year | 3 years | 7 years | Indefinite |
| **Usage Metrics** | 30 days | 1 year | 2 years | 7 years |
| **Performance Metrics** | 7 days | 90 days | 1 year | 2 years |

### **Automated Lifecycle Management**

```sql
-- Data retention function
CREATE OR REPLACE FUNCTION manage_data_retention()
RETURNS VOID AS $$
BEGIN
    -- Archive old violations (move to warm storage)
    INSERT INTO violations_archive 
    SELECT * FROM violations 
    WHERE created_at < NOW() - INTERVAL '30 days';
    
    DELETE FROM violations 
    WHERE created_at < NOW() - INTERVAL '30 days';
    
    -- Archive old scan results
    INSERT INTO scan_results_archive 
    SELECT * FROM scan_results 
    WHERE created_at < NOW() - INTERVAL '7 days';
    
    DELETE FROM scan_results 
    WHERE created_at < NOW() - INTERVAL '7 days';
END;
$$ LANGUAGE plpgsql;

-- Schedule retention management
SELECT cron.schedule('data-retention', '0 2 * * *', 'SELECT manage_data_retention();');
```

## 🔍 Compliance & Audit Architecture

### **Immutable Audit Trail**

```sql
-- Audit event insertion with integrity hash
CREATE OR REPLACE FUNCTION insert_audit_event(
    p_tenant_id UUID,
    p_actor_id UUID,
    p_action TEXT,
    p_object_type TEXT,
    p_object_id UUID,
    p_diff JSONB
) RETURNS UUID AS $$
DECLARE
    event_id UUID;
    integrity_hash TEXT;
BEGIN
    event_id := gen_random_uuid();
    
    -- Generate integrity hash
    integrity_hash := encode(
        digest(
            event_id::text || p_tenant_id::text || p_action || p_object_type || 
            p_object_id::text || p_diff::text || NOW()::text,
            'sha256'
        ),
        'hex'
    );
    
    INSERT INTO audits (
        id, tenant_id, actor_id, action, object_type, 
        object_id, diff, integrity_hash, created_at
    ) VALUES (
        event_id, p_tenant_id, p_actor_id, p_action, p_object_type,
        p_object_id, p_diff, integrity_hash, NOW()
    );
    
    RETURN event_id;
END;
$$ LANGUAGE plpgsql;
```

### **Compliance Evidence Generation**

```sql
-- Generate compliance evidence for specific requirements
CREATE OR REPLACE FUNCTION generate_compliance_evidence(
    p_tenant_id UUID,
    p_standard TEXT,
    p_requirement_id TEXT,
    p_time_range_start TIMESTAMPTZ,
    p_time_range_end TIMESTAMPTZ
) RETURNS UUID AS $$
DECLARE
    evidence_id UUID;
    event_count INTEGER;
    evidence_data JSONB;
    integrity_hash TEXT;
BEGIN
    evidence_id := gen_random_uuid();
    
    -- Collect relevant events
    SELECT COUNT(*), jsonb_agg(a.*)
    INTO event_count, evidence_data
    FROM audits a
    WHERE a.tenant_id = p_tenant_id
    AND a.created_at BETWEEN p_time_range_start AND p_time_range_end
    AND a.action LIKE '%' || p_requirement_id || '%';
    
    -- Generate integrity hash
    integrity_hash := encode(
        digest(
            evidence_id::text || p_tenant_id::text || p_standard || 
            p_requirement_id || evidence_data::text,
            'sha256'
        ),
        'hex'
    );
    
    INSERT INTO compliance_evidence (
        id, tenant_id, standard, requirement_id, evidence_type,
        time_range_start, time_range_end, event_count, evidence_data,
        integrity_hash, generated_at, generated_by
    ) VALUES (
        evidence_id, p_tenant_id, p_standard, p_requirement_id, 'audit_events',
        p_time_range_start, p_time_range_end, event_count, evidence_data,
        integrity_hash, NOW(), 'system'
    );
    
    RETURN evidence_id;
END;
$$ LANGUAGE plpgsql;
```

## 🚀 Performance Optimization

### **Materialized Views for Analytics**

```sql
-- Daily tenant metrics summary
CREATE MATERIALIZED VIEW tenant_metrics_daily AS
SELECT 
    tenant_id,
    DATE(created_at) as metric_date,
    COUNT(*) as total_violations,
    COUNT(*) FILTER (WHERE severity = 'CRITICAL') as critical_violations,
    COUNT(*) FILTER (WHERE severity = 'HIGH') as high_violations,
    AVG(processing_time_ms) as avg_processing_time
FROM violations
GROUP BY tenant_id, DATE(created_at);

-- Refresh materialized view daily
SELECT cron.schedule('refresh-tenant-metrics', '0 1 * * *', 
    'REFRESH MATERIALIZED VIEW tenant_metrics_daily;');
```

### **Connection Pooling Strategy**

```yaml
# PgBouncer configuration for Aurora
[databases]
promptshield = host=aurora-cluster.cluster-xyz.us-east-1.rds.amazonaws.com port=5432 dbname=promptshield

[pgbouncer]
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 25
reserve_pool_size = 5
reserve_pool_timeout = 3
```

## 🔧 Monitoring & Observability

### **Aurora Performance Insights**

```sql
-- Custom performance monitoring queries
CREATE VIEW performance_summary AS
SELECT 
    tenant_id,
    DATE(created_at) as date,
    COUNT(*) as total_operations,
    AVG(latency_ms) as avg_latency,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms) as p95_latency,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms) as p99_latency
FROM performance_metrics
GROUP BY tenant_id, DATE(created_at);
```

### **Health Checks**

```sql
-- Database health check function
CREATE OR REPLACE FUNCTION health_check()
RETURNS JSONB AS $$
DECLARE
    result JSONB;
    tenant_count INTEGER;
    active_rulepacks INTEGER;
    recent_violations INTEGER;
BEGIN
    SELECT COUNT(*) INTO tenant_count FROM tenants WHERE status = 'active';
    SELECT COUNT(*) INTO active_rulepacks FROM rulepacks WHERE is_active = true;
    SELECT COUNT(*) INTO recent_violations FROM violations WHERE created_at > NOW() - INTERVAL '1 hour';
    
    result := jsonb_build_object(
        'status', 'healthy',
        'timestamp', NOW(),
        'metrics', jsonb_build_object(
            'active_tenants', tenant_count,
            'active_rulepacks', active_rulepacks,
            'recent_violations', recent_violations
        )
    );
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;
```

## 📋 Implementation Checklist

### **Phase 1: Core Schema (Week 1)**
- [ ] Create core business tables (tenants, rulepacks, users)
- [ ] Implement RLS policies
- [ ] Set up tenant context functions
- [ ] Create basic indexes

### **Phase 2: Real-Time Operations (Week 2)**
- [ ] Create partitioned tables (violations, scan_results)
- [ ] Implement partition management
- [ ] Set up Aurora Serverless configuration
- [ ] Configure read replicas

### **Phase 3: Compliance & Audit (Week 3)**
- [ ] Create audit tables with integrity hashing
- [ ] Implement compliance evidence generation
- [ ] Set up data retention policies
- [ ] Create compliance reporting functions

### **Phase 4: Analytics & Optimization (Week 4)**
- [ ] Create materialized views
- [ ] Implement performance monitoring
- [ ] Set up automated maintenance
- [ ] Configure connection pooling

### **Phase 5: Production Readiness (Week 5)**
- [ ] Set up Aurora Global Database
- [ ] Implement backup and recovery
- [ ] Configure monitoring and alerting
- [ ] Performance testing and optimization

## 🎯 Success Metrics

### **Performance Targets**
- **Query Latency**: < 10ms for hot data, < 100ms for warm data
- **Throughput**: 10,000+ operations/second
- **Availability**: 99.99% uptime
- **Recovery Time**: < 5 minutes for point-in-time recovery

### **Compliance Targets**
- **Audit Trail Integrity**: 100% tamper-evident
- **Data Retention**: 100% policy compliance
- **Evidence Generation**: < 30 seconds for standard reports
- **Tenant Isolation**: Zero cross-tenant data leaks

### **Cost Optimization**
- **Aurora Serverless**: Auto-scale based on demand
- **Storage Optimization**: 70%+ compression for archived data
- **Query Optimization**: 90%+ cache hit rate for common queries
- **Resource Utilization**: 80%+ efficiency across all resources

This architecture provides a robust, scalable, and compliance-ready database foundation for PromptShield, optimized for Aurora PostgreSQL's capabilities while supporting our multi-tenant, real-time security platform requirements.
