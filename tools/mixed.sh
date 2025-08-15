#!/usr/bin/env bash
set -euo pipefail

# Mixed-payload load test via Envoy (small/medium/large prompts)
# Defaults target typical LLM traffic sizes. Adjust R, DUR, CONNS, TIMEOUT as needed.

URL=${URL:-http://127.0.0.1:8080/post}
R=${R:-3000}                # total RPS
SPLIT_SMALL=${SPLIT_SMALL:-70}
SPLIT_MED=${SPLIT_MED:-25}
SPLIT_LARGE=${SPLIT_LARGE:-5}
DUR=${DUR:-30s}
CONNS=${CONNS:-400}
TIMEOUT=${TIMEOUT:-5s}

SMALL_BODY=${SMALL_BODY:-body-1k.txt}     # ~1KB
MED_BODY=${MED_BODY:-body-5k.txt}         # ~5KB
LARGE_BODY=${LARGE_BODY:-body-32k.txt}    # ~32KB

mk_payload() {
  local path=$1; local bytes=$2
  if [ -f "$path" ]; then return 0; fi
  echo "Generating $path (${bytes}B)"
  if command -v awk >/dev/null 2>&1; then
    awk -v n="$bytes" 'BEGIN{for(i=0;i<n;i++) printf "A"}' > "$path"
  elif command -v python >/dev/null 2>&1; then
    python - "$bytes" > "$path" <<'PY'
import sys
n=int(sys.argv[1]); sys.stdout.write('A'*n)
PY
  else
    # POSIX fallback: concatenate 64B lines until size reached
    : > "$path"
    line="AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
    while [ $(wc -c < "$path" | tr -d ' ') -lt "$bytes" ]; do printf "%s" "$line" >> "$path"; done
    # trim to exact size
    sz=$(wc -c < "$path" | tr -d ' ')
    if [ "$sz" -gt "$bytes" ]; then head -c "$bytes" "$path" > "$path.tmp" && mv "$path.tmp" "$path"; fi
  fi
}

mk_payload "$SMALL_BODY" 1024
mk_payload "$MED_BODY"   5120
mk_payload "$LARGE_BODY" 32768

small_r=$(( R * SPLIT_SMALL / 100 ))
med_r=$(( R * SPLIT_MED / 100 ))
large_r=$(( R * SPLIT_LARGE / 100 ))

echo "Target: $URL"
echo "Rates:  small=$small_r, medium=$med_r, large=$large_r  (total=$R)"
echo "Bodies: $SMALL_BODY ~1KB, $MED_BODY ~5KB, $LARGE_BODY ~32KB"
echo

# Run three concurrent attacks; capture binary results; then aggregate
( echo "POST $URL" | vegeta attack -rate=$small_r -duration=$DUR -connections=$CONNS -max-workers=$((CONNS*4)) -timeout=$TIMEOUT -body="$SMALL_BODY" -header="Content-Type: text/plain" | tee /tmp/vegeta_small.bin >/dev/null ) &
( echo "POST $URL" | vegeta attack -rate=$med_r   -duration=$DUR -connections=$CONNS -max-workers=$((CONNS*4)) -timeout=$TIMEOUT -body="$MED_BODY"   -header="Content-Type: text/plain" | tee /tmp/vegeta_med.bin   >/dev/null ) &
( echo "POST $URL" | vegeta attack -rate=$large_r -duration=$DUR -connections=$CONNS -max-workers=$((CONNS*4)) -timeout=$TIMEOUT -body="$LARGE_BODY" -header="Content-Type: text/plain" | tee /tmp/vegeta_large.bin >/dev/null ) &
wait

echo
echo "=== Per-segment reports ==="
echo "-- small --"; vegeta report /tmp/vegeta_small.bin || true
echo "-- medium --"; vegeta report /tmp/vegeta_med.bin || true
echo "-- large --"; vegeta report /tmp/vegeta_large.bin || true


