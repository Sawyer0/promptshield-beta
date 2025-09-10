import http from 'k6/http'
import { check, sleep } from 'k6'
import { Trend, Counter } from 'k6/metrics'

// Summary stats to include percentiles up to p99.9
export const options = {
  vus: __ENV.VUS ? Number(__ENV.VUS) : 50,
  duration: __ENV.DURATION || '10m',
  summaryTrendStats: ['avg','med','p(90)','p(95)','p(99)','p(99.9)','min','max'],
}

// Base config via env
const BASE = __ENV.BASE || 'http://localhost:8080'
const PATH = __ENV.PATH || '/check'
const FLAG_RATE = __ENV.FLAG_RATE ? Number(__ENV.FLAG_RATE) : 0.02 // 2% flagged by default
const PAYLOAD_CT = __ENV.PAYLOAD_CT || 'application/octet-stream'
const TENANT_ID = __ENV.TENANT_ID || '00000000-0000-0000-0000-000000000001'

// Payload mix: 70% 1–4KB, 25% 8–32KB, 5% 64–256KB
function pickBucket() {
  const r = Math.random()
  if (r < 0.70) return { bucket: '1-4KB',    min: 1*1024,  max: 4*1024 }
  if (r < 0.95) return { bucket: '8-32KB',   min: 8*1024,  max: 32*1024 }
  return            { bucket: '64-256KB', min: 64*1024, max: 256*1024 }
}
function randInt(min, max) { return Math.floor(Math.random() * (max - min + 1)) + min }
function payloadOf(n) {
  // Produce a text payload of size n (bytes). Keep it simple and deterministic.
  return 'A'.repeat(n)
}

// Metrics: overall, per-bucket, flagged vs non-flagged, and queue timing
const lat_overall_ms = new Trend('lat_overall_ms')
const lat_1_4KB_ms = new Trend('lat_1_4KB_ms')
const lat_8_32KB_ms = new Trend('lat_8_32KB_ms')
const lat_64_256KB_ms = new Trend('lat_64_256KB_ms')
const lat_flag_true_ms = new Trend('lat_flag_true_ms')
const lat_flag_false_ms = new Trend('lat_flag_false_ms')
const l3_queue_ms = new Trend('l3_queue_ms')
const flagged_requests = new Counter('flagged_requests')

export default function () {
  const sel = pickBucket()
  const size = randInt(sel.min, sel.max)
  const flagged = Math.random() < FLAG_RATE
  // If flagged, inject a marker to trigger L3 semantic rule
  const body = flagged ? ("[FAKE_MATCH]\n" + payloadOf(size)) : payloadOf(size)

  const url = `${BASE}${PATH}`
  const params = {
    headers: {
      'Content-Type': PAYLOAD_CT,
      'X-PS-Tenant-ID': TENANT_ID,
      // Optional hook if server supports forcing flagged/L3 path
      ...(flagged ? { 'X-PS-Force-L3': '1' } : {}),
    },
    tags: {
      bucket: sel.bucket,
      flagged: String(flagged),
    },
  }

  const res = http.post(url, body, params)

  // Record basic correctness
  check(res, {
    'status is 2xx/4xx': r => (r.status >= 200 && r.status < 300) || (r.status >= 400 && r.status < 500),
  })

  // Latency metrics
  const d = res.timings.duration
  lat_overall_ms.add(d)
  if (flagged) lat_flag_true_ms.add(d); else lat_flag_false_ms.add(d)
  switch (sel.bucket) {
    case '1-4KB': lat_1_4KB_ms.add(d); break
    case '8-32KB': lat_8_32KB_ms.add(d); break
    case '64-256KB': lat_64_256KB_ms.add(d); break
  }
  if (flagged) flagged_requests.add(1)

  // Optional: parse Server-Timing queue time if provided by service
  const st = res.headers['Server-Timing'] || ''
  // e.g., "queue;dur=1.7, l3;dur=45.2"
  const m = /(?:^|,\s*)queue;dur=([\d.]+)/i.exec(st)
  if (m) l3_queue_ms.add(Number(m[1]))

  // Light pacing to avoid hot spin on tiny payloads
  sleep(0.001)
}

