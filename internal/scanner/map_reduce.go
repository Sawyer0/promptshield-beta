package scanner

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/promptshield/promptshield/internal/rules"
	"github.com/promptshield/promptshield/pkg/types"
)

// MapReduceProcessor handles map-reduce operations for large document processing
type MapReduceProcessor struct {
	config *rules.MapReduce
}

// NewMapReduceProcessor creates a new map-reduce processor with the given configuration
func NewMapReduceProcessor(config *rules.MapReduce) *MapReduceProcessor {
	if config == nil || !config.Enabled {
		return nil
	}
	return &MapReduceProcessor{config: config}
}

// ProcessDocument applies map-reduce processing to large documents
func (mrp *MapReduceProcessor) ProcessDocument(ctx context.Context, content string, scanner *Scanner) (types.ScanResult, error) {
	if !mrp.config.Enabled {
		// Bypass map-reduce entirely; process directly without any agent minimization mutations
		return scanner.scanContentDirect(ctx, content, "direct")
	}

	// Determine chunking strategy based on map unit
	chunks, err := mrp.chunkDocument(content)
	if err != nil {
		return types.ScanResult{}, fmt.Errorf("failed to chunk document: %w", err)
	}

	if len(chunks) <= 1 {
		// Small document, process directly without any agent minimization mutations
		res, err := scanner.scanContentDirect(ctx, content, "direct")
		if err != nil {
			return types.ScanResult{}, err
		}
		// Wrap into a reduce-style result with scan info populated
		combined := types.ScanResult{
			Violations: res.Violations,
			Metrics:    res.Metrics,
			ScanInfo: types.ScanInfo{
				TotalViolations: len(res.Violations),
				ScanStatus:      "success",
			},
		}
		return combined, nil
	}

	// Map phase: process each chunk
	mapResults := make([]types.ScanResult, len(chunks))
	for i, chunk := range chunks {
		// Process each chunk directly for security signal scanning (no agent minimization mutations)
		result, err := scanner.scanContentDirect(ctx, chunk, fmt.Sprintf("chunk-%d", i))
		if err != nil {
			return types.ScanResult{}, fmt.Errorf("failed to process chunk %d: %w", i, err)
		}
		mapResults[i] = result
	}

	// Reduce phase: combine results
	return mrp.reduceResults(mapResults)
}

// chunkDocument splits the document into processable chunks
func (mrp *MapReduceProcessor) chunkDocument(content string) ([]string, error) {
	mapUnit := strings.ToLower(mrp.config.MapUnit)
	if mapUnit == "" {
		mapUnit = "paragraph"
	}

	maxTokens := mrp.config.TextMaxTokens
	if maxTokens <= 0 {
		maxTokens = 2000 // default chunk size
	}

	switch mapUnit {
	case "paragraph":
		return mrp.chunkByParagraph(content, maxTokens), nil
	case "sentence":
		return mrp.chunkBySentence(content, maxTokens), nil
	case "line":
		return mrp.chunkByLine(content, maxTokens), nil
	case "token":
		return mrp.chunkByToken(content, maxTokens), nil
	case "semantic":
		return mrp.chunkBySemantic(content, maxTokens), nil
	default:
		return mrp.chunkByParagraph(content, maxTokens), nil
	}
}

// chunkByParagraph splits content by paragraphs, respecting token limits
func (mrp *MapReduceProcessor) chunkByParagraph(content string, maxTokens int) []string {
	paragraphs := strings.Split(content, "\n\n")
	var chunks []string
	var currentChunk strings.Builder
	currentTokens := 0

	for _, para := range paragraphs {
		paraTokens := mrp.estimateTokens(para)

		if currentTokens+paraTokens > maxTokens && currentChunk.Len() > 0 {
			// Start new chunk
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentTokens = 0
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(para)
		currentTokens += paraTokens
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// chunkBySentence splits content by sentences, respecting token limits
func (mrp *MapReduceProcessor) chunkBySentence(content string, maxTokens int) []string {
	// Simple sentence splitting regex
	sentenceRe := regexp.MustCompile(`[.!?]+\s+`)
	sentences := sentenceRe.Split(content, -1)

	var chunks []string
	var currentChunk strings.Builder
	currentTokens := 0

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		sentTokens := mrp.estimateTokens(sentence)

		if currentTokens+sentTokens > maxTokens && currentChunk.Len() > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentTokens = 0
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(sentence)
		currentTokens += sentTokens
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// chunkByLine splits content by lines, respecting token limits
func (mrp *MapReduceProcessor) chunkByLine(content string, maxTokens int) []string {
	lines := strings.Split(content, "\n")

	var chunks []string
	var currentChunk strings.Builder
	currentTokens := 0

	for _, line := range lines {
		lineTokens := mrp.estimateTokens(line)

		if currentTokens+lineTokens > maxTokens && currentChunk.Len() > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentTokens = 0
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n")
		}
		currentChunk.WriteString(line)
		currentTokens += lineTokens
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// chunkByToken splits content by approximate token count
func (mrp *MapReduceProcessor) chunkByToken(content string, maxTokens int) []string {
	words := strings.Fields(content)
	var chunks []string
	var currentChunk strings.Builder
	currentTokens := 0

	for _, word := range words {
		wordTokens := mrp.estimateTokens(word)

		if currentTokens+wordTokens > maxTokens && currentChunk.Len() > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentTokens = 0
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(word)
		currentTokens += wordTokens
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// chunkBySemantic attempts semantic chunking (simplified implementation)
func (mrp *MapReduceProcessor) chunkBySemantic(content string, maxTokens int) []string {
	// For now, fall back to paragraph chunking with semantic boundaries
	// In a full implementation, this would use embeddings or NLP to find semantic boundaries
	paragraphs := mrp.chunkByParagraph(content, maxTokens)

	// Try to merge small paragraphs that are semantically related
	var chunks []string
	var currentChunk strings.Builder
	currentTokens := 0

	for _, para := range paragraphs {
		paraTokens := mrp.estimateTokens(para)

		// Simple heuristic: merge if under half the max tokens and contains connecting words
		if currentTokens+paraTokens < maxTokens/2 && mrp.hasSemanticConnection(currentChunk.String(), para) {
			if currentChunk.Len() > 0 {
				currentChunk.WriteString("\n\n")
			}
			currentChunk.WriteString(para)
			currentTokens += paraTokens
		} else {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, currentChunk.String())
			}
			currentChunk.Reset()
			currentChunk.WriteString(para)
			currentTokens = paraTokens
		}
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// hasSemanticConnection checks for basic semantic connections between text chunks
func (mrp *MapReduceProcessor) hasSemanticConnection(chunk1, chunk2 string) bool {
	// Simple heuristics for semantic connection
	connectingWords := []string{
		"however", "furthermore", "moreover", "additionally", "therefore",
		"consequently", "meanwhile", "similarly", "in contrast", "for example",
		"specifically", "in particular", "as a result", "on the other hand",
	}

	chunk2Lower := strings.ToLower(chunk2)
	for _, word := range connectingWords {
		if strings.Contains(chunk2Lower, word) {
			return true
		}
	}

	// Check for pronoun references
	pronouns := []string{"this", "that", "these", "those", "it", "they", "them"}
	words := strings.Fields(chunk2Lower)
	if len(words) > 0 {
		firstWord := words[0]
		for _, pronoun := range pronouns {
			if firstWord == pronoun {
				return true
			}
		}
	}

	return false
}

// estimateTokens provides a rough token count estimate
func (mrp *MapReduceProcessor) estimateTokens(text string) int {
	// Rough estimate: 1 token ≈ 4 characters for English text
	// This is a simplification; real tokenization would be more accurate
	return len(text) / 4
}

// reduceResults combines map results according to the reduce strategy
func (mrp *MapReduceProcessor) reduceResults(results []types.ScanResult) (types.ScanResult, error) {
	if len(results) == 0 {
		return types.ScanResult{}, nil
	}

	if len(results) == 1 {
		return results[0], nil
	}

	reduceType := strings.ToLower(mrp.config.ReduceType)
	if reduceType == "" {
		reduceType = "union"
	}

	switch reduceType {
	case "union":
		return mrp.reduceUnion(results), nil
	case "intersection":
		return mrp.reduceIntersection(results), nil
	case "highest_severity":
		return mrp.reduceHighestSeverity(results), nil
	case "count_based":
		return mrp.reduceCountBased(results), nil
	case "consensus":
		return mrp.reduceConsensus(results), nil
	default:
		return mrp.reduceUnion(results), nil
	}
}

// reduceUnion combines all violations from all chunks
func (mrp *MapReduceProcessor) reduceUnion(results []types.ScanResult) types.ScanResult {
	combined := types.ScanResult{
		Violations: make([]types.Violation, 0),
		ScanInfo: types.ScanInfo{
			ScanStatus: "success",
		},
	}

	violationMap := make(map[string]types.Violation) // deduplication

	for i, result := range results {
		for _, violation := range result.Violations {
			// Create unique key for deduplication
			key := fmt.Sprintf("%s:%s:%s", violation.RuleID, violation.Category, violation.Severity)
			if existing, exists := violationMap[key]; exists {
				// Keep existing for now - could merge messages if needed
				violationMap[key] = existing
			} else {
				// Add chunk info to message
				violation.Message = fmt.Sprintf("chunk-%d: %s", i, violation.Message)
				violationMap[key] = violation
			}
		}
		// Accumulate metrics
		combined.Metrics.BytesRead += result.Metrics.BytesRead
		combined.Metrics.LinesRead += result.Metrics.LinesRead
	}

	// Convert map to slice
	for _, violation := range violationMap {
		combined.Violations = append(combined.Violations, violation)
	}

	combined.ScanInfo.TotalViolations = len(combined.Violations)
	combined.ScanInfo.ScanStatus = "success"

	return combined
}

// reduceIntersection keeps only violations found in multiple chunks
func (mrp *MapReduceProcessor) reduceIntersection(results []types.ScanResult) types.ScanResult {
	combined := types.ScanResult{
		Violations: make([]types.Violation, 0),
		ScanInfo: types.ScanInfo{
			ScanStatus: "success",
		},
	}

	// Count occurrences of each violation type
	violationCounts := make(map[string]int)
	violationExamples := make(map[string]types.Violation)

	for _, result := range results {
		seen := make(map[string]bool)
		for _, violation := range result.Violations {
			key := fmt.Sprintf("%s:%s", violation.RuleID, violation.Category)
			if !seen[key] {
				violationCounts[key]++
				violationExamples[key] = violation
				seen[key] = true
			}
		}
		// Accumulate metrics
		combined.Metrics.BytesRead += result.Metrics.BytesRead
		combined.Metrics.LinesRead += result.Metrics.LinesRead
	}

	// Keep violations found in at least 2 chunks
	threshold := 2
	if len(results) < 2 {
		threshold = 1
	}

	for key, count := range violationCounts {
		if count >= threshold {
			violation := violationExamples[key]
			violation.Message = fmt.Sprintf("Found in %d chunks: %s", count, violation.Message)
			combined.Violations = append(combined.Violations, violation)
		}
	}

	combined.ScanInfo.TotalViolations = len(combined.Violations)

	return combined
}

// reduceHighestSeverity keeps only the highest severity violations
func (mrp *MapReduceProcessor) reduceHighestSeverity(results []types.ScanResult) types.ScanResult {
	combined := types.ScanResult{
		Violations: make([]types.Violation, 0),
		ScanInfo: types.ScanInfo{
			ScanStatus: "success",
		},
	}

	severityOrder := map[string]int{
		"CRITICAL": 4,
		"HIGH":     3,
		"MEDIUM":   2,
		"LOW":      1,
		"INFO":     0,
	}

	maxSeverity := 0
	for _, result := range results {
		for _, violation := range result.Violations {
			if sev, ok := severityOrder[violation.Severity]; ok && sev > maxSeverity {
				maxSeverity = sev
			}
		}
		// Accumulate metrics
		combined.Metrics.BytesRead += result.Metrics.BytesRead
		combined.Metrics.LinesRead += result.Metrics.LinesRead
	}

	// Collect all violations at the highest severity level
	for _, result := range results {
		for _, violation := range result.Violations {
			if sev, ok := severityOrder[violation.Severity]; ok && sev == maxSeverity {
				combined.Violations = append(combined.Violations, violation)
			}
		}
	}

	combined.ScanInfo.TotalViolations = len(combined.Violations)

	return combined
}

// reduceCountBased applies count-based filtering
func (mrp *MapReduceProcessor) reduceCountBased(results []types.ScanResult) types.ScanResult {
	combined := types.ScanResult{
		Violations: make([]types.Violation, 0),
		ScanInfo: types.ScanInfo{
			ScanStatus: "success",
		},
	}

	// Count total violations per rule
	ruleCounts := make(map[string]int)
	ruleExamples := make(map[string]types.Violation)

	for _, result := range results {
		for _, violation := range result.Violations {
			ruleCounts[violation.RuleID]++
			if _, exists := ruleExamples[violation.RuleID]; !exists {
				ruleExamples[violation.RuleID] = violation
			}
		}
		// Accumulate metrics
		combined.Metrics.BytesRead += result.Metrics.BytesRead
		combined.Metrics.LinesRead += result.Metrics.LinesRead
	}

	// Include rules that triggered multiple times (indicating pattern)
	threshold := len(results) / 2 // At least half the chunks
	if threshold < 1 {
		threshold = 1
	}

	for rule, count := range ruleCounts {
		if count >= threshold {
			violation := ruleExamples[rule]
			violation.Message = fmt.Sprintf("Triggered %d times across chunks: %s", count, violation.Message)
			combined.Violations = append(combined.Violations, violation)
		}
	}

	combined.ScanInfo.TotalViolations = len(combined.Violations)

	return combined
}

// reduceConsensus applies consensus-based reduction
func (mrp *MapReduceProcessor) reduceConsensus(results []types.ScanResult) types.ScanResult {
	combined := types.ScanResult{
		Violations: make([]types.Violation, 0),
		ScanInfo: types.ScanInfo{
			ScanStatus: "success",
		},
	}

	// Require majority consensus for violations
	consensusThreshold := (len(results) + 1) / 2 // Majority

	violationCounts := make(map[string]int)
	violationExamples := make(map[string]types.Violation)

	for _, result := range results {
		seen := make(map[string]bool)
		for _, violation := range result.Violations {
			key := fmt.Sprintf("%s:%s", violation.RuleID, violation.Category)
			if !seen[key] {
				violationCounts[key]++
				violationExamples[key] = violation
				seen[key] = true
			}
		}
		// Accumulate metrics
		combined.Metrics.BytesRead += result.Metrics.BytesRead
		combined.Metrics.LinesRead += result.Metrics.LinesRead
	}

	for key, count := range violationCounts {
		if count >= consensusThreshold {
			violation := violationExamples[key]
			violation.Message = fmt.Sprintf("Consensus: %d/%d chunks: %s", count, len(results), violation.Message)
			combined.Violations = append(combined.Violations, violation)
		}
	}

	combined.ScanInfo.TotalViolations = len(combined.Violations)

	return combined
}

// IsEnabled returns whether map-reduce processing is enabled
func (mrp *MapReduceProcessor) IsEnabled() bool {
	return mrp != nil && mrp.config != nil && mrp.config.Enabled
}

// GetMapUnit returns the configured map unit
func (mrp *MapReduceProcessor) GetMapUnit() string {
	if mrp == nil || mrp.config == nil {
		return "paragraph"
	}
	if mrp.config.MapUnit == "" {
		return "paragraph"
	}
	return mrp.config.MapUnit
}

// GetReduceType returns the configured reduce type
func (mrp *MapReduceProcessor) GetReduceType() string {
	if mrp == nil || mrp.config == nil {
		return "union"
	}
	if mrp.config.ReduceType == "" {
		return "union"
	}
	return mrp.config.ReduceType
}
