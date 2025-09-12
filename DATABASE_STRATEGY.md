# PromptShield Database Strategy

## Overview

PromptShield uses a **hybrid database strategy** that scales from development to production, providing the right database solution for each environment while maintaining consistency and compatibility.

## Database Progression Strategy

### Development → Staging → Production

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

## Environment-Specific Configuration

### 1. Development Environment

**Database:** Local PostgreSQL (Docker)

**Configuration:**
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
    volumes:
      - pg-data:/var/lib/postgresql/data

  promptshield:
    environment:
      PS_PG_DSN: postgres://promptshield:example@postgres:5432/promptshield_dev?sslmode=disable
```

**Migration Strategy:**
```bash
# Use migrations/ directory for development
export PS_PG_DSN="postgres://promptshield:example@localhost:5432/promptshield_dev?sslmode=disable"
go run scripts/run-migrations.go
```

**Benefits:**
- ✅ **Fast iteration** - No network latency
- ✅ **No costs** - Runs locally
- ✅ **Full control** - Can modify schema freely
- ✅ **Offline development** - Works without internet

### 2. Staging Environment

**Database:** Supabase (PostgreSQL)

**Configuration:**
```bash
# .env.staging
PS_PG_DSN=postgres://postgres:[password]@db.[project].supabase.co:5432/postgres?sslmode=require
PS_TENANT_ID=staging-tenant-uuid
PS_ENVIRONMENT=staging
```

**Migration Strategy:**
```bash
# Use migrations_consolidated/ for staging
export PS_PG_DSN="postgres://postgres:[password]@db.[project].supabase.co:5432/postgres?sslmode=require"
go run scripts/run-consolidated-migrations.go
```

**Benefits:**
- ✅ **Shared access** - Team can access staging data
- ✅ **Free tier** - No costs for staging
- ✅ **Easy setup** - Managed PostgreSQL
- ✅ **Real cloud** - Tests cloud connectivity
- ✅ **Backup/restore** - Built-in data protection

**Supabase Free Tier Limits:**
- **Database**: 500MB
- **Bandwidth**: 2GB/month
- **Concurrent connections**: 60
- **Row Level Security**: ✅
- **Point-in-time recovery**: 7 days

### 3. Production Environment

**Database:** Aurora PostgreSQL (AWS)

**Configuration:**
```bash
# .env.production
PS_PG_DSN=postgres://[user]:[password]@[aurora-cluster-endpoint]:5432/promptshield?sslmode=require
AURORA_PROXY_ENDPOINT=[rds-proxy-endpoint]
AURORA_DB_NAME=promptshield
PS_TENANT_ID=production-tenant-uuid
PS_ENVIRONMENT=production
```

**Migration Strategy:**
```bash
# Use migrations_aurora/ for production
export AURORA_PG_DSN="postgres://[user]:[password]@[aurora-cluster-endpoint]:5432/promptshield?sslmode=require"
go run scripts/run-aurora-migrations.go
```

**Benefits:**
- ✅ **Enterprise-grade** - High availability, auto-scaling
- ✅ **Multi-AZ deployment** - 99.99% uptime
- ✅ **Automatic backups** - Point-in-time recovery
- ✅ **Performance** - Optimized for production workloads
- ✅ **Security** - VPC isolation, encryption at rest
- ✅ **Monitoring** - CloudWatch integration

## Migration Directory Strategy

### Clean Up Existing Directories

**Keep These:**
- `migrations/` - Development migrations (local PostgreSQL)
- `migrations_consolidated/` - Staging migrations (Supabase)
- `migrations_aurora/` - Production migrations (Aurora)

**Remove These:**
- Any duplicate or conflicting migration files
- Old migration files that are no longer needed

### Migration Scripts

**Development Migration Script:**
```go
// scripts/run-migrations.go
package main

import (
    "database/sql"
    "fmt"
    "os"
    "path/filepath"
)

func main() {
    dsn := os.Getenv("PS_PG_DSN")
    if dsn == "" {
        dsn = "postgres://promptshield:example@localhost:5432/promptshield_dev?sslmode=disable"
    }
    
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    // Run migrations from migrations/ directory
    runMigrations(db, "migrations")
}
```

**Staging Migration Script:**
```go
// scripts/run-consolidated-migrations.go
package main

import (
    "database/sql"
    "fmt"
    "os"
)

func main() {
    dsn := os.Getenv("PS_PG_DSN")
    if dsn == "" {
        panic("PS_PG_DSN environment variable required for staging")
    }
    
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    // Run migrations from migrations_consolidated/ directory
    runMigrations(db, "migrations_consolidated")
}
```

**Production Migration Script:**
```go
// scripts/run-aurora-migrations.go
package main

import (
    "database/sql"
    "fmt"
    "os"
)

func main() {
    dsn := os.Getenv("AURORA_PG_DSN")
    if dsn == "" {
        panic("AURORA_PG_DSN environment variable required for production")
    }
    
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        panic(err)
    }
    defer db.Close()
    
    // Run migrations from migrations_aurora/ directory
    runMigrations(db, "migrations_aurora")
}
```

## Environment Configuration

### Development Setup

**1. Start Local PostgreSQL:**
```bash
docker compose -f docker-compose.dev.yml up -d postgres
```

**2. Run Development Migrations:**
```bash
export PS_PG_DSN="postgres://promptshield:example@localhost:5432/promptshield_dev?sslmode=disable"
go run scripts/run-migrations.go
```

**3. Start Application:**
```bash
export PS_PG_DSN="postgres://promptshield:example@localhost:5432/promptshield_dev?sslmode=disable"
export PS_DEV_BYPASS_AUTH=true
make run
```

### Staging Setup

**1. Create Supabase Project:**
- Go to https://supabase.com
- Create new project
- Get connection string

**2. Configure Environment:**
```bash
# .env.staging
PS_PG_DSN=postgres://postgres:[password]@db.[project].supabase.co:5432/postgres?sslmode=require
PS_TENANT_ID=staging-tenant-uuid
PS_ENVIRONMENT=staging
```

**3. Run Staging Migrations:**
```bash
export PS_PG_DSN="postgres://postgres:[password]@db.[project].supabase.co:5432/postgres?sslmode=require"
go run scripts/run-consolidated-migrations.go
```

**4. Deploy to Staging:**
```bash
docker compose -f docker-compose.yml -f docker-compose.staging.yml up -d
```

### Production Setup

**1. Create Aurora Cluster:**
```bash
# Using Terraform or AWS CLI
aws rds create-db-cluster \
  --db-cluster-identifier promptshield-aurora \
  --engine aurora-postgresql \
  --engine-version 13.7 \
  --master-username promptshield \
  --master-user-password [secure-password]
```

**2. Configure Environment:**
```bash
# .env.production
PS_PG_DSN=postgres://promptshield:[password]@[aurora-cluster-endpoint]:5432/promptshield?sslmode=require
AURORA_PROXY_ENDPOINT=[rds-proxy-endpoint]
AURORA_DB_NAME=promptshield
PS_TENANT_ID=production-tenant-uuid
PS_ENVIRONMENT=production
```

**3. Run Production Migrations:**
```bash
export AURORA_PG_DSN="postgres://promptshield:[password]@[aurora-cluster-endpoint]:5432/promptshield?sslmode=require"
go run scripts/run-aurora-migrations.go
```

**4. Deploy to Production:**
```bash
docker compose -f docker-compose.prod.yml up -d
```

## Docker Compose Configuration

### Development (docker-compose.dev.yml)
```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: promptshield_dev
      POSTGRES_USER: promptshield
      POSTGRES_PASSWORD: example
    ports:
      - "5432:5432"
    volumes:
      - pg-data:/var/lib/postgresql/data

  promptshield:
    build: .
    environment:
      PS_PG_DSN: postgres://promptshield:example@postgres:5432/promptshield_dev?sslmode=disable
      PS_DEV_BYPASS_AUTH: "true"
    ports:
      - "9090:9090"
    depends_on:
      - postgres

volumes:
  pg-data:
```

### Staging (docker-compose.staging.yml)
```yaml
version: '3.8'
services:
  promptshield:
    build: .
    environment:
      PS_PG_DSN: ${PS_PG_DSN}
      PS_TENANT_ID: ${PS_TENANT_ID}
      PS_ENVIRONMENT: staging
    ports:
      - "9090:9090"
```

### Production (docker-compose.prod.yml)
```yaml
version: '3.8'
services:
  promptshield:
    image: ghcr.io/promptshield/enforcer:0.2.0
    environment:
      PS_PG_DSN: ${PS_PG_DSN}
      AURORA_PROXY_ENDPOINT: ${AURORA_PROXY_ENDPOINT}
      AURORA_DB_NAME: ${AURORA_DB_NAME}
      PS_TENANT_ID: ${PS_TENANT_ID}
      PS_ENVIRONMENT: production
    ports:
      - "9090:9090"
```

## Migration Strategy

### Schema Evolution

**Development:**
- **Rapid iteration** - Modify schema freely
- **Test migrations** - Validate before staging
- **Local testing** - Full control over data

**Staging:**
- **Consolidated migrations** - Clean, ordered migrations
- **Team testing** - Shared environment
- **Production-like** - Similar to production setup

**Production:**
- **Aurora-optimized** - Production-specific optimizations
- **Zero-downtime** - Careful migration planning
- **Rollback capability** - Safe deployment strategy

### Migration Best Practices

**1. Always Test Migrations:**
```bash
# Test in development first
go run scripts/run-migrations.go

# Test in staging
go run scripts/run-consolidated-migrations.go

# Deploy to production
go run scripts/run-aurora-migrations.go
```

**2. Backup Before Production:**
```bash
# Aurora automatic backups, but verify
aws rds describe-db-cluster-snapshots --db-cluster-identifier promptshield-aurora
```

**3. Monitor Migration Performance:**
```bash
# Check migration logs
docker logs promptshield-migration

# Monitor database performance
aws cloudwatch get-metric-statistics --namespace AWS/RDS
```

## Cost Analysis

### Development
- **Cost**: $0 (local PostgreSQL)
- **Benefits**: Fast iteration, full control
- **Limitations**: Single developer, no sharing

### Staging
- **Cost**: $0 (Supabase free tier)
- **Benefits**: Team access, cloud testing
- **Limitations**: 500MB storage, 2GB bandwidth

### Production
- **Cost**: $200-500/month (Aurora)
- **Benefits**: Enterprise features, high availability
- **Limitations**: Higher cost, more complex setup

## Benefits of This Strategy

### 1. **Cost Optimization**
- **Development**: Free (local)
- **Staging**: Free (Supabase free tier)
- **Production**: Pay only for enterprise features

### 2. **Team Collaboration**
- **Staging**: Shared environment for testing
- **Production**: Enterprise-grade reliability
- **Development**: Fast local iteration

### 3. **Risk Mitigation**
- **Gradual progression** from simple to complex
- **Testing at each stage** before production
- **Rollback capability** at each level

### 4. **Scalability**
- **Start simple** with local development
- **Scale up** to cloud staging
- **Enterprise production** with Aurora

## Next Steps

### 1. **Clean Up Migration Directories**
```bash
# Remove duplicate/conflicting migrations
# Keep only the three directories we need
# Update migration scripts
```

### 2. **Update Documentation**
```bash
# Update DEPLOYMENT.md with new strategy
# Update README.md with setup instructions
# Create environment-specific guides
```

### 3. **Test the Progression**
```bash
# Test development setup
# Test staging setup
# Test production setup
# Verify migration scripts work
```

### 4. **Deploy to Staging**
```bash
# Set up Supabase project
# Run staging migrations
# Deploy staging environment
# Test with team
```

This hybrid strategy gives you the best of all worlds: **fast development**, **shared staging**, and **enterprise production** - all while maintaining consistency and compatibility across environments.
