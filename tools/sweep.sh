#!/usr/bin/env bash
set -euo pipefail

URL=${1:-http://127.0.0.1:8080/post}
BODY=${2:-body64k.txt}
RATES=(6000 8000 10000 12000)
CONNS=${CONNS:-1000}
THREADS=${THREADS:-$(nproc)}
DUR=${DUR:-30s}
TIMEOUT=${TIMEOUT:-10s}

# Ensure body file exists; generate a 64KB ASCII payload if missing
if [ ! -f "$BODY" ]; then
  echo "Body file not found: $BODY; generating a 64KB payload..."
  if command -v dd >/dev/null 2>&1; then
    dd if=/dev/zero bs=1 count=65536 2>/dev/null | tr '\0' 'A' > "$BODY"
  else
    python - <<'PY' > "$BODY"
import sys
sys.stdout.write('A'*65536)
PY
  fi
fi

echo "Target: $URL"
echo "Body:   $BODY"
echo "Rates:  ${RATES[*]} req/s  (connections=$CONNS threads=$THREADS duration=$DUR timeout=$TIMEOUT)"

for r in "${RATES[@]}"; do
  echo
  echo "=== Rate ${r}/s ==="
  echo "POST $URL" | vegeta attack \
    -rate=$r -duration=$DUR -connections=$CONNS -max-workers=$((CONNS*4)) -timeout=$TIMEOUT \
    -body="$BODY" -header="Content-Type: text/plain" \
    | tee "/tmp/vegeta_${r}.bin" \
    | vegeta report
done


