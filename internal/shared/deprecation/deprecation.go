package deprecation

import (
	"fmt"
	"strings"
)

// LegacyOutputFormatWarning returns a standardized deprecation message for
// output formats that are considered experimental/unstable.
// It avoids mentioning project release status and focuses on feature stability.
func LegacyOutputFormatWarning(format string) (bool, string) {
	f := strings.ToLower(format)
	switch f {
	case "markdown", "csv", "html", "table":
		return true, fmt.Sprintf("[DEPRECATION] The '%s' output format has been removed. Use --output-format=json or --output-format=ndjson.", f)
	case "ndjson":
		return false, ""
	default:
		return false, ""
	}
}

// ExperimentalFeatureMessage builds a generic standardized deprecation message
// for unstable features, optionally suggesting an alternative and planned removal.
func ExperimentalFeatureMessage(feature, alternative, plannedRemoval string) string {
	if alternative != "" && plannedRemoval != "" {
		return fmt.Sprintf("[DEPRECATION] The '%s' feature is experimental and not stable. Planned removal: %s. Use %s.", feature, plannedRemoval, alternative)
	}
	if alternative != "" {
		return fmt.Sprintf("[DEPRECATION] The '%s' feature is experimental and not stable. Use %s.", feature, alternative)
	}
	return fmt.Sprintf("[DEPRECATION] The '%s' feature is experimental and not stable and may change or be removed without notice.", feature)
}
