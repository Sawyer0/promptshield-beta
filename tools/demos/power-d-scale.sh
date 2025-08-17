#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
source "$SCRIPT_DIR/common.sh"

GATEWAY_URL_DEFAULT="http://promptshield.local:8080"
GATEWAY_URL="${GATEWAY_URL:-$GATEWAY_URL_DEFAULT}"
OPENAI_API_KEY="${OPENAI_API_KEY:-}"
CONCURRENCY="${CONCURRENCY:-50}"
DURATION="${DURATION:-10}"

info "Power Demo D — Scale Showcase"
GATEWAY_URL=$(choose_gateway_url "$GATEWAY_URL")
ensure_gateway_up "$GATEWAY_URL"

if command -v hey >/dev/null 2>&1; then
  info "Running short load test via gateway (POST /v1/chat/completions)"
  BODY='{"model":"gpt-4o-mini","messages":[{"role":"user","content":"ping"}]}'
  hey -z "${DURATION}s" -c "$CONCURRENCY" -H "Authorization: Bearer $OPENAI_API_KEY" -H "Content-Type: application/json" -m POST -d "$BODY" "$GATEWAY_URL/v1/chat/completions" | cat || true
else
  warn "Install 'hey' to see live RPS numbers (https://github.com/rakyll/hey)."
fi

success "Scale demo complete."

