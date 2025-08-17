---
title: Power Demo C — Performance Proof
description: Side-by-side latency measurement showing minimal overhead.
---

## Objective
Show negligible latency impact when routing through PromptShield Gateway.

## Visual
- Two timers: “Direct to OpenAI” vs “Via PromptShield”.

## Script
- Measure typical prompt round-trip time.
- Example results:
  - Direct to OpenAI: 487ms
  - Through PromptShield: 492ms
- On-screen: “~5ms added latency while scanning every request.”

## Notes
- Warm the model cache to avoid cold-start skew.
- Run multiple trials, show P50/P95 bars.

