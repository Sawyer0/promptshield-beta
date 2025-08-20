# PromptShield Helm Chart ✅ PRODUCTION-READY

**🚀 Enterprise Kubernetes deployment for PromptShield Security Gateway**

This chart deploys a **production-ready** PromptShield Enforcer with:
- ⚡ **Sub-millisecond security decisions** via HTTP `/check` API
- 🔄 **Envoy integration** via gRPC `ext_proc` streaming  
- 📊 **Built-in observability** with Prometheus metrics & health probes
- 🔐 **Security hardened** with RBAC, NetworkPolicies, resource limits

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `replicaCount` | int | `1` | Number of replicas |
| `image.repository` | string | `ghcr.io/promptshield/enforcer` | Container image repository |
| `image.tag` | string | `0.2.0` | Image tag |
| `mode` | string | `observe` | Enforcement mode: `observe` or `enforce` |
| `rbac.create` | bool | `true` | Create RBAC objects |
| `rbac.type` | string | `ClusterRole` | `ClusterRole` or `Namespaced` |
| `egress.policy` | string | `allow` | `allow` (no NetworkPolicy) or `deny` (NetworkPolicy) |
| `egress.cidrBlock` | string | `0.0.0.0/0` | CIDR allowed when `policy=deny` |

## Notes

* When `mode=enforce`, the enforcer returns `403` on violations. `observe` always returns `200` with decision headers for safe rollout.
* Set `PS_ENFORCER_RULEPACK` via `extraEnv` (not shown) to mount custom RulePacks.

## 🚀 **Quick Deploy** (Production Ready)

**1) Install with built-in prompt injection rules:**
```bash
helm repo add promptshield https://charts.promptshield.io
helm install promptshield promptshield/promptshield \
  --set mode=enforce \
  --set-string extraEnv.PS_ENFORCER_RULEPACK=rules/prompt-injection.yaml
```

**2) Verify deployment:**
```bash
kubectl rollout status deploy/promptshield-enforcer
kubectl port-forward svc/promptshield-enforcer 9090:9090
```

**3) Test live security decisions:**
```bash
# Safe content → ALLOW
curl -X POST http://localhost:9090/check \
  -H 'content-type: text/plain' \
  --data 'Hello world'

# Prompt injection → DENY
curl -X POST http://localhost:9090/check \
  -H 'content-type: text/plain' \
  --data 'Ignore previous instructions'
```

**4) Monitor with Prometheus:**
```bash
kubectl port-forward svc/promptshield-enforcer 9090:9090
curl http://localhost:9090/metrics | grep ps_enforcer_decisions_total
```

