package decisions

import (
	"math"
	"sort"
	"sync"
	"time"
)

// DefaultCapacity is the number of records a Log keeps when none is given.
const DefaultCapacity = 500

// Query filters a Recent listing.
type Query struct {
	// Limit caps the number of records returned; zero or less means all retained.
	Limit int
	// Decision keeps only records with this outcome when set.
	Decision string
}

// Stats summarises decisions since the log was created. Counts are lifetime
// totals; latency percentiles and the per-minute rate use retained records.
type Stats struct {
	Total        int64            `json:"total"`
	ByDecision   map[string]int64 `json:"by_decision"`
	BlockRate    float64          `json:"block_rate"`
	LatencyP50Ms float64          `json:"latency_p50_ms"`
	LatencyP95Ms float64          `json:"latency_p95_ms"`
	LastMinute   int              `json:"last_minute"`
	Retained     int              `json:"retained"`
	Capacity     int              `json:"capacity"`
	Since        time.Time        `json:"since"`
}

// Log is a fixed-size ring of recent decisions. It is safe for concurrent use.
type Log struct {
	mu      sync.RWMutex
	buf     []Record
	next    int
	size    int
	total   int64
	counts  map[string]int64
	started time.Time
	now     func() time.Time
}

// NewLog returns a Log that retains up to capacity records.
func NewLog(capacity int) *Log {
	if capacity <= 0 {
		capacity = DefaultCapacity
	}
	return &Log{
		buf:     make([]Record, capacity),
		counts:  map[string]int64{Allow: 0, Deny: 0, Quarantine: 0},
		started: time.Now().UTC(),
		now:     time.Now,
	}
}

// Add stores a record, evicting the oldest one when the log is full.
func (l *Log) Add(r Record) {
	if r.Time.IsZero() {
		r.Time = l.now().UTC()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf[l.next] = r
	l.next = (l.next + 1) % len(l.buf)
	if l.size < len(l.buf) {
		l.size++
	}
	l.total++
	l.counts[r.Decision]++
}

// Recent returns retained records, newest first, filtered by q.
func (l *Log) Recent(q Query) []Record {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Record, 0, l.size)
	for i := 1; i <= l.size; i++ {
		r := l.buf[(l.next-i+len(l.buf))%len(l.buf)]
		if q.Decision != "" && r.Decision != q.Decision {
			continue
		}
		out = append(out, r)
		if q.Limit > 0 && len(out) == q.Limit {
			break
		}
	}
	return out
}

// Get returns the retained record with the given request id.
func (l *Log) Get(requestID string) (Record, bool) {
	for _, r := range l.Recent(Query{}) {
		if r.RequestID == requestID {
			return r, true
		}
	}
	return Record{}, false
}

// Stats computes the current summary.
func (l *Log) Stats() Stats {
	records := l.Recent(Query{})
	l.mu.RLock()
	s := Stats{
		Total:      l.total,
		ByDecision: make(map[string]int64, len(l.counts)),
		Retained:   l.size,
		Capacity:   len(l.buf),
		Since:      l.started,
	}
	for k, v := range l.counts {
		s.ByDecision[k] = v
	}
	l.mu.RUnlock()

	if s.Total > 0 {
		blocked := s.ByDecision[Deny] + s.ByDecision[Quarantine]
		s.BlockRate = float64(blocked) / float64(s.Total)
	}
	cutoff := l.now().Add(-time.Minute)
	latencies := make([]float64, 0, len(records))
	for _, r := range records {
		latencies = append(latencies, r.LatencyMs)
		if r.Time.After(cutoff) {
			s.LastMinute++
		}
	}
	s.LatencyP50Ms = percentile(latencies, 0.50)
	s.LatencyP95Ms = percentile(latencies, 0.95)
	return s
}

// percentile returns the nearest-rank percentile of values (p in 0..1).
func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	rank := int(math.Ceil(p*float64(len(sorted)))) - 1
	if rank < 0 {
		rank = 0
	}
	return sorted[rank]
}
