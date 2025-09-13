package ps.detectors.arb

test_any_block_blocks {
  input := {
    "events": [
      {"detector_id": "a", "decision": "alert", "reason": "pii.email", "confidence": 0.95},
      {"detector_id": "b", "decision": "block", "reason": "pii.email", "confidence": 0.91}
    ],
    "config": {"thresholds": {"default": 0.8}, "precedence": ["block","quarantine","redact","alert","allow"]}
  }
  decision with input as input
  decision.action == "block"
}

test_precedence_redact_over_alert {
  input := {
    "events": [
      {"detector_id": "a", "decision": "alert", "reason": "pii.email", "confidence": 0.95},
      {"detector_id": "b", "decision": "redact", "reason": "secrets", "confidence": 0.90}
    ],
    "config": {"thresholds": {"default": 0.5}}
  }
  decision with input as input
  decision.action == "redact"
}

