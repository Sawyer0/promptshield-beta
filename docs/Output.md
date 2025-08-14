### Output formats

Supported formats:

1) `stylish` (default)

Example:

```
Input: /abs/path/input.txt
[WARNING] /abs/path/input.txt:1:1 Detect 'ignore previous instructions' (demo-ignore-previous)
```

2) `json`

The JSON schema corresponds to `pkg/types.ScanResult`:

```json
{
  "input": "/abs/path/input.txt",
  "violations": [
    {
      "rule_id": "hello",
      "message": "rule violation",
      "severity": "WARNING",
      "line": 1,
      "column": 1,
      "category": "Data Leakage",
      "rule_timeout_ms": 50,
      "response_action": "redact",
      "response_message": "PII detected",
      "response_replacement": "***-**-****"
    }
  ],
  "metrics": {
    "bytes_read": 12,
    "lines_read": 1
  }
}
```

3) `github`

GitHub Actions annotations. Each violation is emitted as a workflow command, for example:

```
::error file=path/to/file.txt,line=1,col=1::Possible injection (rule-id)
```

Severity mapping: INFO→notice, WARNING→warning, ERROR/CRITICAL→error.

4) `ndjson`

- Event-per-violation streaming: one compact JSON object per violation, followed by a final summary line
- Emissions occur as results are produced in deterministic path order
- Suitable for piping to tools like `jq` or streaming collectors

Schema (events):

```
{ "type": "violation", "file": "/abs/path/input.txt", "rule_id": "hello", "message": "...", "severity": "WARNING", "line": 1, "column": 1 }
{ "type": "summary", "files_scanned": 2, "violation_count": 3 }
```

Notes:

- Deprecated formats (markdown, csv, html, table) have been removed for simplicity and maintenance. Use JSON for machine output and NDJSON for streaming pipelines.

Notes:

- For all non-streaming formats, one output block is emitted per input file
- Ordering is deterministic: inputs are sorted by absolute path and results are merged in that order, regardless of parallelism
- Progress is printed to stderr by default for human-readable formats (not for JSON/quiet)
- `trace_file` is deprecated; prefer OpenTelemetry via `telemetry.endpoint`.

### Audit events

When `audit_file` (or `PS_AUDIT_FILE`) is set, the CLI writes NDJSON audit events with hash chaining:

- Files rotate daily: `<base>.YYYY-MM-DD.ndjson`
- Sensitive values are redacted (API keys, tokens) before writing. You can disable this with `redaction.enabled: false` (or `PS_REDACTION_ENABLED=false`), but a warning may be logged.

Event shapes:

```
{ "timestamp": "2025-01-01T00:00:00Z", "type": "config_effective", "data": { "config_file": "./promptshield.yaml", "output": "stylish", "workers": 4, "debug": false, "quiet": false, "overrides": { "output_format": "flag" } }, "hash": "...", "prev_hash": "..." }

{ "timestamp": "2025-01-01T00:00:01Z", "type": "scan_start", "data": { "args": ["scan", "..."], "config_file": "./promptshield.yaml", "output": "ndjson", "workers": 4, "fail_on": "ERROR", "rulepack": "rules/basic-security.yaml", "overrides": { "workers": "env" } }, "hash": "...", "prev_hash": "..." }

{ "timestamp": "2025-01-01T00:00:02Z", "type": "scan_file", "data": { "input": "/abs/path/input.txt", "violations": 2 }, "hash": "...", "prev_hash": "..." }

{ "timestamp": "2025-01-01T00:00:03Z", "type": "scan_end", "data": {}, "hash": "...", "prev_hash": "..." }
```

Notes:

- Hash chaining uses SHA-256 over a canonical JSON payload
- Audit event payloads never include raw secrets; redaction is applied recursively


