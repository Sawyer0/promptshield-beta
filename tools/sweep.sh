#!/usr/bin/env bash
set -euo pipefail

URL=${1:-http://127.0.0.1:9090/check}
BODY=${2:-body64k.txt}
RATES=(6000 8000 10000 12000)
CONNS=${CONNS:-1000}
THREADS=${THREADS:-$(nproc)}
DUR=${DUR:-30s}
TIMEOUT=${TIMEOUT:-10s}

echo "Target: $URL"
echo "Body:   $BODY"
echo "Rates:  ${RATES[*]} req/s  (connections=$CONNS threads=$THREADS duration=$DUR timeout=$TIMEOUT)"

for r in "${RATES[@]}"; do
  echo
  echo "=== Rate ${r}/s ==="
  echo "POST $URL" | vegeta attack \
    -rate=$r -duration=$DUR -connections=$CONNS -max-workers=$((CONNS*4)) -timeout=$TIMEOUT \
    -body="$BODY" -header="Content-Type: text/plain" \
    | tee "/tmp/vegeta_${r}.bin" >/dev/null \
    | vegeta report
done


