#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)"
source "$SCRIPT_DIR/common.sh"

GATEWAY_URL_DEFAULT="http://127.0.0.1:18081"
GATEWAY_URL="${GATEWAY_URL:-$GATEWAY_URL_DEFAULT}"

FILE_PATH="${1:-}"
if [[ -z "$FILE_PATH" || ! -f "$FILE_PATH" ]]; then
  error "Usage: $0 <rules.yaml|rules.json>"
  exit 2
fi

ensure_gateway_up "$GATEWAY_URL"

mime="application/json"
case "$FILE_PATH" in
  *.yaml|*.yml) mime="application/x-yaml" ;;
  *.json) mime="application/json" ;;
esac

info "Uploading rules from $FILE_PATH to $GATEWAY_URL/v1/rules (Content-Type: $mime)"
dir=$(mktemp -d)
body_file="$dir/body.bin"
cp "$FILE_PATH" "$body_file"

outdir=$(mktemp -d)
meta="$outdir/meta.txt"; body="$outdir/body.json"; errf="$outdir/stderr.txt"
curl -sS -o "$body" -w "%{http_code} %{time_total}\n" -H "Content-Type: $mime" -X POST "$GATEWAY_URL/v1/rules" --data-binary @"$body_file" > "$meta" 2>"$errf" || true
printf "HTTP %s\n" "$(awk '{print $1}' "$meta")"
cat "$body" | json_pretty || cat "$body" || true

success "Rules uploaded."

