package scanner

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/promptshield/promptshield/internal/rules"
)

// ContextMinimizer handles context minimization according to agent hardening patterns
type ContextMinimizer struct {
	config *rules.ContextMinimization
}

// NewContextMinimizer creates a new context minimizer with the given configuration
func NewContextMinimizer(config *rules.ContextMinimization) *ContextMinimizer {
	if config == nil || !config.Enabled {
		return nil
	}
	return &ContextMinimizer{config: config}
}

// MinimizeContext applies context minimization rules to the input text
func (cm *ContextMinimizer) MinimizeContext(content string, stripPoint string) (string, error) {
	if !cm.config.Enabled {
		return content, nil
	}

	// Use provided strip point or default from config
	point := stripPoint
	if point == "" {
		point = cm.config.StripPoint
	}
	if point == "" {
		point = "after_tool_selection" // default
	}

	maskToken := cm.config.MaskToken
	if maskToken == "" {
		maskToken = "<USER_TEXT>"
	}

	switch strings.ToLower(point) {
	case "after_tool_selection":
		return cm.minimizeAfterToolSelection(content, maskToken)
	case "before_execution":
		return cm.minimizeBeforeExecution(content, maskToken)
	case "step_by_step":
		return cm.minimizeStepByStep(content, maskToken)
	default:
		return cm.minimizeAfterToolSelection(content, maskToken)
	}
}

// minimizeAfterToolSelection masks user content after tool selection is made
func (cm *ContextMinimizer) minimizeAfterToolSelection(content string, maskToken string) (string, error) {
	// Parse JSON to identify user messages
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		// Not JSON, apply text-based minimization
		return cm.minimizeTextContent(content, maskToken), nil
	}

	// Handle OpenAI-style messages
	if messages, ok := data["messages"].([]interface{}); ok {
		for i, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				if role, ok := msgMap["role"].(string); ok && role == "user" {
					if content, ok := msgMap["content"].(string); ok {
						msgMap["content"] = cm.maskUserContent(content, maskToken)
						messages[i] = msgMap
					}
				}
			}
		}
		data["messages"] = messages
	}

	// Re-serialize
	result, err := json.Marshal(data)
	if err != nil {
		return content, err
	}
	return string(result), nil
}

// minimizeBeforeExecution masks content right before tool execution
func (cm *ContextMinimizer) minimizeBeforeExecution(content string, maskToken string) (string, error) {
	// More aggressive masking - only keep essential tool parameters
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return cm.minimizeTextContent(content, maskToken), nil
	}

	// Keep only essential fields for tool execution
	essential := map[string]interface{}{
		"model":       data["model"],
		"tools":       data["tools"],
		"tool_choice": data["tool_choice"],
	}

	// Mask messages but keep tool calls
	if messages, ok := data["messages"].([]interface{}); ok {
		var filteredMessages []interface{}
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				if role, ok := msgMap["role"].(string); ok {
					switch role {
					case "system", "assistant":
						// Keep system and assistant messages
						filteredMessages = append(filteredMessages, msgMap)
					case "user":
						// Mask user content
						maskedMsg := map[string]interface{}{
							"role":    "user",
							"content": maskToken,
						}
						filteredMessages = append(filteredMessages, maskedMsg)
					case "tool":
						// Keep tool results
						filteredMessages = append(filteredMessages, msgMap)
					}
				}
			}
		}
		essential["messages"] = filteredMessages
	}

	result, err := json.Marshal(essential)
	if err != nil {
		return content, err
	}
	return string(result), nil
}

// minimizeStepByStep applies step-based minimization
func (cm *ContextMinimizer) minimizeStepByStep(content string, maskToken string) (string, error) {
	// Apply minimization based on step count
	step := cm.config.Step
	if step <= 0 {
		step = 1
	}

	// More aggressive masking for later steps
	if step >= 3 {
		return cm.minimizeBeforeExecution(content, maskToken)
	} else if step >= 2 {
		return cm.minimizeAfterToolSelection(content, maskToken)
	}

	// First step - minimal masking
	return content, nil
}

// maskUserContent masks user content while preserving retained patterns
func (cm *ContextMinimizer) maskUserContent(content string, maskToken string) string {
	if len(cm.config.Retain) == 0 {
		return maskToken
	}

	// Find all retained patterns first
	var retainedMatches []string
	processedContent := content

	for _, pattern := range cm.config.Retain {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		matches := re.FindAllString(processedContent, -1)
		retainedMatches = append(retainedMatches, matches...)
	}

	// If we found retained patterns, return only those
	if len(retainedMatches) > 0 {
		return strings.Join(retainedMatches, " ")
	}

	// No patterns matched, return mask token
	return maskToken
}

// minimizeTextContent applies basic text minimization for non-JSON content
func (cm *ContextMinimizer) minimizeTextContent(content string, maskToken string) string {
	if len(cm.config.Retain) == 0 {
		return maskToken
	}

	result := content

	// Apply retention patterns
	for _, pattern := range cm.config.Retain {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		// Keep matches, mask everything else
		matches := re.FindAllString(result, -1)
		if len(matches) > 0 {
			return strings.Join(matches, " ")
		}
	}

	return maskToken
}

// IsEnabled returns whether context minimization is enabled
func (cm *ContextMinimizer) IsEnabled() bool {
	return cm != nil && cm.config != nil && cm.config.Enabled
}

// GetStripPoint returns the configured strip point
func (cm *ContextMinimizer) GetStripPoint() string {
	if cm == nil || cm.config == nil {
		return ""
	}
	return cm.config.StripPoint
}

// GetMaskToken returns the configured mask token
func (cm *ContextMinimizer) GetMaskToken() string {
	if cm == nil || cm.config == nil {
		return "<USER_TEXT>"
	}
	if cm.config.MaskToken == "" {
		return "<USER_TEXT>"
	}
	return cm.config.MaskToken
}
