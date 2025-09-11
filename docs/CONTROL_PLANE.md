# PromptShield Control Plane

## Overview

The PromptShield Control Plane is a production-ready service for managing rulepack lifecycle, versioning, distribution, and tenant isolation. It provides a REST API for CRUD operations and real-time distribution to gateway enforcement points.

## Architecture

- **Database**: PostgreSQL with JSONB storage for rulePacks
- **Messaging**: Redis Streams for low-latency distribution 
- **Validation**: JSON Schema + semantic validation using existing rules engine
- **Multi-tenancy**: Full tenant isolation with audit trails

## Quick Start

### 1. Setup Database

```bash
# Run the migration
psql $PS_PG_DSN -f migrations/0001_init.sql
```

### 2. Environment Configuration

```bash
export PS_PG_DSN="postgres://user:pass@localhost/promptshield?sslmode=disable"
export PS_REDIS_ADDR="localhost:6379"  # Optional - no messaging if empty
export PS_CONTROL_PLANE_ADDR=":8085"   # Optional - defaults to :8085
```

### 3. Start Control Plane

```bash
go build -o bin/ps-gateway ./gateway
./bin/ps-gateway
```

The control plane functionality is now integrated into the main gateway service.

## API Reference

### Tenant Management

First, create a tenant in the database:
```sql
INSERT INTO tenants (id, name) VALUES (gen_random_uuid(), 'my-tenant');
```

### Rulepack Lifecycle

#### 1. Create Rulepack

```bash
curl -X POST http://localhost:8085/v1/rulepacks \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "your-tenant-uuid",
    "name": "security-rules-v1",
    "description": "Base security ruleset"
  }'
```

#### 2. Upload Version (Draft)

```bash
curl -X POST http://localhost:8085/v1/rulepacks/{rulepack-id}/versions \
  -H "Content-Type: application/json" \
  -d '{
    "version": 1,
    "dsl": {
      "apiVersion": "promptshield.io/v1",
      "kind": "RulePack",
      "metadata": {"name": "example"},
      "rules": [{
        "id": "test-rule",
        "level": 1,
        "keywords": ["password", "secret"]
      }]
    }
  }'
```

#### 3. Approve Version

```bash
curl -X POST http://localhost:8085/v1/rulepacks/{rulepack-id}/versions/{version}/approve
```

#### 4. Activate Version

```bash
curl -X POST http://localhost:8085/v1/rulepacks/{rulepack-id}/versions/{version}/activate \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "your-tenant-uuid",
    "dsl": { /* same DSL as upload */ }
  }'
```

#### 5. Create Assignment

```bash
curl -X POST http://localhost:8085/v1/assignments \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "your-tenant-uuid", 
    "rulepackId": "your-rulepack-uuid",
    "targetScope": "env:production",
    "priority": 100
  }'
```

### Retrieval

#### Get Active Rulepack
```bash
curl http://localhost:8085/v1/rulepacks/{rulepack-id}
```

#### Get Specific Version
```bash
curl http://localhost:8085/v1/rulepacks/{rulepack-id}/versions/{version}
```

#### Stream Updates (SSE)
```bash
curl http://localhost:8085/v1/stream
```

### Health Check

```bash
curl http://localhost:8085/healthz
```

## Data Model

### Tenants
- `id`: UUID primary key
- `name`: Unique tenant identifier
- `created_at`: Timestamp

### Rulepacks
- `id`: UUID primary key  
- `tenant_id`: Foreign key to tenants
- `name`: Rulepack name (unique per tenant)
- `description`: Optional description
- `current_version_id`: Points to active version

### Rulepack Versions
- `id`: UUID primary key
- `rulepack_id`: Foreign key to rulepacks
- `version`: Integer version number
- `dsl`: JSONB validated rulePack definition
- `status`: `draft`, `approved`, `active`, `archived`
- `created_by`: Optional user ID
- `approved_by`: Optional approver ID

### Assignments
- `id`: UUID primary key
- `tenant_id`: Foreign key to tenants
- `rulepack_id`: Foreign key to rulepacks  
- `target_scope`: Target identifier (e.g., `env:prod`, `app:web`)
- `priority`: Integer priority for ordering

### Audits
- `id`: UUID primary key
- `tenant_id`: Optional foreign key
- `actor_id`: Optional user ID
- `action`: Action performed
- `object_type`: Type of object modified
- `object_id`: ID of modified object
- `diff`: JSONB change details

## Gateway Integration

Gateways can subscribe to updates via:

1. **Redis Streams** (preferred): Subscribe to `rulepacks.updates` stream
2. **Server-Sent Events**: Connect to `/v1/stream` endpoint
3. **Polling**: Periodically GET rulepack endpoints

Updates contain:
```json
{
  "tenantId": "uuid",
  "targetScope": "env:prod", 
  "rulepackId": "uuid",
  "version": 42,
  "checksum": "sha256-hash"
}
```

## Validation

All uploaded DSL is validated against:

1. **JSON Schema**: Structural validation using `internal/rules/schema.json`
2. **Semantic Validation**: Business logic (duplicates, level requirements, etc.)
3. **Regex Safety**: Pattern complexity and safety checks
4. **Normalization**: YAML→JSON conversion for consistency

## Security Features

- **Tenant Isolation**: All data scoped by tenant ID
- **Audit Trail**: Immutable logs for compliance
- **Input Validation**: Comprehensive DSL validation
- **Graceful Degradation**: Works without Redis/messaging

## Observability

- **Health Checks**: `/healthz` endpoint
- **Structured Logging**: Request correlation and error tracking
- **Metrics**: Built-in HTTP middleware for request tracking
- **Graceful Shutdown**: Proper cleanup on SIGTERM/SIGINT

## Development Notes

- **Database**: Uses pgx/v5 for PostgreSQL connectivity
- **HTTP**: Chi router with middleware (logging, recovery, request ID)  
- **Validation**: Reuses existing `internal/rules` validation engine
- **Messaging**: Redis Streams with fallback to no-op if not configured
- **Testing**: Comprehensive validation and repository layer testing recommended