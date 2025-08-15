# OpenAI Provider (Semantic L3)

OpenAI integration provides Level‑3 semantic analysis used when a RulePack rule specifies `level: 3` with a `semantic` section.

The adapter uses the official SDK with retryable HTTP, OpenTelemetry instrumentation, bounded concurrency, and an LRU cache.

## Enablement

Set the following environment variables:

- `PS_SEMANTIC_ENABLED=true`
- `PS_SEMANTIC_PROVIDER=openai`
- Provide an API key via one of:
  - OS keychain
  - `OPENAI_API_KEY` (or `PS_OPENAI_API_KEY`)

Optional tuning (applies to all semantic providers):
- `PS_SEMANTIC_MAX_CONCURRENCY` (default 2)
- `PS_SEMANTIC_CACHE_SIZE` (default 1000)
- `PS_SEMANTIC_CACHE_TTL` (e.g., `15m` default)

Base URL defaults to `https://api.openai.com/v1`.

## RulePack requirements

Level‑3 rules must specify both `semantic.model` and `semantic.analysis_prompt`. No defaults are provided. The system returns an error if `analysis_prompt` is missing.

Example:

```yaml
rules:
  - id: pii-semantic-personal-info
    level: 3
    severity: HIGH
    semantic:
      model: gpt-4o-mini
      max_tokens: 50
      temperature: 0.0
      analysis_prompt: |
        You are a strict binary classifier. Return ONLY one of these tokens:
        "VIOLATION" or "SAFE".
        Classify the following input for personal information leakage.
        Input:
        {input}
    response:
      action: quarantine
      message: "Semantic PII detected"
```

The `analysis_prompt` must instruct the model to respond with a single token `VIOLATION` or `SAFE`. The adapter interprets only these two values; any other response is treated as SAFE with low confidence.

## Behavior

- Concurrency guard limits in‑flight semantic requests (default 2)
- Rate limiting: default ~10 RPS with burst 20
- Cache stores recent classifications keyed by `(model, normalized_input, prompt)` with TTL
- Inputs are redacted/truncated before transmission to reduce sensitive data exposure
- OpenTelemetry traces are emitted for HTTP calls when OTel is configured

## Troubleshooting

- Ensure `PS_SEMANTIC_ENABLED=true` and `PS_SEMANTIC_PROVIDER=openai`
- Provide the API key via OS keychain or `OPENAI_API_KEY`
- Confirm RulePack Level‑3 rules include both `model` and `analysis_prompt`
- Check logs for `semantic request` / `semantic response` entries
