# Egress Proxy for Zero‑Code Enforcement

This optional overlay lets you protect other applications without SDKs or code changes.

How it works
- Run an Envoy egress proxy next to the app.
- The proxy intercepts all outbound HTTP(S): either via HTTP(S)_PROXY env vars or transparent redirect.
- It injects tenant identity headers and forwards the requests through PromptShield’s Gateway via:
  - ext_authz (fast decision on headers/body)
  - ext_proc (streaming inspection/mutation)

Files
- docker-compose.egress.yml — starts envoy-egress
- deploy/envoy/egress-dfp.tmpl.yaml — Envoy dynamic forward proxy config template
- deploy/envoy/egress-entrypoint.sh — renders template and starts Envoy

Quick start (local)
1) Start the base stack:
```bash
docker compose up -d --build
```

2) Start egress overlay:
```bash
docker compose -f docker-compose.yml -f docker-compose.egress.yml \
  -e PS_TENANT_ID=<tenant-uuid> \
  -e PS_FRONTEND_TOKEN=<frontend-token> \
  up -d
```

3) Point an app at the proxy (same Docker network):
- Set in app container env:
  - HTTP_PROXY=http://envoy-egress:8080
  - HTTPS_PROXY=http://envoy-egress:8080

Or from host for a quick test:
```bash
# Proxies through envoy-egress on localhost:8080
curl -x http://localhost:8080 https://example.com/
```

Kubernetes (transparent mode)
- Deploy envoy-egress as a sidecar and route outbound traffic via iptables REDIRECT to 8080
- See docs/K8S_SIDE_CAR_EGRESS.md for an example manifest

Security & Identity
- The egress proxy injects:
  - X-PS-Tenant-Id: the tenant UUID
  - X-PS-Token: a frontend token (scoped to the tenant)
- Your Gateway uses these to enforce tenant policies and authenticate the egress proxy’s calls.

Limits
- Without Envoy, enforcement is low‑code (apps call /v1/check explicitly).
- With Envoy egress, enforcement is zero‑code for outbound HTTP.

