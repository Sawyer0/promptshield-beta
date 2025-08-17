# PromptShield Helm Chart

This chart deploys the PromptShield Enforcer (HTTP + gRPC) into a Kubernetes cluster.

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

```bash
helm repo add promptshield https://charts.promptshield.io
helm install ps promptshield/promptshield \
  --set mode=enforce \
  --set rbac.type=Namespaced
```

