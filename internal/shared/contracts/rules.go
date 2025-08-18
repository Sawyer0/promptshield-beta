package contracts

import (
	"context"
	"time"

	"github.com/promptshield/promptshield/internal/shared/types"
)

// RuleEngine defines the interface for rule processing and evaluation
type RuleEngine interface {
	// EvaluateRules evaluates rules against content
	EvaluateRules(ctx context.Context, content []byte, rules []*types.Rule) ([]*types.RuleMatch, error)
	
	// EvaluateRulePack evaluates a complete rule pack against content
	EvaluateRulePack(ctx context.Context, content []byte, rulePack *types.RulePack) ([]*types.RuleMatch, error)
	
	// EvaluateLevel evaluates rules of a specific level
	EvaluateLevel(ctx context.Context, content []byte, level types.RuleLevel) ([]*types.RuleMatch, error)
	
	// CompileRules compiles rules for efficient evaluation
	CompileRules(ctx context.Context, rules []*types.Rule) (*types.CompiledRuleSet, error)
	
	// EvaluateCompiledRules evaluates pre-compiled rules
	EvaluateCompiledRules(ctx context.Context, content []byte, compiled *types.CompiledRuleSet) ([]*types.RuleMatch, error)
	
	// GetSupportedLevels returns supported rule levels
	GetSupportedLevels() []types.RuleLevel
	
	// ValidateRule validates a single rule
	ValidateRule(ctx context.Context, rule *types.Rule) (*types.ValidationResult, error)
}

// RulePackManager defines the interface for managing rule packs
type RulePackManager interface {
	// LoadRulePack loads a rule pack from data
	LoadRulePack(ctx context.Context, data []byte) (*types.RulePack, error)
	
	// SaveRulePack saves a rule pack to storage
	SaveRulePack(ctx context.Context, rulePack *types.RulePack) error
	
	// GetRulePack retrieves a rule pack by ID
	GetRulePack(ctx context.Context, packID string) (*types.RulePack, error)
	
	// ListRulePacks lists all available rule packs
	ListRulePacks(ctx context.Context) ([]*types.RulePackInfo, error)
	
	// UpdateRulePack updates an existing rule pack
	UpdateRulePack(ctx context.Context, packID string, rulePack *types.RulePack) error
	
	// DeleteRulePack deletes a rule pack
	DeleteRulePack(ctx context.Context, packID string) error
	
	// ValidateRulePack validates a complete rule pack
	ValidateRulePack(ctx context.Context, rulePack *types.RulePack) (*types.ValidationResult, error)
	
	// MergeRulePacks merges multiple rule packs
	MergeRulePacks(ctx context.Context, packIDs []string) (*types.RulePack, error)
}

// RuleCompiler defines the interface for compiling rules
type RuleCompiler interface {
	// CompilePattern compiles a pattern for efficient matching
	CompilePattern(ctx context.Context, pattern *types.Pattern) (*types.CompiledPattern, error)
	
	// CompileRegex compiles a regex pattern
	CompileRegex(ctx context.Context, regex string) (*types.CompiledRegex, error)
	
	// CompileKeywords compiles keyword patterns
	CompileKeywords(ctx context.Context, keywords []string) (*types.CompiledKeywords, error)
	
	// CompileSemantic compiles semantic analysis rules
	CompileSemantic(ctx context.Context, semantic *types.Semantic) (*types.CompiledSemantic, error)
	
	// OptimizeRules optimizes compiled rules for performance
	OptimizeRules(ctx context.Context, rules []*types.CompiledPattern) ([]*types.CompiledPattern, error)
	
	// GetCompilationStats returns compilation statistics
	GetCompilationStats() *types.CompilationStats
	
	// ValidatePattern validates pattern syntax
	ValidatePattern(ctx context.Context, pattern *types.Pattern) error
}

// RuleMatcher defines the interface for pattern matching
type RuleMatcher interface {
	// MatchKeywords matches keyword patterns in content
	MatchKeywords(ctx context.Context, content []byte, keywords *types.CompiledKeywords) ([]*types.Match, error)
	
	// MatchRegex matches regex patterns in content
	MatchRegex(ctx context.Context, content []byte, regex *types.CompiledRegex) ([]*types.Match, error)
	
	// MatchSemantic performs semantic analysis on content
	MatchSemantic(ctx context.Context, content []byte, semantic *types.CompiledSemantic) ([]*types.Match, error)
	
	// MatchStream performs streaming pattern matching
	MatchStream(ctx context.Context, content <-chan []byte, patterns []*types.CompiledPattern) (<-chan *types.Match, error)
	
	// GetMatchStats returns matching statistics
	GetMatchStats() *types.MatchStats
	
	// SetMatchOptions sets matching options
	SetMatchOptions(options *types.MatchOptions) error
}

// RuleValidator defines the interface for rule validation
type RuleValidator interface {
	// ValidateRulePack validates a complete rule pack
	ValidateRulePack(ctx context.Context, rulePack *types.RulePack) (*types.ValidationResult, error)
	
	// ValidateRule validates a single rule
	ValidateRule(ctx context.Context, rule *types.Rule) (*types.ValidationResult, error)
	
	// ValidatePattern validates a pattern
	ValidatePattern(ctx context.Context, pattern *types.Pattern) (*types.ValidationResult, error)
	
	// ValidateSemantic validates semantic configuration
	ValidateSemantic(ctx context.Context, semantic *types.Semantic) (*types.ValidationResult, error)
	
	// CheckRuleConflicts checks for conflicts between rules
	CheckRuleConflicts(ctx context.Context, rules []*types.Rule) ([]*types.RuleConflict, error)
	
	// ValidateRulePerformance validates rule performance characteristics
	ValidateRulePerformance(ctx context.Context, rule *types.Rule) (*types.PerformanceValidation, error)
	
	// GetValidationRules returns current validation rules
	GetValidationRules() []*types.ValidationRule
}

// RuleOptimizer defines the interface for rule optimization
type RuleOptimizer interface {
	// OptimizeRulePack optimizes a rule pack for performance
	OptimizeRulePack(ctx context.Context, rulePack *types.RulePack) (*types.RulePack, error)
	
	// OptimizeRuleOrder optimizes rule execution order
	OptimizeRuleOrder(ctx context.Context, rules []*types.Rule) ([]*types.Rule, error)
	
	// DeduplicateRules removes duplicate rules
	DeduplicateRules(ctx context.Context, rules []*types.Rule) ([]*types.Rule, error)
	
	// SimplifyPatterns simplifies complex patterns
	SimplifyPatterns(ctx context.Context, patterns []*types.Pattern) ([]*types.Pattern, error)
	
	// AnalyzePerformance analyzes rule performance
	AnalyzePerformance(ctx context.Context, rules []*types.Rule) (*types.PerformanceAnalysis, error)
	
	// SuggestOptimizations suggests optimization opportunities
	SuggestOptimizations(ctx context.Context, rulePack *types.RulePack) ([]*types.OptimizationSuggestion, error)
}

// RuleCache defines the interface for caching compiled rules
type RuleCache interface {
	// GetCompiledRule retrieves a compiled rule from cache
	GetCompiledRule(ctx context.Context, ruleID string) (*types.CompiledPattern, error)
	
	// StoreCompiledRule stores a compiled rule in cache
	StoreCompiledRule(ctx context.Context, ruleID string, compiled *types.CompiledPattern) error
	
	// GetCompiledRulePack retrieves a compiled rule pack from cache
	GetCompiledRulePack(ctx context.Context, packID string) (*types.CompiledRuleSet, error)
	
	// StoreCompiledRulePack stores a compiled rule pack in cache
	StoreCompiledRulePack(ctx context.Context, packID string, compiled *types.CompiledRuleSet) error
	
	// InvalidateRule invalidates a cached rule
	InvalidateRule(ctx context.Context, ruleID string) error
	
	// InvalidateRulePack invalidates a cached rule pack
	InvalidateRulePack(ctx context.Context, packID string) error
	
	// ClearCache clears all cached rules
	ClearCache(ctx context.Context) error
	
	// GetCacheStats returns cache statistics
	GetCacheStats(ctx context.Context) (*types.CacheStats, error)
}

// RuleMonitor defines the interface for monitoring rule performance
type RuleMonitor interface {
	// RecordRuleExecution records rule execution metrics
	RecordRuleExecution(ctx context.Context, ruleID string, duration time.Duration, matches int) error
	
	// RecordRulePackExecution records rule pack execution metrics
	RecordRulePackExecution(ctx context.Context, packID string, duration time.Duration, matches int) error
	
	// GetRuleMetrics returns metrics for a specific rule
	GetRuleMetrics(ctx context.Context, ruleID string, timeRange types.TimeRange) (*types.RuleMetrics, error)
	
	// GetRulePackMetrics returns metrics for a rule pack
	GetRulePackMetrics(ctx context.Context, packID string, timeRange types.TimeRange) (*types.RulePackMetrics, error)
	
	// GetPerformanceReport returns rule performance report
	GetPerformanceReport(ctx context.Context, timeRange types.TimeRange) (*types.RulePerformanceReport, error)
	
	// GetSlowRules returns rules with poor performance
	GetSlowRules(ctx context.Context, threshold time.Duration) ([]*types.SlowRule, error)
	
	// GetIneffectiveRules returns rules with low match rates
	GetIneffectiveRules(ctx context.Context, threshold float64) ([]*types.IneffectiveRule, error)
}

// RuleTestingFramework defines the interface for testing rules
type RuleTestingFramework interface {
	// TestRule tests a rule against test cases
	TestRule(ctx context.Context, rule *types.Rule, testCases []*types.RuleTestCase) (*types.RuleTestResult, error)
	
	// TestRulePack tests a rule pack against test cases
	TestRulePack(ctx context.Context, rulePack *types.RulePack, testCases []*types.RuleTestCase) (*types.RulePackTestResult, error)
	
	// GenerateTestCases generates test cases for a rule
	GenerateTestCases(ctx context.Context, rule *types.Rule) ([]*types.RuleTestCase, error)
	
	// ValidateTestCases validates test case quality
	ValidateTestCases(ctx context.Context, testCases []*types.RuleTestCase) (*types.TestCaseValidation, error)
	
	// RunBenchmark runs performance benchmarks for rules
	RunBenchmark(ctx context.Context, rules []*types.Rule, dataset [][]byte) (*types.BenchmarkResult, error)
	
	// GetTestCoverage returns test coverage for rules
	GetTestCoverage(ctx context.Context, rules []*types.Rule, testCases []*types.RuleTestCase) (*types.TestCoverage, error)
}

// RuleVersionManager defines the interface for managing rule versions
type RuleVersionManager interface {
	// CreateVersion creates a new version of a rule pack
	CreateVersion(ctx context.Context, packID string, rulePack *types.RulePack) (*types.RulePackVersion, error)
	
	// GetVersion retrieves a specific version of a rule pack
	GetVersion(ctx context.Context, packID string, version int) (*types.RulePack, error)
	
	// ListVersions lists all versions of a rule pack
	ListVersions(ctx context.Context, packID string) ([]*types.RulePackVersion, error)
	
	// CompareVersions compares two versions of a rule pack
	CompareVersions(ctx context.Context, packID string, version1, version2 int) (*types.VersionComparison, error)
	
	// RollbackVersion rolls back to a previous version
	RollbackVersion(ctx context.Context, packID string, version int) error
	
	// DeleteVersion deletes a specific version
	DeleteVersion(ctx context.Context, packID string, version int) error
	
	// GetVersionHistory returns version history
	GetVersionHistory(ctx context.Context, packID string) ([]*types.VersionHistory, error)
}

// RuleAnalyzer defines the interface for analyzing rule effectiveness
type RuleAnalyzer interface {
	// AnalyzeRuleEffectiveness analyzes rule effectiveness
	AnalyzeRuleEffectiveness(ctx context.Context, ruleID string, timeRange types.TimeRange) (*types.RuleEffectiveness, error)
	
	// AnalyzeRulePackEffectiveness analyzes rule pack effectiveness
	AnalyzeRulePackEffectiveness(ctx context.Context, packID string, timeRange types.TimeRange) (*types.RulePackEffectiveness, error)
	
	// GetFalsePositiveRate calculates false positive rate for a rule
	GetFalsePositiveRate(ctx context.Context, ruleID string, timeRange types.TimeRange) (float64, error)
	
	// GetFalseNegativeRate calculates false negative rate for a rule
	GetFalseNegativeRate(ctx context.Context, ruleID string, timeRange types.TimeRange) (float64, error)
	
	// AnalyzeRuleCorrelation analyzes correlation between rules
	AnalyzeRuleCorrelation(ctx context.Context, ruleIDs []string, timeRange types.TimeRange) (*types.RuleCorrelation, error)
	
	// GetRuleImpact calculates rule impact on system performance
	GetRuleImpact(ctx context.Context, ruleID string, timeRange types.TimeRange) (*types.RuleImpact, error)
	
	// SuggestRuleImprovements suggests improvements for rules
	SuggestRuleImprovements(ctx context.Context, ruleID string) ([]*types.RuleImprovement, error)
}