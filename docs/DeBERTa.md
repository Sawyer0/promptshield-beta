# DeBERTa v3 Prompt Injection (ProtectAI) Integration

This deployment uses a DeBERTa v3-based classifier for Level-3 (semantic) analysis instead of external provider models. The analyzer is called via a configurable HTTP inference endpoint and returns only a boolean (flagged) and a numeric confidence (risk score). No chain-of-thought is ever returned to users; a short, internal-only rationale is logged for operators.

## Environment variables

- PS_SEMANTIC_ENABLED=true
- PS_DEBERTA_ENDPOINT=http://localhost:8089/infer
  - Or a HuggingFace Inference endpoint:
    https://api-inference.huggingface.co/models/protectai/deberta-v3-base-prompt-injection-v2
- HF_TOKEN=... (optional if required by your inference endpoint)

Policy bridge (weighted scoring):
- PS_ALPHA=0.7               # weight for L3 risk (confidence)
- PS_BETA=0.3                # weight for pattern score (L1/L2)
- PS_BLOCK_THRESHOLD=0.75    # final score threshold to quarantine when no rule action already blocks

Conversation signals:
- X-PS-Conversation-ID: <stable id>
  - The enforcer computes simple drift/privileged jump signals and adds them to scanner runtime context for gated rules.

Testing aids:
- PS_FAKE_L3=true            # deterministic fake L3 analyzer (develop/test)
- PS_FAKE_L3_DELAY_MS=200    # simulate latency/timeout

## How it works

- The scanner’s L3 semantic evaluation calls the DeBERTa endpoint and obtains a risk score (confidence) when matched.
- Violations now include:
  - level: 1|2|3
  - confidence: float (only for L3 matches)
- The HTTP /scan path computes:
  - risk_score = max(L3 confidence across violations)
  - pattern_score = 1.0 if any Level 2 (regex) matched; 0.5 if only Level 1 matched; else 0.0
  - final_score = α·risk_score + β·pattern_score
  - If final_score >= PS_BLOCK_THRESHOLD and the request would otherwise be allowed, quarantine it (reason: policy_bridge_threshold).

## Rationale (internal only)

- A short, compact rationale line is logged for operators:
  "risk=<r>,pattern=<p>,final=<f>,rules=<n>"
- It is not exposed to users in API responses or headers.

## Conversation context (minimal)

- If the request includes X-PS-Conversation-ID, the enforcer stores the last prompt text (truncated) in memory (no secrets/content are logged).
- On subsequent requests within the same conversation, we compute:
  - conv_drift = high|low (based on Jaccard token distance)
  - conv_priv_jump = true|false (simple keyword heuristics)
- These flags are set in the scanner runtime context and can be used in RulePack conditions (when/unless) to escalate checks.

## RulePack notes

- Keep using Level-3 rules with confidence_threshold to control DeBERTa risk mapping per rule.
- Gated rules can reference runtime context keys:
  - conv_drift: ["high", "low"]
  - conv_priv_jump: ["true", "false"]

## Migration from external providers

- No external provider calls are made in this path.
- Ensure PS_DEBERTA_ENDPOINT is reachable and returns one of the supported JSON shapes:
  - [{"label":"PROMPT_INJECTION","score":0.98}, ...]
  - {"risk_score":0.92, "label":"malicious"}
  - {"labels":[...], "scores":[...]}

## Security considerations

- No chain-of-thought is exposed externally.
- Logs omit raw content and only record short rationale lines and numeric scores.
- For sensitive production environments, prefer a private/local inference endpoint.

