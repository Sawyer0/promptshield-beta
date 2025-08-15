## Performance, Benchmarks, and SLAs

This document captures PromptShield's performance goals, how to run the benchmark suite, and how to enforce SLAs.

### Benchmarks

- internal/scanner:
  - `BenchmarkScanLargeFile`: large string content, keyword rules
  - `BenchmarkScanOneGiB`: 1 GiB streaming reader, bounded memory
  - `BenchmarkParallelScan`, `BenchmarkRuleMatching`, Aho/regex gates, P95
- gateway:
  - `BenchmarkGatewayHTTPCheck64KB`: HTTP `/check` end‑to‑end with a 64KB body
  - `BenchmarkGatewayGRPCExtProc_SetupOnly`: gRPC ext_proc minimal stream setup/overhead

Run all benchmarks:
```bash
go test -run=^$ -bench . -benchmem -count=1 ./...
```

Useful targets:
```bash
# Quick P95 scanner bench
make bench-quick

# 1 GiB streaming scanner bench
make bench-large
```

Run specific benchmarks:
```bash
go test -run=^$ ./internal/scanner -bench '^BenchmarkScan(LargeFile|OneGiB)$' -benchmem -count=1
go test -run=^$ ./gateway -bench BenchmarkGatewayHTTPCheck64KB -benchmem -count=1
go test -run=^$ ./gateway -bench BenchmarkGatewayGRPCExtProc_SetupOnly -benchmem -count=1
```

### SLA Tests (opt‑in)

SLA tests are disabled by default and must be explicitly enabled to avoid flakiness across different developer machines and CI hardware. Set `PS_ENFORCE_SLA=1` to enable.

Defaults (override via env):
- Scanner throughput: `≥ 200 MB/s` (override `PS_SLA_SCANNER_MBPS_MIN`)
- Gateway HTTP `/check` throughput: `≥ 10 MB/s` (override `PS_SLA_HTTP_MBPS_MIN`)
- gRPC ext_proc latency: `≤ 25 ms` per small stream (override `PS_SLA_GRPC_MS_MAX`)

Commands:
```bash
# Scanner SLA
PS_ENFORCE_SLA=1 go test ./internal/scanner -run TestScannerThroughput_SLA -count=1

# Gateway HTTP SLA
PS_ENFORCE_SLA=1 go test ./gateway -run TestHTTPCheck_SLA -count=1

# Gateway gRPC SLA
PS_ENFORCE_SLA=1 go test ./gateway -run TestGRPCExtProc_SLA -count=1
```

### Latest Reference Numbers (example machine)

Environment: Windows amd64, Intel(R) Core(TM) Ultra 5 125U.

- Scanner
  - `BenchmarkScanLargeFile`: ~1.47 ms/op, ~685 MB/s, ~3.0 MB/op, 17 allocs/op
  - `BenchmarkScanOneGiB`: ~4.68 s/op, ~229 MB/s, ~2.96 GB/op
- Gateway
  - HTTP `/check` 64KB body: ~0.39 ms/op, ~169 MB/s (after streaming change)
  - gRPC ext_proc setup path: ~17–19 µs/op

Notes:
- Some "heavy" regex benches intentionally stress worst‑case behavior and are not representative of normal workloads.
- Throughput numbers vary by CPU, OS, Go version, and power profiles. Treat these as baselines, not hard guarantees across all hardware.

### Implementation Notes

- Gateway HTTP `/check` now streams directly from the request body into the scanner (no temp files), improving throughput by ~50x on small bodies.
- `internal/scanner` maintains streaming‑first scanning with bounded memory and supports very large inputs (1 GiB benchmark included).
- Chaos testing can be enabled via:
  - `PS_CHAOS=1` (enable)
  - `PS_CHAOS_FAIL_PCT=5` (percent operations fail)
  - `PS_CHAOS_DELAY_MS_AVG=10` (avg delay)



### Realistic Payload Profiles

Most live traffic should model these sizes and mix:

- Small (≈ 70% of requests)
  - Prompts: 0.25–2 KB (typical single‑turn user input)
  - Responses: 1–5 KB
- Medium (≈ 25%)
  - Prompts: 2–10 KB (multi‑turn + instructions)
  - Responses: 5–20 KB
- Large (≈ 5%)
  - Prompts: 10–50 KB (system prompts, long context)
  - Responses: 20–100 KB
- Stress tier (keep separate): 50–100+ KB prompts, 100–500+ KB responses

This profile is a better default than 64KB‑only stress testing.


### Mixed‑Payload Load Test (70/25/5)

Use `tools/mixed.sh` to generate small/medium/large bodies and run three concurrent fixed‑rate tests matching a 70/25/5 mix.

Defaults:
- Target: Envoy listener `http://127.0.0.1:8080/post`
- Sizes: ~1 KB, ~5 KB, ~32 KB
- Mix: 70% / 25% / 5%

Run against Envoy (end‑to‑end path):
```bash
bash tools/mixed.sh
```

Run against the enforcer directly (scanner‑only capacity):
```bash
URL=http://127.0.0.1:9090/check R=800 CONNS=100 TIMEOUT=5s DUR=20s bash tools/mixed.sh
```

Notes:
- Ensure license is set for unrestricted testing (evaluation mode is heavily rate‑limited):
  - `PROMPTSHIELD_LICENSE_KEY=...` (env‑based)
- For end‑to‑end runs, keep reasonable edge timeouts and limits:
  - In `envoy-config.yaml`, set `ext_proc.message_timeout: 5.000s`
  - Disable `ext_authz` during load tests to avoid unrelated gating
- On Windows, reduce concurrency (e.g., `CONNS=50–200`) to avoid ephemeral port exhaustion.


### Reference Results (Realistic sizes)

Environment: WSL2 Ubuntu on Intel(R) Core(TM) Ultra 5 125U, Go 1.25.0.

- Scanner‑only (direct `/check`, mixed 70/25/5; R=800, CONNS=100, 20s):
  - Small (~1 KB): 100% success; mean ~1.10 ms
  - Medium (~5 KB): 100% success; mean ~1.07 ms
  - Large (~32 KB): 100% success; mean ~1.26 ms
- Scanner‑only (100 KB body; R=400, CONNS=100, 20s):
  - 100% success; mean ~2.16 ms; P95 ~4.9 ms; P99 ~13 ms

Interpretation: The engine’s compute path is low single‑digit milliseconds even for large prompts, supporting inline enforcement with negligible latency overhead. End‑to‑end performance then depends on Envoy/backend sizing and OS/network limits.


### WSL2 High-Load Results (Fast mode)

Environment: WSL2 Ubuntu, Intel(R) Core(TM) Ultra 5 125U, Go 1.25.0.

Enforcer flags/env during tests:
- `PS_ENFORCER_FAST=1` `PS_ENFORCER_DISABLE_METRICS=1` `PS_ENFORCER_DISABLE_TRACING=1`
- `GOMAXPROCS=$(nproc)` and `ulimit -n 65535`
- License set to unlimited RPS (base64url token with `entitlements.max_rps=0`)

Measured with `wrk` (each 30s):

| Scenario | Tool | Threads | Concurrency/Rate | Body size | RPS | p50 | p75 | p90 | p99 | Socket timeouts |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `/check` small body | wrk | 14 | c=1000 | 2 bytes | 42,651 | 17.32ms | 68.50ms | 117.10ms | 238.13ms | 771 |
| `/check` small body (timeout=10s) | wrk | 14 | c=1000 | 2 bytes | 41,177 | 18.62ms | 69.28ms | 118.85ms | 241.52ms | 330 |
| `/check` small body | wrk | 14 | c=1500 | 2 bytes | 42,909 | 24.51ms | 103.75ms | 174.49ms | 356.18ms | 341 |
| `/check` small body (2 procs, GOMAXPROCS=1 each) | wrk ×2 | 7+7 | c=750+750 | 2 bytes | ~10,805 total | 195.95ms | 229.29ms | 251.14ms | 298.77ms | 31+21 |
| `/check` 64KB body (streaming) | wrk | 16 | c=1000 | 65,536 bytes | 15,782 | 44.58ms | 201.59ms | 321.23ms | 657.06ms | 87 |

Notes:
- Small-body runs sustain ~41–43k RPS with P95 < 300ms (inferred from p90/p99).
- For 64KB bodies at c=1000, P95 exceeds 300ms; reducing concurrency or rate keeps P95 under 300ms while sustaining multi‑k RPS.
- Fixed‑rate `vegeta` at 15k RPS with 64KB bodies overloaded this host (≥ 80% > 1s). Use rate sweeps (8–12k/s) to maintain P95 < 300ms.

Reproduction (64KB streaming):
```bash
python3 - <<'PY' > body64k.txt
import sys; sys.stdout.write('x'*65536)
PY
cat > post64k.lua <<'EOF'
wrk.method = "POST"
wrk.body   = io.open('body64k.txt'):read('*a')
wrk.headers["Content-Type"] = "text/plain"
EOF
wrk -t16 -c1000 -d30s --timeout 10s --latency -s ./post64k.lua http://127.0.0.1:9090/check
```
