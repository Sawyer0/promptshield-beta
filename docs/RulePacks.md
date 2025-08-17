# RulePacks – composition, imports, options, and performance

This document describes the RulePack schema used by PromptShield and how rules are evaluated.

## Composition

- `composition.strategy`: `all_matches` (default), `first_match`, or `priority_order`
  - `all_matches`: evaluate all rules; multiple violations per line may be reported
  - `first_match`: stop evaluating additional rules on a line after the first rule matches
  - `priority_order`: merge packs in deterministic order; earlier rules take precedence (first-wins for duplicate rule IDs)

Example (first_match):

```yaml
apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: my-pack
composition:
  strategy: first_match
rules:
  - id: fast-keyword
    level: 1
    severity: WARNING
    keywords: ["ignore previous instructions"]
  - id: regex-followup
    level: 2
    severity: WARNING
    patterns:
      - regex: "(?i)ignore\\s+previous"
```

Example (priority_order with extends):

```yaml
apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: app-rules
extends: [base-rules]
composition:
  strategy: priority_order
rules:
  - id: injection-001
    level: 2
    severity: ERROR
    patterns:
      - regex: "(?i)drop\\s+table"
```

## Rule options

Keyword rules (level 1) support fine‑grained options:

```yaml
rules:
  - id: kw-example
    level: 1
    severity: WARNING
    keywords: ["password", "secret"]
    options:
      case_sensitive: false   # default derived from performance.case_sensitive
      whole_word: false       # default derived from performance.whole_word
```

Behavior:
- `case_sensitive`: when false, matching is case‑insensitive
- `whole_word`: when true, matches only when surrounded by non‑word characters

Global defaults can be set under `performance`:

```yaml
performance:
  case_sensitive: false
  whole_word: true
```

Rule‑level options override these defaults.

## Regex rules (level 2)

Regex patterns (level 2) support flags:

```yaml
patterns:
  - regex: "\\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\\.[A-Za-z]{2,}\\b"
    flags: [ignorecase, multiline]
```

Allowed flags: `ignorecase` (`i`), `multiline` (`m`). Invalid flags are rejected by the validator.

The engine reports multiple matches per line when present.

### Pattern complexity guards

PromptShield validates regex complexity to avoid catastrophic backtracking. Packs are rejected if patterns exceed limits on nodes/depth/alternations/repeats or length (default 1000 chars). Limits can be tuned via env:

```bash
PS_MAX_REGEX_NODES=750
PS_MAX_REGEX_DEPTH=30
PS_MAX_REGEX_ALTERNATIONS=300
PS_MAX_REGEX_REPEATS=1000
```

These checks also apply during scanner rule compilation; overly complex patterns are skipped defensively.

## Semantic rules (level 3)

Level‑3 evaluation is a last‑resort semantic classifier used only when L1/L2 do not match the line. It is opt‑in and requires explicit configuration:

```yaml
rules:
  - id: sem-manipulation
    level: 3
    severity: ERROR
    semantic:
      model: gpt-4o-mini           # required
      analysis_prompt: |
        Respond VIOLATION or SAFE for: {input}   # required; {input} is replaced with line text
### Response Actions

Rules may specify response mapping to control runtime behavior when a violation is detected on responses:

```yaml
response:
  action: replace            # one of: redact|mask|replace|deny|block|quarantine
  message: "Sensitive content removed"
  replacement: "[REDACTED_RESPONSE]"   # required for replace; ignored otherwise
```

- `redact|mask|quarantine`: the enforcer will mutate response body chunks to redact sensitive content (streaming-safe)
- `replace`: the enforcer will terminate the stream with a 200 and the replacement body (full replacement)
- `deny|block`: the enforcer will return 403 ImmediateResponse
      confidence_threshold: 0.85    # optional
      fallback_on_error: true       # optional
    fallback:
      patterns:
        - regex: '\\b(jailbreak|DAN|evil)\\b'
          flags: [ignorecase]
```

Enable semantic evaluation with environment variables (no defaults, explicit provider required):

```bash
export PS_SEMANTIC_ENABLED=true
export PS_SEMANTIC_PROVIDER=openai   # or 'anthropic'
export OPENAI_API_KEY=...            # or PS_OPENAI_API_KEY for openai
# for anthropic instead: ANTHROPIC_API_KEY or PS_ANTHROPIC_API_KEY
```

Provider adapter respects per‑rule timeouts, global defaults, concurrency and caching:
- Budget: `rule.timeout` or `performance.per_rule_timeout`
- Concurrency: `PS_SEMANTIC_MAX_CONCURRENCY` (default 2)
- Cache: `PS_SEMANTIC_CACHE_SIZE` (default 1000), `PS_SEMANTIC_CACHE_TTL` (default 15m)
- Privacy: adapter redacts likely tokens and truncates payloads before send
 - Tracing: if `telemetry.endpoint` is configured, provider HTTP spans are emitted via OpenTelemetry
 - Logging: when `debug` is enabled (env/config), providers emit structured request/response summaries (no payloads); cache hits are logged

## Imports and Extends

RulePacks can compose other packs for reuse and distribution:

- `imports:`: include additional packs. Supported forms:
  - Relative/absolute file paths
  - Directory paths (all `*.yml`/`*.yaml`, non-recursive)
  - Glob patterns, including recursive `**` (e.g., `packs/**/security*.yaml`)
  - HTTP/HTTPS URLs when explicitly enabled via `PS_ALLOW_NET_IMPORTS=1`
  - Marketplace slugs `namespace/name@version` resolved via `PS_MARKETPLACE_DIR` (local) or `PS_REGISTRY_URL` (remote when `PS_ALLOW_NET_IMPORTS=1`)

- `extends:`: inherit from other packs present in the load set. Extends are resolved deterministically (DFS) and merged before local rules. `overrides:` apply after merge.

Example:

```yaml
apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: app-rules
extends: [base-security]
overrides:
  - rule_id: pii-detection
    severity: CRITICAL
    # enabled: false  # disable a rule
```

Imports are resolved deterministically with cycle prevention and de‑duplication. Imported packs are merged alongside the base pack subject to the selected composition strategy.

## Performance and timeouts

```yaml
performance:
  max_length: 1048576          # optional; skip expensive regex when a line exceeds this length
  timeout: "5s"               # optional file‑level timeout
  per_rule_timeout: "50ms"    # default per‑rule time budget (rule.timeout overrides)
  total_scan_timeout: "30s"   # end‑to‑end budget
```

These values are read by the enforcer and scanner; invalid durations are rejected. Per‑rule `timeout` overrides the global.

## Example pack

```yaml
apiVersion: promptshield.io/v1
kind: RulePack
metadata:
  name: example-pack
rules:
  - id: demo-ignore-previous
    level: 1
    severity: HIGH
    keywords: ["ignore previous instructions"]
    options:
      case_sensitive: false
      whole_word: false
  - id: demo-api-key-regex
    level: 2
    severity: WARNING
    patterns:
      - regex: "(?i)sk-[a-z0-9]{10,}"
        flags: [ignorecase]
performance:
  case_sensitive: false
  whole_word: false
  per_rule_timeout: "50ms"
```

