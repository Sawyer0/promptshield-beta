---
title: The Ultimate 30-Second Demo
description: Concise script: show vulnerability, apply one-line change, prove protection, end on metrics.
---

## Duration
30 seconds

## Script
0–5s — Vulnerability
- Show the attack working (prompt injection reveals system prompt).

5–10s — Add PromptShield
- Show one-line change:

```diff
- api_url := "https://api.openai.com/v1/chat"
+ api_url := "http://promptshield.local:8080/v1/chat"
```

10–20s — Protection
- Repeat attack → blocked. On-screen: “🛡️ Threat blocked: Instruction injection attempt.”

20–30s — Proof & Close
- Flash dashboard metrics: “2,847 attacks blocked this week”, “~5ms added latency”.
- VO: “PromptShield. Infrastructure-level AI security.”

