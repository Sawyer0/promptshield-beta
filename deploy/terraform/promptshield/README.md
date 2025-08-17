# PromptShield Terraform Module

This module deploys the PromptShield Helm chart with a minimal set of variables.

```hcl
module "promptshield" {
  source = "github.com/promptshield/promptshield//deploy/terraform/promptshield"

  namespace     = "security"
  chart_version = "0.2.0"
  mode          = "enforce"    # observe | enforce
  egress_policy = "deny"       # allow | deny
  rbac_type     = "Namespaced" # ClusterRole | Namespaced
}
```
