variable "namespace" { type = string default = "promptshield" }
variable "chart_version" { type = string default = "0.2.0" }
variable "mode" { type = string default = "observe" }
variable "egress_policy" { type = string default = "allow" }
variable "rbac_type" { type = string default = "ClusterRole" }
