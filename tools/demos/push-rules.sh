#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
source "$SCRIPT_DIR/common.sh"

GATEWAY_URL_DEFAULT="http://127.0.0.1:18081"
GATEWAY_URL="${GATEWAY_URL:-$GATEWAY_URL_DEFAULT}"

ensure_gateway_up "$GATEWAY_URL"

case "${1:-}" in
  healthcare)
    BODY='{"rules":[
      {"id":"HIPAA-PII","category":"PII","severity":"high","message":"Detected PHI/PII pattern","patterns":["(?i)(ssn|social security|mrn|patient id)","(?i)(dob|date of birth)"]},
      {"id":"HIPAA-EXPORT","category":"DataExfil","severity":"high","message":"Disallowed export of medical data","patterns":["(?i)(export|leak).*(patient|record|medical)"]}
    ]}'
    ;;
  finance)
    BODY='{"rules":[
      {"id":"PCI-PAN","category":"Payment","severity":"high","message":"Credit card PAN sequence",
       "patterns":["(?i)\b(?:\d[ -]*?){13,16}\b"]},
      {"id":"FIN-TRANSFER","category":"Fraud","severity":"medium","message":"Suspicious transfer instruction","patterns":["(?i)(wire|transfer).*(all|entire|full).*(funds|balance)"]}
    ]}'
    ;;
  retail)
    BODY='{"rules":[
      {"id":"GDPR-DATA","category":"PII","severity":"high","message":"EU personal data request","patterns":["(?i)(download|export|share).*(customer|user).*(data)"]}
    ]}'
    ;;
  education)
    BODY='{"rules":[
      {"id":"FERPA","category":"StudentData","severity":"high","message":"Student record disclosure","patterns":["(?i)(student|record).*(share|export|email)"]}
    ]}'
    ;;
  government)
    BODY='{"rules":[
      {"id":"ITAR","category":"Classified","severity":"high","message":"Controlled info handling","patterns":["(?i)(itar|controlled|classified).*(share|export)"]}
    ]}'
    ;;
  *)
    error "Usage: $0 {healthcare|finance|retail|education|government}"
    exit 2
    ;;
esac

info "Pushing rules to $GATEWAY_URL/v1/rules for profile: ${1}"
dir=$(curl_post_json_capture "$GATEWAY_URL/v1/rules" ":" "$BODY")
code=$(resp_http_code "$dir")
printf "HTTP %s\n" "$code"
resp_body_cat_redacted "$dir" | json_pretty || true
success "Rules updated."

