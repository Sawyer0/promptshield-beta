package api

import (
	"context"
	"sync"
	"time"

	"github.com/promptshield/promptshield/internal/usage"
)

// UsageCache implements usage.UsageStore with in-memory caching for high-speed access.
// This is intended for short-term caching; production deployments should use
// persistent storage like PostgreSQL or Redis for usage tracking.
type UsageCache struct {
	mu      sync.RWMutex
	records []usage.Row // Store usage rows in memory cache
}

func NewUsageCache() usage.UsageStore {
	return &UsageCache{
		records: make([]usage.Row, 0),
	}
}

func (s *UsageCache) Record(ctx context.Context, tenant, route, decision string, bytes int64, t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Create a usage row from the parameters
	row := usage.Row{
		Tenant:        tenant,
		Route:         route,
		Decision:      decision,
		IntervalStart: t,
		Count:         1,
		Bytes:         bytes,
	}
	
	s.records = append(s.records, row)
	return nil
}

// RecordTokens records usage with detailed token tracking for LLM billing/observability
func (s *UsageCache) RecordTokens(ctx context.Context, record usage.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Create a usage row from the record
	row := usage.Row{
		Tenant:           record.Tenant,
		Route:            record.Route,
		Decision:         record.Decision,
		Provider:         record.Provider,
		Model:            record.Model,
		IntervalStart:    record.Timestamp,
		Count:            1,
		Bytes:            record.Bytes,
		PromptTokens:     record.PromptTokens,
		CompletionTokens: record.CompletionTokens,
		TotalTokens:      record.TotalTokens,
	}
	
	s.records = append(s.records, row)
	return nil
}

func (s *UsageCache) Query(ctx context.Context, query usage.Query) (usage.Result, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var filteredRows []usage.Row
	
	// Filter records by time range and tenant
	for _, row := range s.records {
		if row.IntervalStart.Before(query.Start) || row.IntervalStart.After(query.End) {
			continue
		}
		
		if query.Tenant != "" && row.Tenant != query.Tenant {
			continue
		}
		
		filteredRows = append(filteredRows, row)
	}
	
	// Aggregate the filtered rows by interval and groupBy criteria
	aggregatedRows := s.aggregateRows(filteredRows, query)
	
	return usage.Result{
		WindowStart: query.Start,
		WindowEnd:   query.End,
		Rows:        aggregatedRows,
	}, nil
}

func (s *UsageCache) aggregateRows(rows []usage.Row, query usage.Query) []usage.Row {
	// Group rows by time window and groupBy criteria
	groups := make(map[string]*usage.Row)
	
	for _, row := range rows {
		// Truncate timestamp to interval boundary
		var windowStart time.Time
		switch query.Interval {
		case usage.IntervalMinute:
			windowStart = row.IntervalStart.Truncate(time.Minute)
		case usage.IntervalHour:
			windowStart = row.IntervalStart.Truncate(time.Hour)
		case usage.IntervalDay:
			windowStart = time.Date(row.IntervalStart.Year(), row.IntervalStart.Month(), 
				row.IntervalStart.Day(), 0, 0, 0, 0, row.IntervalStart.Location())
		default:
			windowStart = row.IntervalStart.Truncate(time.Hour)
		}
		
		// Create group key based on window and groupBy criteria
		groupKey := windowStart.Format(time.RFC3339)
		
		for _, groupBy := range query.GroupBy {
			switch groupBy {
			case usage.GroupByTenant:
				groupKey += "|tenant:" + row.Tenant
			case usage.GroupByRoute:
				groupKey += "|route:" + row.Route
			case usage.GroupByDecision:
				groupKey += "|decision:" + row.Decision
			}
		}
		
		// Create or update group
		if existingRow, exists := groups[groupKey]; exists {
			existingRow.Count += row.Count
			existingRow.Bytes += row.Bytes
			existingRow.PromptTokens += row.PromptTokens
			existingRow.CompletionTokens += row.CompletionTokens
			existingRow.TotalTokens += row.TotalTokens
		} else {
			groups[groupKey] = &usage.Row{
				IntervalStart:    windowStart,
				Tenant:           row.Tenant,
				Route:            row.Route,
				Decision:         row.Decision,
				Provider:         row.Provider,
				Model:            row.Model,
				Count:            row.Count,
				Bytes:            row.Bytes,
				PromptTokens:     row.PromptTokens,
				CompletionTokens: row.CompletionTokens,
				TotalTokens:      row.TotalTokens,
			}
		}
	}
	
	// Convert map to slice
	var aggregatedRows []usage.Row
	for _, row := range groups {
		aggregatedRows = append(aggregatedRows, *row)
	}
	
	return aggregatedRows
}

// Close implements the usage.UsageStore interface
func (s *UsageCache) Close(ctx context.Context) error {
	return nil
}