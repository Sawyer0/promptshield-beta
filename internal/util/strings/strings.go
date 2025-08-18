package strings

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Truncate truncates a string to a maximum length
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// TruncateMiddle truncates a string in the middle, keeping start and end
func TruncateMiddle(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	
	startLen := (maxLen - 3) / 2
	endLen := maxLen - 3 - startLen
	return s[:startLen] + "..." + s[len(s)-endLen:]
}

// Contains checks if a string contains a substring (case-sensitive)
func Contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ContainsIgnoreCase checks if a string contains a substring (case-insensitive)
func ContainsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// ContainsAny checks if a string contains any of the substrings
func ContainsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// ContainsAll checks if a string contains all of the substrings
func ContainsAll(s string, substrs []string) bool {
	for _, substr := range substrs {
		if !strings.Contains(s, substr) {
			return false
		}
	}
	return true
}

// RemoveWhitespace removes all whitespace from a string
func RemoveWhitespace(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s, " ", ""), "\t", ""), "\n", ""), "\r", "")
}

// NormalizeWhitespace replaces multiple whitespace with single space
func NormalizeWhitespace(s string) string {
	re := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(re.ReplaceAllString(s, " "))
}

// IsBlank checks if a string is empty or contains only whitespace
func IsBlank(s string) bool {
	return strings.TrimSpace(s) == ""
}

// IsNotBlank checks if a string is not empty and contains non-whitespace
func IsNotBlank(s string) bool {
	return !IsBlank(s)
}

// DefaultIfBlank returns a default value if string is blank
func DefaultIfBlank(s, defaultValue string) string {
	if IsBlank(s) {
		return defaultValue
	}
	return s
}

// FirstNonBlank returns the first non-blank string from the list
func FirstNonBlank(strs ...string) string {
	for _, s := range strs {
		if IsNotBlank(s) {
			return s
		}
	}
	return ""
}

// Reverse reverses a string
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// IsPalindrome checks if a string is a palindrome
func IsPalindrome(s string) bool {
	s = strings.ToLower(s)
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		if runes[i] != runes[j] {
			return false
		}
	}
	return true
}

// RandomString generates a random string of specified length
func RandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// RandomAlphanumeric generates a random alphanumeric string
func RandomAlphanumeric(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	return string(bytes), nil
}

// Sanitize removes potentially dangerous characters from a string
func Sanitize(s string) string {
	// Remove null bytes
	s = strings.ReplaceAll(s, "\x00", "")
	// Remove control characters except tab, newline, carriage return
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// ToSnakeCase converts a string to snake_case
func ToSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

// ToCamelCase converts a string to camelCase
func ToCamelCase(s string) string {
	words := regexp.MustCompile(`[_\-\s]+`).Split(s, -1)
	for i, word := range words {
		if i == 0 {
			words[i] = strings.ToLower(word)
		} else {
			words[i] = strings.Title(strings.ToLower(word))
		}
	}
	return strings.Join(words, "")
}

// ToPascalCase converts a string to PascalCase
func ToPascalCase(s string) string {
	words := regexp.MustCompile(`[_\-\s]+`).Split(s, -1)
	for i, word := range words {
		words[i] = strings.Title(strings.ToLower(word))
	}
	return strings.Join(words, "")
}

// ToKebabCase converts a string to kebab-case
func ToKebabCase(s string) string {
	return strings.ReplaceAll(ToSnakeCase(s), "_", "-")
}

// Wrap wraps text to a specified width
func Wrap(text string, width int) string {
	if width <= 0 {
		return text
	}
	
	var result []string
	lines := strings.Split(text, "\n")
	
	for _, line := range lines {
		if len(line) <= width {
			result = append(result, line)
			continue
		}
		
		words := strings.Fields(line)
		currentLine := ""
		
		for _, word := range words {
			if len(currentLine)+len(word)+1 > width {
				if currentLine != "" {
					result = append(result, currentLine)
					currentLine = word
				} else {
					// Word is longer than width, split it
					for len(word) > width {
						result = append(result, word[:width])
						word = word[width:]
					}
					currentLine = word
				}
			} else {
				if currentLine == "" {
					currentLine = word
				} else {
					currentLine += " " + word
				}
			}
		}
		if currentLine != "" {
			result = append(result, currentLine)
		}
	}
	
	return strings.Join(result, "\n")
}

// Indent indents each line of text
func Indent(text string, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

// Dedent removes common leading whitespace from each line
func Dedent(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return text
	}
	
	// Find minimum indent
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	
	if minIndent <= 0 {
		return text
	}
	
	// Remove common indent
	for i, line := range lines {
		if len(line) >= minIndent {
			lines[i] = line[minIndent:]
		}
	}
	
	return strings.Join(lines, "\n")
}

// SplitLines splits text into lines, handling different line endings
func SplitLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.Split(text, "\n")
}

// JoinLines joins lines with the system line separator
func JoinLines(lines []string) string {
	return strings.Join(lines, "\n")
}

// CountOccurrences counts occurrences of a substring
func CountOccurrences(s, substr string) int {
	return strings.Count(s, substr)
}

// ReplaceMultiple replaces multiple strings in one pass
func ReplaceMultiple(s string, replacements map[string]string) string {
	for old, new := range replacements {
		s = strings.ReplaceAll(s, old, new)
	}
	return s
}

// EscapeHTML escapes HTML special characters
func EscapeHTML(s string) string {
	replacements := map[string]string{
		"&":  "&amp;",
		"<":  "&lt;",
		">":  "&gt;",
		"\"": "&quot;",
		"'":  "&#39;",
	}
	return ReplaceMultiple(s, replacements)
}

// UnescapeHTML unescapes HTML special characters
func UnescapeHTML(s string) string {
	replacements := map[string]string{
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": "\"",
		"&#39;":  "'",
	}
	return ReplaceMultiple(s, replacements)
}

// Mask masks sensitive parts of a string
func Mask(s string, showFirst, showLast int) string {
	if len(s) <= showFirst+showLast {
		return strings.Repeat("*", len(s))
	}
	
	masked := s[:showFirst]
	masked += strings.Repeat("*", len(s)-showFirst-showLast)
	masked += s[len(s)-showLast:]
	return masked
}

// MaskEmail masks an email address
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return Mask(email, 2, 2)
	}
	
	local := parts[0]
	domain := parts[1]
	
	if len(local) <= 3 {
		local = strings.Repeat("*", len(local))
	} else {
		local = local[:1] + strings.Repeat("*", len(local)-2) + local[len(local)-1:]
	}
	
	return fmt.Sprintf("%s@%s", local, domain)
}

// IsEmail checks if a string is a valid email format
func IsEmail(s string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(s)
}

// IsURL checks if a string is a valid URL format
func IsURL(s string) bool {
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return urlRegex.MatchString(s)
}

// IsAlphanumeric checks if a string contains only letters and numbers
func IsAlphanumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// IsNumeric checks if a string contains only digits
func IsNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(s) > 0
}

// IsAlpha checks if a string contains only letters
func IsAlpha(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return len(s) > 0
}

// Coalesce returns the first non-empty string (alias for DefaultIfBlank)
func Coalesce(s, def string) string {
	return DefaultIfBlank(s, def)
}