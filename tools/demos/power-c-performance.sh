#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
source "$SCRIPT_DIR/common.sh"

GATEWAY_URL_DEFAULT="http://promptshield.local:8080"
MODEL_URL_DEFAULT="http://127.0.0.1:18080"
GATEWAY_URL="${GATEWAY_URL:-$GATEWAY_URL_DEFAULT}"
MODEL_URL="${MODEL_URL:-$MODEL_URL_DEFAULT}"
OPENAI_API_KEY="${OPENAI_API_KEY:-}"
TRIALS="${TRIALS:-5}"

info "Power Demo C — Performance Proof"
ensure_provider_up "$MODEL_URL"
GATEWAY_URL=$(choose_gateway_url "$GATEWAY_URL")
ensure_gateway_up "$GATEWAY_URL" "$MODEL_URL"

info "Measuring $TRIALS trials: direct vs via gateway (POST /v1/chat/completions)"

direct_times=()
via_times=()
SYS="You are a helpful assistant."
PROMPT="Say 'hello'."

for i in $(seq 1 "$TRIALS"); do
  dir=$(run_chat "$MODEL_URL" "$OPENAI_API_KEY" "gpt-4o-mini" "$SYS" "$PROMPT")
  via=$(run_chat "$GATEWAY_URL" "$OPENAI_API_KEY" "gpt-4o-mini" "$SYS" "$PROMPT")
  dt=$(resp_time_total "$dir"); vt=$(resp_time_total "$via")
  printf "Trial %d: direct=%s via=%s\n" "$i" "$dt" "$vt"
  direct_times+=("${dt:-0}")
  via_times+=("${vt:-0}")
done

printf "P50 direct: %s\n" "$(percentile 50 "${direct_times[@]}")"
printf "P50 via:    %s\n" "$(percentile 50 "${via_times[@]}")"

success "Performance demo complete."

