package ps.detectors.validate

default valid := false

required_fields := [
  "schema_version",
  "detector_id",
  "version",
  "endpoint",
  "direction",
  "decision",
  "reason",
  "event_ts",
  "correlation_id"
]

schema_ok {
  input.schema_version == "detector_event/v1"
}

fields_ok {
  every rf in required_fields {
    input[rf]
  }
}

direction_ok {
  input.direction == "request";
} {
  input.direction == "response";
} {
  input.direction == "tool";
} {
  input.direction == "model";
}

decision_ok {
  input.decision == "allow";
} {
  input.decision == "alert";
} {
  input.decision == "redact";
} {
  input.decision == "quarantine";
} {
  input.decision == "block";
}

valid {
  schema_ok
  fields_ok
  direction_ok
  decision_ok
}

