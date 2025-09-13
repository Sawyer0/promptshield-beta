package ps.compliance.bind

test_mapped_when_detectors_and_endpoints_present {
  data := {
    "catalog": {
      "detectors": {"promptguard_v2": true, "presidio_phi": true},
      "endpoints": {"chatbot-prod": true}
    }
  }
  input := {"binding": {"detectors": [{"id":"promptguard_v2"}], "endpoints": ["chatbot-prod"]}}
  result with input as input with data as data
  result.status == "mapped"
}

test_enforced_when_events_seen {
  data := {"catalog": {"detectors": {"a": true}, "endpoints": {"e": true}}}
  input := {"binding": {"detectors": [{"id":"a"}], "endpoints": ["e"], "stats": {"events_seen": 10}}}
  result with input as input with data as data
  result.status == "enforced"
}

