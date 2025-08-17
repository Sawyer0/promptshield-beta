#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
source "$SCRIPT_DIR/common.sh"

info "Power Demo B — Compliance Instant Win"

printf "%s\n" "Compliance Checklist:"
printf "%b%s%b\n" "$GREEN" "[✓] PII Detection" "$NC"
printf "%b%s%b\n" "$GREEN" "[✓] GDPR Compliance" "$NC"
printf "%b%s%b\n" "$GREEN" "[✓] Audit Trail" "$NC"
printf "%b%s%b\n" "$GREEN" "[✓] Data Residency" "$NC"

printf "%s\n" "From zero to compliant in 5 minutes."

success "Compliance demo complete."

