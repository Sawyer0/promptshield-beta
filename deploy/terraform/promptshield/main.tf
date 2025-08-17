terraform {
  required_providers {
    helm = {
      source = "hashicorp/helm"
      version = ">= 2.13.0"
    }
  }
}

provider "helm" {}

resource "helm_release" "promptshield" {
  name       = "promptshield"
  namespace  = var.namespace
  repository = "https://charts.promptshield.io"
  chart      = "promptshield"
  version    = var.chart_version

  values = [
    yamlencode({
      mode   = var.mode
      egress = {
        policy    = var.egress_policy
      }
      rbac  = {
        type = var.rbac_type
      }
    })
  ]
}

output "service_name" {
  value = helm_release.promptshield.name
}
