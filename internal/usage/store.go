package usage

import (
	"context"
	"time"
)

// Decision labels for usage accounting
const (
	DecisionAllow      = "allow"
	DecisionQuarantine = "quarantine"
	DecisionDeny       = "deny"
)

// UsageStore defines the interface for recording and querying usage windows.
type UsageStore interface {
	// Record increments counters for a single decision event at time t.
	Record(ctx context.Context, tenant, route, decision string, bytes int64, t time.Time) error
	// Query aggregates usage over a time window with an interval and optional group by dimensions.
	Query(ctx context.Context, q Query) (Result, error)
	// Close flushes and releases resources.
	Close(ctx context.Context) error
}

// Query specifies usage aggregation parameters.
type Query struct {
	Tenant   string
	Start    time.Time
	End      time.Time
	Interval Interval  // Minute, Hour, Day
	GroupBy  []GroupBy // Optional grouping: Tenant, Route, Decision
}

// Interval represents aggregation bucket size.
type Interval string

const (
	IntervalMinute Interval = "minute"
	IntervalHour   Interval = "hour"
	IntervalDay    Interval = "day"
)

// GroupBy represents supported grouping keys.
type GroupBy string

const (
	GroupByTenant   GroupBy = "tenant"
	GroupByRoute    GroupBy = "route"
	GroupByDecision GroupBy = "decision"
)

// Result contains aggregated rows over the requested window.
type Result struct {
	WindowStart time.Time `json:"window_start"`
	WindowEnd   time.Time `json:"window_end"`
	Rows        []Row     `json:"rows"`
}

// Row is a single aggregated record.
type Row struct {
	Tenant        string    `json:"tenant,omitempty"`
	Route         string    `json:"route,omitempty"`
	Decision      string    `json:"decision,omitempty"`
	IntervalStart time.Time `json:"interval_start"`
	Count         int64     `json:"count"`
	Bytes         int64     `json:"bytes"`
}
