package usage

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// RedisUsageStore implements UsageStore using Redis hashes keyed per minute/tenant/route.
// Key format: <prefix>:usage:<ts_minute>:<tenant>:<route>
type RedisUsageStore struct {
	rdb    *redis.Client
	prefix string
	// TTL for per-minute keys; default 35 days when zero
	ttl time.Duration
}

func NewRedisUsageStore(rdb *redis.Client, prefix string, ttl time.Duration) *RedisUsageStore {
	if prefix == "" {
		prefix = "ps"
	}
	if ttl <= 0 {
		ttl = 35 * 24 * time.Hour
	}
	return &RedisUsageStore{rdb: rdb, prefix: prefix, ttl: ttl}
}

func usageKey(prefix string, ts int64, tenant, route string) string {
	if tenant == "" {
		tenant = "default"
	}
	if route == "" {
		route = "default"
	}
	// Avoid ':' inside tenant/route by replacing with '|'
	tenant = strings.ReplaceAll(tenant, ":", "|")
	route = strings.ReplaceAll(route, ":", "|")
	return fmt.Sprintf("%s:usage:%d:%s:%s", prefix, ts, tenant, route)
}

func (s *RedisUsageStore) Record(ctx context.Context, tenant, route, decision string, bytes int64, t time.Time) error {
	ts := floorToMinute(t).Unix()
	key := usageKey(s.prefix, ts, tenant, route)
	var col string
	switch decision {
	case DecisionQuarantine:
		col = "quarantine"
	case DecisionDeny:
		col = "deny"
	default:
		col = "allow"
	}
	pipe := s.rdb.TxPipeline()
	pipe.HIncrBy(ctx, key, col, 1)
	pipe.HIncrBy(ctx, key, "bytes", bytes)
	pipe.Expire(ctx, key, s.ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// RecordTokens records usage with detailed token tracking for LLM billing/observability
func (s *RedisUsageStore) RecordTokens(ctx context.Context, record Record) error {
	ts := floorToMinute(record.Timestamp).Unix()
	
	// Enhanced key format includes provider and model for granular tracking
	providerKey := fmt.Sprintf("%s:tokens:%d:%s:%s:%s:%s", 
		s.prefix, ts, record.Tenant, record.Route, record.Provider, record.Model)
	
	var col string
	switch record.Decision {
	case DecisionQuarantine:
		col = "quarantine"
	case DecisionDeny:
		col = "deny"
	default:
		col = "allow"
	}
	
	pipe := s.rdb.TxPipeline()
	// Basic counters
	pipe.HIncrBy(ctx, providerKey, col, 1)
	pipe.HIncrBy(ctx, providerKey, "bytes", record.Bytes)
	
	// Token counters for billing
	if record.PromptTokens > 0 {
		pipe.HIncrBy(ctx, providerKey, "prompt_tokens", record.PromptTokens)
	}
	if record.CompletionTokens > 0 {
		pipe.HIncrBy(ctx, providerKey, "completion_tokens", record.CompletionTokens)
	}
	if record.TotalTokens > 0 {
		pipe.HIncrBy(ctx, providerKey, "total_tokens", record.TotalTokens)
	}
	
	pipe.Expire(ctx, providerKey, s.ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *RedisUsageStore) Close(ctx context.Context) error { return s.rdb.Close() }

func (s *RedisUsageStore) Query(ctx context.Context, q Query) (Result, error) {
	if q.End.Before(q.Start) {
		return Result{}, fmt.Errorf("invalid window")
	}
	start := floorToMinute(q.Start).Unix()
	end := floorToMinute(q.End).Unix()
	bucketSize := int64(60) // minute
	switch q.Interval {
	case IntervalHour:
		bucketSize = 3600
	case IntervalDay:
		bucketSize = 86400
	}
	type keyAgg struct {
		bucket int64
		tenant string
		route  string
	}
	includeTenant := false
	includeRoute := false
	for _, g := range q.GroupBy {
		if g == GroupByTenant {
			includeTenant = true
		}
		if g == GroupByRoute {
			includeRoute = true
		}
	}
	acc := make(map[keyAgg][5]int64) // [0]=count, [1]=bytes, [2]=prompt_tokens, [3]=completion_tokens, [4]=total_tokens
	scan := func(pattern string) ([]string, error) {
		var (
			cursor uint64
			keys   []string
		)
		for {
			ks, cur, err := s.rdb.Scan(ctx, cursor, pattern, 1000).Result()
			if err != nil {
				return nil, err
			}
			keys = append(keys, ks...)
			cursor = cur
			if cursor == 0 {
				break
			}
		}
		return keys, nil
	}
	for ts := start; ts < end; ts += 60 {
		pattern := fmt.Sprintf("%s:usage:%d:*", s.prefix, ts)
		keys, err := scan(pattern)
		if err != nil {
			return Result{}, err
		}
		if len(keys) == 0 {
			continue
		}
		// Batch HMGET
		pipe := s.rdb.Pipeline()
		cmds := make([]*redis.SliceCmd, 0, len(keys))
		for range keys {
			cmds = append(cmds, pipe.HMGet(ctx, keys[len(cmds)], "allow", "quarantine", "deny", "bytes", "prompt_tokens", "completion_tokens", "total_tokens"))
		}
		if _, err := pipe.Exec(ctx); err != nil {
			return Result{}, err
		}
		for i, key := range keys {
			parts := strings.Split(key, ":")
			if len(parts) < 5 {
				continue
			}
			kTenant := parts[len(parts)-2]
			kRoute := parts[len(parts)-1]
			// Only include requested tenant if specified
			if q.Tenant != "" && q.Tenant != strings.ReplaceAll(kTenant, "|", ":") {
				continue
			}
			vals := cmds[i].Val()
			var allow, quarantine, deny, bytesVal, promptTokens, completionTokens, totalTokens int64
			if len(vals) >= 4 {
				allow = toInt64(vals[0])
				quarantine = toInt64(vals[1])
				deny = toInt64(vals[2])
				bytesVal = toInt64(vals[3])
				if len(vals) >= 7 {
					promptTokens = toInt64(vals[4])
					completionTokens = toInt64(vals[5])
					totalTokens = toInt64(vals[6])
				}
			}
			bucket := (ts / bucketSize) * bucketSize
			aggKey := keyAgg{bucket: bucket}
			if includeTenant {
				aggKey.tenant = strings.ReplaceAll(kTenant, "|", ":")
			}
			if includeRoute {
				aggKey.route = strings.ReplaceAll(kRoute, "|", ":")
			}
			prev := acc[aggKey]
			prev[0] += allow + quarantine + deny
			prev[1] += bytesVal
			prev[2] += promptTokens
			prev[3] += completionTokens
			prev[4] += totalTokens
			acc[aggKey] = prev
		}
	}
	// Build rows
	rows := make([]Row, 0, len(acc))
	for k, v := range acc {
		r := Row{
			IntervalStart:    time.Unix(k.bucket, 0).UTC(), 
			Count:            v[0], 
			Bytes:            v[1],
			PromptTokens:     v[2],
			CompletionTokens: v[3],
			TotalTokens:      v[4],
		}
		if includeTenant {
			r.Tenant = k.tenant
		}
		if includeRoute {
			r.Route = k.route
		}
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].IntervalStart.Before(rows[j].IntervalStart) })
	return Result{WindowStart: floorToMinute(q.Start), WindowEnd: floorToMinute(q.End), Rows: rows}, nil
}

func toInt64(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case string:
		if t == "" {
			return 0
		}
		if n, err := strconv.ParseInt(t, 10, 64); err == nil {
			return n
		}
		return 0
	case []byte:
		if n, err := strconv.ParseInt(string(t), 10, 64); err == nil {
			return n
		}
		return 0
	case nil:
		return 0
	default:
		return 0
	}
}
