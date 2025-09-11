# PromptShield Docker Setup

This document describes how to run PromptShield using Docker Compose with the new BFF (Backend for Frontend) architecture.

## Architecture Overview

The Docker setup includes:

- **frontend-bff**: Node.js Express server serving the React UI and BFF APIs
- **promptshield-1/2/3**: Go enforcer API instances
- **envoy-1/2**: Optional enforcement proxies
- **nginx**: Load balancer and reverse proxy
- **redis**: Session and cache storage
- **prometheus/grafana**: Monitoring stack

## Quick Start

### Prerequisites

- Docker Desktop installed and running
- Ports 3001, 8080, 9090 available

### Setup

1. **Build and start the stack:**
   ```bash
   ./scripts/docker-setup.sh
   ```

   Or manually:
   ```bash
   docker compose up -d --build
   ```

2. **Test the setup:**
   ```bash
   ./scripts/test-setup.sh
   ```

3. **Access the application:**
   - Frontend UI: http://localhost:3001
   - Enforcer API: http://localhost:9090
   - Nginx Proxy: http://localhost:80
   - Enforcement Proxy: http://localhost:8080
   - Grafana: http://localhost:3000 (admin/admin)
   - Prometheus: http://localhost:9092

## Configuration

### Environment Variables

The setup uses environment variables from the docker-compose.yaml file. Key variables:

- `PS_PG_DSN`: PostgreSQL connection string (Supabase)
- `SESSION_SECRET`: Secret for session encryption
- `PS_ENFORCER_MODE`: Enforcement mode (observe/enforce)
- `PS_ENFORCER_ADMIN_TOKEN`: Admin authentication token

### Database Setup

The application expects a PostgreSQL database with the schema defined in `frontend/RulepackManager/shared/schema.ts`. The setup uses Supabase by default.

## Service Details

### Frontend BFF

- **Port**: 3001 (external), 8096 (internal)
- **Purpose**: Serves React UI and Express APIs
- **Database**: Uses same PostgreSQL as enforcer
- **Authentication**: Session-based with Replit OIDC (placeholder)

### Enforcer API

- **Port**: 9090
- **Purpose**: Security enforcement and rulepack management
- **Database**: PostgreSQL for enterprise features
- **Authentication**: JWT tokens (optional)

### Nginx Proxy

- **Ports**: 80 (API), 8080 (enforcement), 8098 (frontend BFF)
- **Purpose**: Load balancing and reverse proxy
- **Features**: Rate limiting, CORS, SSL termination

## Development

### Viewing Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f frontend-bff
docker compose logs -f promptshield-1
```

### Restarting Services

```bash
# Restart all
docker compose restart

# Restart specific service
docker compose restart frontend-bff
```

### Rebuilding

```bash
# Rebuild and restart
docker compose up -d --build

# Rebuild specific service
docker compose up -d --build frontend-bff
```

## Troubleshooting

### Common Issues

1. **Port conflicts**: Ensure ports 3001, 8080, 9090 are available
2. **Database connection**: Check `PS_PG_DSN` environment variable
3. **Build failures**: Ensure Docker has enough memory (4GB+ recommended)

### Health Checks

```bash
# Enforcer health
curl http://localhost:9090/healthz

# Frontend BFF health
curl http://localhost:3001/api/healthz

# Nginx proxy health
curl http://localhost:80/healthz
```

### Debugging

```bash
# Enter container shell
docker compose exec frontend-bff sh
docker compose exec promptshield-1 sh

# View container resources
docker stats
```

## Production Considerations

### Security

- Change default passwords and tokens
- Enable TLS/SSL
- Configure proper firewall rules
- Use secrets management for sensitive data

### Performance

- Adjust resource limits in docker-compose.yaml
- Configure proper database connection pooling
- Enable monitoring and alerting
- Consider horizontal scaling

### Monitoring

- Grafana dashboards available at http://localhost:3000
- Prometheus metrics at http://localhost:9092
- Application logs via Docker Compose

## Next Steps

1. **Authentication**: Configure proper JWT keys for BFF-Enforcer communication
2. **TLS**: Add SSL certificates and enable HTTPS
3. **Scaling**: Add more enforcer instances as needed
4. **Backup**: Configure database backups
5. **CI/CD**: Set up automated deployment pipeline
