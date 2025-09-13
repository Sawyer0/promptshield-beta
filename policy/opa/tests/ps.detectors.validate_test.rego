package ps.detectors.validate

test_valid_event_passes {
  input := {
    "schema_version": "detector_event/v1",
    "detector_id": "promptguard_v2",
    "version": "2.3.1",
    "endpoint": "chatbot-prod",
    "direction": "response",
    "decision": "block",
    "reason": "pii.email",
    "event_ts": "2025-09-13T14:02:55Z",
    "correlation_id": "req_abc123"
  }
  valid with input as input
}

test_bad_event_rejected_missing_fields {
  not valid with input as {"schema_version":"detector_event/v1"}
}

