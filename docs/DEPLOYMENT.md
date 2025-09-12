# PromptShield Deployment (Clean, DeBERTa-only)

Overview
- Control plane: UI + Gateway/Enforcer + Aurora Postgres. Users define RulePacks, protected endpoints, and tools. No default RulePacks are shipped.
- Data plane (optional): Egress proxy overlay for zero‑code enforcement. Intercepts outbound HTTP from protected apps and calls the Gateway.
- Semantics: DeBERTa-only. Configure PS_DEBERTA_ENDPOINT (HF token optional; your environment does not require one).
- Telemetry: OpenTelemetry Collector enabled by default (OTLP gRPC 4317, HTTP 4318, Prometheus on 8889).

Prereqs
- Docker / Docker Compose
- Aurora PostgreSQL (prod) with PromptShield tables

Production (base)
1) Configure .env.production
- Required
  - PS_PG_DSN=postgresql://USER:PASSWORD@AURORA_ENDPOINT:5432/DB?sslmode=require
  - PS_TENANT_ID=<tenant-uuid>
  - PS_ADMIN_TOKEN=<admin-token>
  - PS_BFF_JWT_PRIVATE_KEY=<PEM private key>
  - PS_BFF_JWT_PUBLIC_KEY=<PEM public key>
  - CLERK_SECRET_KEY=<clerk-secret>
  - CLERK_PUBLISHABLE_KEY=<clerk-publishable>
  - PS_DEBERTA_ENDPOINT=http://host.docker.internal:8089/infer
- Optional
  - PS_TELEMETRY=1 (default)
  - PS_TELEMETRY_ENDPOINT=otel-collector:4317
  - PS_TRACE_SAMPLE=1.0 (0.0-1.0)
  - PS_GATEWAY_DISABLE_TRACING=false
  - PS_ENFORCER_ENFORCEMENT_MODE=enforce
  - PS_ENFORCER_TIMEOUT=300ms
  - PS_ENFORCER_MAX_BODY_BYTES=10485760

2) Start
```bash
docker compose up -d --build
```

3) Verify
```bash
# Gateway
curl -s http://localhost:9090/healthz

# OTEL metrics endpoint from collector
curl -s http://localhost:8889/metrics | head

# BFF
open http://localhost:3000
```

Development overlay
- Adds local Postgres and exposes gateway ports.
```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d --build
```

Zero‑code egress overlay (optional)
- Intercepts outbound HTTP from protected apps with no SDK or code changes.
- See docs/EGRESS_PROXY.md for details.
```bash
docker compose -f docker-compose.yml -f docker-compose.egress.yml up -d
```

Notes
- No RulePacks are shipped. All policy must be created by users in the UI and will be stored in Aurora.
- If no policies exist, the Gateway operates fail‑open by default.
- You removed Nginx and provider clusters; deployment is simplified and DeBERTa-only.

### Option 1: Supabase (Recommended)

#### Free Tier Limits:
- **Database**: 500MB
- **Bandwidth**: 2GB/month  
- **Concurrent connections**: 60
- **Row Level Security**: ✅
- **Point-in-time recovery**: 7 days

#### Setup Steps:

1. **Create Supabase Account**
   ```bash
   # Go to https://supabase.com
   # Create account and new project
   # Choose region closest to your primary users
   ```

2. **Get Connection String**
   ```bash
   # Project Settings > Database > Connection string
   # Format: postgres://postgres:[password]@[host]:5432/postgres
   ```

3. **Run Setup Script**
   ```bash
   ./scripts/setup-supabase.sh 'your-connection-string'
   ```

4. **Configure Environment**
   ```bash
   export PS_PG_DSN="postgres://postgres:password@db.yourproject.supabase.co:5432/postgres"
   export PS_REDIS_ADDR=""  # Optional - leave empty for now
   ```

### Option 2: PlanetScale (MySQL Alternative)

#### Free Tier Limits:
- **Database**: 1GB storage
- **Reads**: 1 billion/month
- **Writes**: 10 million/month
- **Global replicas**: ✅ (read-only)

⚠️ **Note**: Would require converting PostgreSQL code to MySQL

### Option 3: AWS RDS Free Tier

#### Free Tier Limits:
- **Database**: 20GB storage
- **Compute**: 750 hours/month (t3.micro)
- **Backups**: 20GB
- **Multi-AZ**: ❌ (not in free tier)

## Global Scaling Strategy

### Phase 1: Single Region (Free Tier)
```
┌─────────────────┐    ┌─────────────────┐
│   Supabase      │    │  Control Plane  │
│   (US-East)     │◄───┤  (US-East)      │
│   PostgreSQL    │    │  API Gateway    │
└─────────────────┘    └─────────────────┘
```

### Phase 2: Multi-Region Read Replicas ($25-100/month)
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Primary DB    │    │  Control Plane  │    │  Read Replica   │
│   (US-East)     │◄───┤  (US-East)      ├───►│  (EU-West)      │
│   PostgreSQL    │    │  Write/Read     │    │  Read Only      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Phase 3: Global Multi-Master ($500+/month)
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   US-East DB    │◄──►│   EU-West DB    │◄──►│   APAC DB       │
│   (Primary)     │    │   (Replica)     │    │   (Replica)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         ▲                       ▲                       ▲
         │                       │                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Control Plane  │    │  Control Plane  │    │  Control Plane  │
│  (US-East)      │    │  (EU-West)      │    │  (APAC)         │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Cost Projection

### Year 1 (MVP to Revenue)
- **Months 1-6**: Free tier (Supabase/PlanetScale) - $0
- **Months 7-12**: Pro tier - $25-50/month

### Year 2 (Growth Phase)  
- **Read replicas**: $100-300/month
- **Increased storage**: $50-100/month
- **Total**: $150-400/month

### Year 3+ (Global Scale)
- **Multi-region setup**: $500-2000/month
- **Enterprise features**: $1000-5000/month
- **Total**: $1500-7000/month

## Production Checklist

### Database Security
- [ ] Enable Row Level Security (RLS)
- [ ] Configure connection pooling
- [ ] Set up automated backups
- [ ] Enable audit logging
- [ ] Configure SSL/TLS encryption

### Application Security
- [ ] Environment variable configuration
- [ ] API rate limiting
- [ ] Request validation
- [ ] CORS configuration
- [ ] Health check endpoints

### Monitoring & Observability
- [ ] Database performance monitoring
- [ ] Application metrics (Prometheus)
- [ ] Log aggregation
- [ ] Alerting rules
- [ ] Uptime monitoring

### Disaster Recovery
- [ ] Automated backups (daily)
- [ ] Point-in-time recovery testing
- [ ] Failover procedures
- [ ] Data export capabilities
- [ ] Recovery time objectives (RTO)

## Environment Configuration

### Development
```bash
export PS_PG_DSN="postgres://postgres:password@localhost:5432/promptshield_dev"
export PS_CONTROL_PLANE_ADDR=":8085"
export PS_REDIS_ADDR="localhost:6379"
```

### Staging
```bash
export PS_PG_DSN="postgres://postgres:password@staging-db.supabase.co:5432/postgres"
export PS_CONTROL_PLANE_ADDR=":8085" 
export PS_REDIS_ADDR="staging-redis.company.com:6379"
```

### Production
```bash
export PS_PG_DSN="postgres://postgres:password@prod-db.supabase.co:5432/postgres"
export PS_CONTROL_PLANE_ADDR=":8085"
export PS_REDIS_ADDR="prod-redis.company.com:6379"
```

## Kubernetes Deployment (Future)

```yaml
# k8s/control-plane-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ps-control-plane
spec:
  replicas: 3
  selector:
    matchLabels:
      app: ps-control-plane
  template:
    metadata:
      labels:
        app: ps-control-plane
    spec:
      containers:
      - name: control-plane
        image: promptshield/control-plane:latest
        env:
        - name: PS_PG_DSN
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: connection-string
        ports:
        - containerPort: 8085
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8085
          initialDelaySeconds: 30
          periodSeconds: 10
```

## Migration Strategy

### Phase 1: Start with Supabase Free
1. Deploy control plane with Supabase
2. Monitor usage and performance
3. Set up basic monitoring

### Phase 2: Scale Database
1. Upgrade to Supabase Pro ($25/month)
2. Add read replicas in key regions
3. Implement caching layer

### Phase 3: Global Infrastructure
1. Deploy control plane in multiple regions
2. Set up global load balancing
3. Implement data residency compliance

## Support & Monitoring

### Key Metrics to Track
- Database connection pool utilization
- Query performance (P95 latency)
- Storage usage growth
- API request volume
- Error rates and patterns

### Alerting Thresholds
- Database CPU > 80%
- Connection pool > 90% full
- Query latency > 500ms P95
- Error rate > 1%
- Storage > 80% full

**Next Step**: Run `./scripts/setup-supabase.sh` with your Supabase connection string!