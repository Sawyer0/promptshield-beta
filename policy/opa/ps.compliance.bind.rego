package ps.compliance.bind

# Validates requirement bindings against detector catalog and endpoint list.

default result := {
  "status": "unmapped",
  "errors": [],
  "missing": {"detectors": [], "endpoints": []}
}

catalog := {d | d := data.catalog.detectors[_]}
endpoints := {e | e := data.catalog.endpoints[_]}

missing_detectors := {d | d := input.binding.detectors[_].id; not catalog[d]}
missing_endpoints := {ep | ep := input.binding.endpoints[_]; not endpoints[ep]}

mapped {
  count(input.binding.detectors) > 0
  count(input.binding.endpoints) > 0
}

enforced {
  mapped
  input.binding.stats.events_seen > 0
}

status := s {
  enforced
  s := "enforced"
} else := s {
  mapped
  s := "mapped"
} else := s {
  s := "unmapped"
}

result := {
  "status": status,
  "errors": [],
  "missing": {
    "detectors": array.from_set(missing_detectors),
    "endpoints": array.from_set(missing_endpoints)
  }
}

