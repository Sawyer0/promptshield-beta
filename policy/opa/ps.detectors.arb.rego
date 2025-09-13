package ps.detectors.arb

# Input:
# {
#   "events": [
#     {"detector_id":"...","decision":"block","reason":"...","confidence":0.92,"controls":{"GDPR":["Art.32"]}},
#     ...
#   ],
#   "config": {
#     "thresholds": { "default": 0.8, "by_reason": {"pii.email": 0.7} },
#     "precedence": ["block","quarantine","redact","alert","allow"]
#   }
# }

default decision := {
  "action": "allow",
  "reasons": [],
  "controls": {},
  "obligations": [],
  "policy_version": "v1"
}

threshold := t {
  t := input.config.thresholds.default
} else := t {
  some r
  t := input.config.thresholds.by_reason[r]
  r == reason
}

# Compute accepted events (meet threshold if confidence present; else accept)
accepted[e] {
  e := input.events[_]
  not e.confidence
} {
  e := input.events[_]
  e.confidence >= threshold
}

# Gather reasons and controls
reasons := {r | e := accepted[_]; r := e.reason}
controls := merge_controls({c | e := accepted[_]; c := e.controls})

# Precedence determination
precedence := p {
  p := input.config.precedence
} else := ["block","quarantine","redact","alert","allow"]

some_action(a) {
  e := accepted[_]
  a := e.decision
}

action := picked {
  picked := precedence[i]
  some_action(picked)
}

decision := {
  "action": action,
  "reasons": sort(array.retain(array.from_set(reasons), _)),
  "controls": controls,
  "obligations": obligations_for_action[action],
  "policy_version": "v1"
}

obligations_for_action := {
  "block": [],
  "quarantine": [],
  "redact": [{"type": "mask", "key": "pattern", "value": "(?i)(api[_-]?key|secret|token)\"?[:=]\"?.+"}],
  "alert": [],
  "allow": []
}

merge_controls(cs) = out {
  out := {}
  forall c in cs {
    out := merge(out, c)
  }
}

merge(a, b) = c {
  c := a
  keys := {k | k := object.keys(b)[_]} 
  forall k in keys {
    vs := array.concat(array.from_set(set(a[k]) | true), array.from_set(set(b[k]) | true))
    c[k] := array.retain(vs, _)
  }
}

