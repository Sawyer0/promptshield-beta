#!/bin/sh
set -euo pipefail

# Render Envoy config from template by simple env substitution
# Required envs: TENANT_ID, FRONTEND_TOKEN, GATEWAY_HTTP_HOST, GATEWAY_HTTP_PORT, GATEWAY_GRPC_HOST, GATEWAY_GRPC_PORT

: "${TENANT_ID:?TENANT_ID required}"
: "${FRONTEND_TOKEN:?FRONTEND_TOKEN required}"
: "${GATEWAY_HTTP_HOST:?GATEWAY_HTTP_HOST required}"
: "${GATEWAY_HTTP_PORT:?GATEWAY_HTTP_PORT required}"
: "${GATEWAY_GRPC_HOST:?GATEWAY_GRPC_HOST required}"
: "${GATEWAY_GRPC_PORT:?GATEWAY_GRPC_PORT required}"

# Substitute placeholders
cat /config/egress-dfp.tmpl.yaml \
  | sed "s/{{TENANT_ID}}/${TENANT_ID}/g" \
  | sed "s/{{FRONTEND_TOKEN}}/${FRONTEND_TOKEN}/g" \
  | sed "s/{{GATEWAY_HTTP_HOST}}/${GATEWAY_HTTP_HOST}/g" \
  | sed "s/{{GATEWAY_HTTP_PORT}}/${GATEWAY_HTTP_PORT}/g" \
  | sed "s/{{GATEWAY_GRPC_HOST}}/${GATEWAY_GRPC_HOST}/g" \
  | sed "s/{{GATEWAY_GRPC_PORT}}/${GATEWAY_GRPC_PORT}/g" \
  > /etc/envoy/envoy.yaml

# Start Envoy
exec /usr/local/bin/envoy -c /etc/envoy/envoy.yaml --service-cluster egress --log-level info

