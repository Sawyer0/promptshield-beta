package types

import (
	"time"
)

// RulePack models a user-authored YAML rule pack per v2 schema
// From internal/rules/types.go
type RulePack struct {
	APIVersion  string            `yaml:"apiVersion" json:"apiVersion"`
	Kind        string            `yaml:"kind" json:"kind"`
	Metadata    RulePackMetadata  `yaml:"metadata" json:"metadata"`
	Rules       []Rule            `yaml:"rules" json:"rules"`
	Extends     []string          `yaml:"extends" json:"extends,omitempty"`
	Imports     []string          `yaml:"imports" json:"imports,omitempty"`
	Composition *Composition      `yaml:"composition" json:"composition,omitempty"`
	Performance *Performance      `yaml:"performance" json:"performance,omitempty"`
	Overrides   []Override        `yaml:"overrides" json:"overrides,omitempty"`
	Context     map[string]string `yaml:"context" json:"context,omitempty"`
	// SourcePath is the filesystem path the pack was loaded from (not serialized)
	SourcePath string `yaml:"-" json:"-"`
}

// RulePackMetadata contains metadata about a rule pack
type RulePackMetadata struct {
	Name        string   `yaml:"name" json:"name"`
	Version     string   `yaml:"version" json:"version"`
	Description string   `yaml:"description" json:"description,omitempty"`
	Author      string   `yaml:"author" json:"author,omitempty"`
	License     string   `yaml:"license" json:"license,omitempty"`
	Homepage    string   `yaml:"homepage" json:"homepage,omitempty"`
	Repository  string   `yaml:"repository" json:"repository,omitempty"`
	Tags        []string `yaml:"tags" json:"tags,omitempty"`
}

// Rule represents a single scanning rule
type Rule struct {
	ID       string `yaml:"id" json:"id"`
	Name     string `yaml:"name" json:"name"`
	Level    int    `yaml:"level" json:"level"`       // 1=keyword, 2=regex, 3=semantic
	Severity string `yaml:"severity" json:"severity"` // INFO|WARNING|ERROR|CRITICAL
	Category string `yaml:"category" json:"category,omitempty"`
	Enabled  *bool  `yaml:"enabled" json:"enabled,omitempty"`
	Verifier string `yaml:"verifier" json:"verifier,omitempty"`

	// Rule matchers
	Keywords []string  `yaml:"keywords" json:"keywords,omitempty"`
	Patterns []Pattern `yaml:"patterns" json:"patterns,omitempty"`
	Semantic *Semantic `yaml:"semantic" json:"semantic,omitempty"`
	Fallback *Fallback `yaml:"fallback" json:"fallback,omitempty"`

	// Rule behavior
	Logic    string        `yaml:"logic" json:"logic,omitempty"` // any|all|custom
	Options  RuleOptions   `yaml:"options" json:"options,omitempty"`
	Response *RuleResponse `yaml:"response" json:"response,omitempty"`
	Cache    *RuleCache    `yaml:"cache" json:"cache,omitempty"`

	// Context gating (optional)
	When   *Condition `yaml:"when" json:"when,omitempty"`
	Unless *Condition `yaml:"unless" json:"unless,omitempty"`

	// Optional per-rule timeout (e.g., "50ms")
	Timeout string `yaml:"timeout" json:"timeout,omitempty"`
}

// RuleOptions configures rule matching behavior
type RuleOptions struct {
	CaseSensitive bool `yaml:"case_sensitive" json:"case_sensitive,omitempty"`
	WholeWord     bool `yaml:"whole_word" json:"whole_word,omitempty"`
}

// Pattern represents a regex pattern for Level 2 rules
type Pattern struct {
	Name     string   `yaml:"name" json:"name"`
	Regex    string   `yaml:"regex" json:"regex"`
	Flags    []string `yaml:"flags" json:"flags,omitempty"`
	Verifier string   `yaml:"verifier" json:"verifier,omitempty"`
}

// Condition gates rule execution using merged context
// A rule matches the condition if for every key in Match, the runtime context
// contains at least one of the listed values (OR within a key, AND across keys)
type Condition struct {
	Match map[string][]string `yaml:"match" json:"match"`
}

// Semantic configuration for Level 3 rules
type Semantic struct {
	Model               string  `yaml:"model" json:"model"`
	Temperature         float64 `yaml:"temperature" json:"temperature,omitempty"`
	MaxTokens           int     `yaml:"max_tokens" json:"max_tokens,omitempty"`
	AnalysisPrompt      string  `yaml:"analysis_prompt" json:"analysis_prompt"`
	ConfidenceThreshold float64 `yaml:"confidence_threshold" json:"confidence_threshold,omitempty"`
	FallbackOnError     bool    `yaml:"fallback_on_error" json:"fallback_on_error,omitempty"`
}

// Fallback defines fallback behavior when semantic analysis fails
type Fallback struct {
	Patterns []Pattern `yaml:"patterns" json:"patterns"`
	Action   string    `yaml:"action" json:"action"`
}

// RuleResponse defines how to respond when a rule matches
type RuleResponse struct {
	Action      string `yaml:"action" json:"action"` // allow, deny, quarantine, redact
	Message     string `yaml:"message" json:"message,omitempty"`
	Replacement string `yaml:"replacement" json:"replacement,omitempty"` // For redaction
}

// RuleCache configures caching for rule results
type RuleCache struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	TTL        string `yaml:"ttl" json:"ttl,omitempty"`
	MaxEntries int    `yaml:"max_entries" json:"max_entries,omitempty"`
}

// Composition defines how rules are composed together
type Composition struct {
	Strategy string `yaml:"strategy" json:"strategy"`                     // all_matches, first_match, priority_order
	Priority int    `yaml:"priority,omitempty" json:"priority,omitempty"` // Higher values = higher priority
}

// Performance configures performance limits and timeouts
type Performance struct {
	MaxLength        int    `yaml:"max_length" json:"max_length,omitempty"`
	Timeout          string `yaml:"timeout" json:"timeout,omitempty"`
	PerRuleTimeout   string `yaml:"per_rule_timeout" json:"per_rule_timeout,omitempty"`
	TotalScanTimeout string `yaml:"total_scan_timeout" json:"total_scan_timeout,omitempty"`
	Gate             struct {
		Enabled     bool `yaml:"enabled" json:"enabled"`
		MinTokenLen int  `yaml:"min_token_len" json:"min_token_len,omitempty"`
	} `yaml:"gate" json:"gate,omitempty"`
}

// Override allows overriding specific rule properties
type Override struct {
	RuleID   string `yaml:"rule_id" json:"rule_id"`
	Severity string `yaml:"severity" json:"severity,omitempty"`
	Enabled  *bool  `yaml:"enabled" json:"enabled,omitempty"`
}

// RulePackInfo represents metadata about a rulepack (from contracts/repository.go)
type RulePackInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Author      string `json:"author,omitempty"`
	RuleCount   int    `json:"rule_count"`
	LoadedAt    string `json:"loaded_at,omitempty"`
}

// CompiledRule represents a rule that has been compiled and optimized
type CompiledRule struct {
	Original   Rule                   `json:"original"`
	ID         string                 `json:"id"`
	Level      int                    `json:"level"`
	Severity   ViolationSeverity      `json:"severity"`
	CompiledAt string                 `json:"compiled_at"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// RuleEvaluationResult represents the result of evaluating a single rule
type RuleEvaluationResult struct {
	RuleID     string  `json:"rule_id"`
	Matched    bool    `json:"matched"`
	Confidence float64 `json:"confidence,omitempty"`
	Evidence   string  `json:"evidence,omitempty"`
	Location   string  `json:"location,omitempty"`
	LatencyMs  int64   `json:"latency_ms"`
	CacheHit   bool    `json:"cache_hit,omitempty"`
	Error      string  `json:"error,omitempty"`
}

// ScanContext provides context for rule evaluation
type ScanContext struct {
	TenantID    string                 `json:"tenant_id,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	Provider    Provider               `json:"provider,omitempty"`
	Model       string                 `json:"model,omitempty"`
	Endpoint    string                 `json:"endpoint,omitempty"`
	ContentType string                 `json:"content_type,omitempty"`
	UserContext map[string]interface{} `json:"user_context,omitempty"`
}

// RuleLevel represents the level of a rule (1=keyword, 2=regex, 3=semantic)
type RuleLevel int

const (
	RuleLevelKeyword  RuleLevel = 1
	RuleLevelRegex    RuleLevel = 2
	RuleLevelSemantic RuleLevel = 3
)

// RuleMatch represents a match found by a rule
type RuleMatch struct {
	RuleID     string                 `json:"rule_id"`
	RuleName   string                 `json:"rule_name"`
	Level      RuleLevel              `json:"level"`
	Severity   ViolationSeverity      `json:"severity"`
	Category   string                 `json:"category,omitempty"`
	Matched    bool                   `json:"matched"`
	Confidence float64                `json:"confidence,omitempty"`
	Evidence   string                 `json:"evidence,omitempty"`
	Location   string                 `json:"location,omitempty"`
	StartPos   int                    `json:"start_pos,omitempty"`
	EndPos     int                    `json:"end_pos,omitempty"`
	LatencyMs  int64                  `json:"latency_ms"`
	CacheHit   bool                   `json:"cache_hit,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// CompiledRuleSet represents a set of compiled rules for efficient evaluation
type CompiledRuleSet struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	CompiledAt  time.Time              `json:"compiled_at"`
	Rules       []*CompiledRule        `json:"rules"`
	Level1Rules []*CompiledRule        `json:"level1_rules,omitempty"`
	Level2Rules []*CompiledRule        `json:"level2_rules,omitempty"`
	Level3Rules []*CompiledRule        `json:"level3_rules,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Performance *Performance           `json:"performance,omitempty"`
}

// CompiledPattern represents a compiled pattern for efficient matching
type CompiledPattern struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // "regex", "keyword", "semantic"
	Pattern     string                 `json:"pattern"`
	CompiledAt  time.Time              `json:"compiled_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Performance *PatternPerformance    `json:"performance,omitempty"`
}

// CompiledRegex represents a compiled regex pattern
type CompiledRegex struct {
	ID          string                 `json:"id"`
	Pattern     string                 `json:"pattern"`
	Flags       []string               `json:"flags,omitempty"`
	CompiledAt  time.Time              `json:"compiled_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Performance *PatternPerformance    `json:"performance,omitempty"`
}

// CompiledKeywords represents compiled keyword patterns
type CompiledKeywords struct {
	ID            string                 `json:"id"`
	Keywords      []string               `json:"keywords"`
	CaseSensitive bool                   `json:"case_sensitive"`
	WholeWord     bool                   `json:"whole_word"`
	CompiledAt    time.Time              `json:"compiled_at"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Performance   *PatternPerformance    `json:"performance,omitempty"`
}

// CompiledSemantic represents compiled semantic analysis configuration
type CompiledSemantic struct {
	ID                  string                 `json:"id"`
	Model               string                 `json:"model"`
	AnalysisPrompt      string                 `json:"analysis_prompt"`
	Temperature         float64                `json:"temperature,omitempty"`
	MaxTokens           int                    `json:"max_tokens,omitempty"`
	ConfidenceThreshold float64                `json:"confidence_threshold,omitempty"`
	FallbackOnError     bool                   `json:"fallback_on_error,omitempty"`
	CompiledAt          time.Time              `json:"compiled_at"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	Performance         *PatternPerformance    `json:"performance,omitempty"`
}

// PatternPerformance contains performance metrics for a pattern
type PatternPerformance struct {
	CompilationTimeMs  int64     `json:"compilation_time_ms"`
	AverageMatchTimeMs float64   `json:"average_match_time_ms"`
	MatchCount         int64     `json:"match_count"`
	ErrorCount         int64     `json:"error_count"`
	LastUsed           time.Time `json:"last_used,omitempty"`
}

// Match represents a single match found in content
type Match struct {
	PatternID   string                 `json:"pattern_id"`
	PatternName string                 `json:"pattern_name"`
	PatternType string                 `json:"pattern_type"`
	StartPos    int                    `json:"start_pos"`
	EndPos      int                    `json:"end_pos"`
	Text        string                 `json:"text"`
	Confidence  float64                `json:"confidence,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MatchStats contains statistics about pattern matching
type MatchStats struct {
	TotalMatches  int64     `json:"total_matches"`
	TotalPatterns int64     `json:"total_patterns"`
	AverageTimeMs float64   `json:"average_time_ms"`
	CacheHitRate  float64   `json:"cache_hit_rate"`
	ErrorRate     float64   `json:"error_rate"`
	LastUpdated   time.Time `json:"last_updated"`
}

// MatchOptions configures pattern matching behavior
type MatchOptions struct {
	CaseSensitive bool                   `json:"case_sensitive,omitempty"`
	WholeWord     bool                   `json:"whole_word,omitempty"`
	MaxMatches    int                    `json:"max_matches,omitempty"`
	Timeout       time.Duration          `json:"timeout,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// CompilationStats contains statistics about rule compilation
type CompilationStats struct {
	TotalRules    int64     `json:"total_rules"`
	CompiledRules int64     `json:"compiled_rules"`
	FailedRules   int64     `json:"failed_rules"`
	TotalTimeMs   int64     `json:"total_time_ms"`
	AverageTimeMs float64   `json:"average_time_ms"`
	LastCompiled  time.Time `json:"last_compiled"`
}

// RuleConflict represents a conflict between rules
type RuleConflict struct {
	Rule1ID  string `json:"rule1_id"`
	Rule2ID  string `json:"rule2_id"`
	Type     string `json:"type"` // "overlap", "contradiction", "dependency"
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

// PerformanceValidation represents performance validation results
type PerformanceValidation struct {
	Valid           bool     `json:"valid"`
	EstimatedTimeMs float64  `json:"estimated_time_ms"`
	ComplexityScore float64  `json:"complexity_score"`
	Warnings        []string `json:"warnings,omitempty"`
	Errors          []string `json:"errors,omitempty"`
}

// ValidationRule represents a validation rule
type ValidationRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	Severity    string `json:"severity"`
}

// PerformanceAnalysis represents analysis of rule performance
type PerformanceAnalysis struct {
	RuleID           string   `json:"rule_id"`
	AverageTimeMs    float64  `json:"average_time_ms"`
	MaxTimeMs        float64  `json:"max_time_ms"`
	MinTimeMs        float64  `json:"min_time_ms"`
	ComplexityScore  float64  `json:"complexity_score"`
	OptimizationTips []string `json:"optimization_tips,omitempty"`
}

// OptimizationSuggestion represents a suggestion for rule optimization
type OptimizationSuggestion struct {
	RuleID               string  `json:"rule_id"`
	Type                 string  `json:"type"` // "simplify", "cache", "reorder"
	Description          string  `json:"description"`
	Impact               string  `json:"impact"` // "low", "medium", "high"
	Effort               string  `json:"effort"` // "low", "medium", "high"
	EstimatedImprovement float64 `json:"estimated_improvement,omitempty"`
}

// RuleMetrics contains metrics for a specific rule
type RuleMetrics struct {
	RuleID         string    `json:"rule_id"`
	ExecutionCount int64     `json:"execution_count"`
	MatchCount     int64     `json:"match_count"`
	AverageTimeMs  float64   `json:"average_time_ms"`
	MaxTimeMs      float64   `json:"max_time_ms"`
	ErrorCount     int64     `json:"error_count"`
	CacheHitRate   float64   `json:"cache_hit_rate"`
	LastExecuted   time.Time `json:"last_executed,omitempty"`
}

// RulePackMetrics contains metrics for a rule pack
type RulePackMetrics struct {
	PackID         string    `json:"pack_id"`
	ExecutionCount int64     `json:"execution_count"`
	RuleCount      int64     `json:"rule_count"`
	AverageTimeMs  float64   `json:"average_time_ms"`
	MaxTimeMs      float64   `json:"max_time_ms"`
	ErrorCount     int64     `json:"error_count"`
	LastExecuted   time.Time `json:"last_executed,omitempty"`
}

// RulePerformanceReport contains a comprehensive performance report
type RulePerformanceReport struct {
	TimeRange        TimeRange          `json:"time_range"`
	TotalExecutions  int64              `json:"total_executions"`
	TotalMatches     int64              `json:"total_matches"`
	AverageTimeMs    float64            `json:"average_time_ms"`
	SlowestRules     []*SlowRule        `json:"slowest_rules,omitempty"`
	IneffectiveRules []*IneffectiveRule `json:"ineffective_rules,omitempty"`
	GeneratedAt      time.Time          `json:"generated_at"`
}

// SlowRule represents a rule with poor performance
type SlowRule struct {
	RuleID         string  `json:"rule_id"`
	RuleName       string  `json:"rule_name"`
	AverageTimeMs  float64 `json:"average_time_ms"`
	MaxTimeMs      float64 `json:"max_time_ms"`
	ExecutionCount int64   `json:"execution_count"`
}

// IneffectiveRule represents a rule with low match rates
type IneffectiveRule struct {
	RuleID         string  `json:"rule_id"`
	RuleName       string  `json:"rule_name"`
	ExecutionCount int64   `json:"execution_count"`
	MatchCount     int64   `json:"match_count"`
	MatchRate      float64 `json:"match_rate"`
}

// RuleTestCase represents a test case for a rule
type RuleTestCase struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Input       string                 `json:"input"`
	Expected    bool                   `json:"expected"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RuleTestResult represents the result of testing a rule
type RuleTestResult struct {
	RuleID      string            `json:"rule_id"`
	TotalTests  int               `json:"total_tests"`
	PassedTests int               `json:"passed_tests"`
	FailedTests int               `json:"failed_tests"`
	Accuracy    float64           `json:"accuracy"`
	TestCases   []*TestCaseResult `json:"test_cases,omitempty"`
	ExecutedAt  time.Time         `json:"executed_at"`
}

// TestCaseResult represents the result of a single test case
type TestCaseResult struct {
	TestCaseID string `json:"test_case_id"`
	Passed     bool   `json:"passed"`
	Expected   bool   `json:"expected"`
	Actual     bool   `json:"actual"`
	Error      string `json:"error,omitempty"`
}

// RulePackTestResult represents the result of testing a rule pack
type RulePackTestResult struct {
	PackID      string            `json:"pack_id"`
	TotalRules  int               `json:"total_rules"`
	TotalTests  int               `json:"total_tests"`
	PassedTests int               `json:"passed_tests"`
	FailedTests int               `json:"failed_tests"`
	Accuracy    float64           `json:"accuracy"`
	RuleResults []*RuleTestResult `json:"rule_results,omitempty"`
	ExecutedAt  time.Time         `json:"executed_at"`
}

// TestCaseValidation represents validation of test cases
type TestCaseValidation struct {
	Valid        bool     `json:"valid"`
	TotalCases   int      `json:"total_cases"`
	ValidCases   int      `json:"valid_cases"`
	InvalidCases int      `json:"invalid_cases"`
	Errors       []string `json:"errors,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

// BenchmarkResult represents benchmark results
type BenchmarkResult struct {
	RuleID        string    `json:"rule_id"`
	DatasetSize   int       `json:"dataset_size"`
	TotalTimeMs   int64     `json:"total_time_ms"`
	AverageTimeMs float64   `json:"average_time_ms"`
	Throughput    float64   `json:"throughput"` // items per second
	MemoryUsage   int64     `json:"memory_usage_bytes"`
	ExecutedAt    time.Time `json:"executed_at"`
}

// TestCoverage represents test coverage for rules
type TestCoverage struct {
	RuleID           string   `json:"rule_id"`
	TotalTestCases   int      `json:"total_test_cases"`
	CoveredTestCases int      `json:"covered_test_cases"`
	CoveragePercent  float64  `json:"coverage_percent"`
	UncoveredAreas   []string `json:"uncovered_areas,omitempty"`
}

// RulePackVersion represents a version of a rule pack
type RulePackVersion struct {
	PackID    string    `json:"pack_id"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by,omitempty"`
	Changes   []string  `json:"changes,omitempty"`
	Active    bool      `json:"active"`
}

// VersionComparison represents a comparison between two versions
type VersionComparison struct {
	PackID        string   `json:"pack_id"`
	Version1      int      `json:"version1"`
	Version2      int      `json:"version2"`
	AddedRules    []string `json:"added_rules,omitempty"`
	RemovedRules  []string `json:"removed_rules,omitempty"`
	ModifiedRules []string `json:"modified_rules,omitempty"`
	Changes       []string `json:"changes,omitempty"`
}

// VersionHistory represents version history for a rule pack
type VersionHistory struct {
	PackID    string    `json:"pack_id"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by,omitempty"`
	Summary   string    `json:"summary,omitempty"`
	Active    bool      `json:"active"`
}

// RuleEffectiveness represents the effectiveness of a rule
type RuleEffectiveness struct {
	RuleID         string    `json:"rule_id"`
	TimeRange      TimeRange `json:"time_range"`
	TruePositives  int64     `json:"true_positives"`
	FalsePositives int64     `json:"false_positives"`
	TrueNegatives  int64     `json:"true_negatives"`
	FalseNegatives int64     `json:"false_negatives"`
	Precision      float64   `json:"precision"`
	Recall         float64   `json:"recall"`
	F1Score        float64   `json:"f1_score"`
	Accuracy       float64   `json:"accuracy"`
}

// RulePackEffectiveness represents the effectiveness of a rule pack
type RulePackEffectiveness struct {
	PackID            string               `json:"pack_id"`
	TimeRange         TimeRange            `json:"time_range"`
	TotalRules        int                  `json:"total_rules"`
	EffectiveRules    int                  `json:"effective_rules"`
	AveragePrecision  float64              `json:"average_precision"`
	AverageRecall     float64              `json:"average_recall"`
	AverageF1Score    float64              `json:"average_f1_score"`
	RuleEffectiveness []*RuleEffectiveness `json:"rule_effectiveness,omitempty"`
}

// RuleCorrelation represents correlation between rules
type RuleCorrelation struct {
	Rule1ID     string  `json:"rule1_id"`
	Rule2ID     string  `json:"rule2_id"`
	Correlation float64 `json:"correlation"`
	Confidence  float64 `json:"confidence"`
	SampleSize  int64   `json:"sample_size"`
}

// RuleImpact represents the impact of a rule on system performance
type RuleImpact struct {
	RuleID           string  `json:"rule_id"`
	ExecutionTimeMs  float64 `json:"execution_time_ms"`
	MemoryUsageBytes int64   `json:"memory_usage_bytes"`
	CPUUsagePercent  float64 `json:"cpu_usage_percent"`
	ThroughputImpact float64 `json:"throughput_impact"`
	CostImpact       float64 `json:"cost_impact,omitempty"`
}

// RuleImprovement represents a suggested improvement for a rule
type RuleImprovement struct {
	RuleID           string  `json:"rule_id"`
	Type             string  `json:"type"` // "optimization", "accuracy", "coverage"
	Description      string  `json:"description"`
	Impact           string  `json:"impact"` // "low", "medium", "high"
	Effort           string  `json:"effort"` // "low", "medium", "high"
	Priority         int     `json:"priority"`
	EstimatedBenefit float64 `json:"estimated_benefit,omitempty"`
}
