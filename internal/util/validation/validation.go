package validation

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/promptshield/promptshield/internal/shared/errors"
)

// Validator represents a validation rule
type Validator func(value interface{}) error

// ValidationResult represents the result of validation
type ValidationResult struct {
	Valid  bool
	Errors []error
	Fields map[string][]string
}

// NewValidationResult creates a new validation result
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		Valid:  true,
		Fields: make(map[string][]string),
	}
}

// AddError adds an error to the validation result
func (v *ValidationResult) AddError(field string, err error) {
	v.Valid = false
	v.Errors = append(v.Errors, err)
	if field != "" {
		v.Fields[field] = append(v.Fields[field], err.Error())
	}
}

// AddFieldError adds a field-specific error
func (v *ValidationResult) AddFieldError(field, message string) {
	v.Valid = false
	err := fmt.Errorf("%s: %s", field, message)
	v.Errors = append(v.Errors, err)
	v.Fields[field] = append(v.Fields[field], message)
}

// GetFieldErrors returns errors for a specific field
func (v *ValidationResult) GetFieldErrors(field string) []string {
	return v.Fields[field]
}

// Error returns a combined error message
func (v *ValidationResult) Error() string {
	if v.Valid {
		return ""
	}
	var messages []string
	for _, err := range v.Errors {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// Required validates that a value is not nil or empty
func Required(fieldName string, value interface{}) error {
	if value == nil {
		return errors.MissingRequiredField(fieldName)
	}
	
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return errors.MissingRequiredField(fieldName)
		}
	case []byte:
		if len(v) == 0 {
			return errors.MissingRequiredField(fieldName)
		}
	case int, int8, int16, int32, int64:
		// Numbers are always considered present
	case uint, uint8, uint16, uint32, uint64:
		// Numbers are always considered present
	case float32, float64:
		// Numbers are always considered present
	case bool:
		// Booleans are always considered present
	default:
		// For other types, check if it's a zero value
		if fmt.Sprintf("%v", v) == "<nil>" {
			return errors.MissingRequiredField(fieldName)
		}
	}
	
	return nil
}

// MinLength validates minimum string length
func MinLength(fieldName string, value string, min int) error {
	if len(value) < min {
		return errors.InvalidFieldValue(fieldName, value, fmt.Sprintf("minimum length %d", min))
	}
	return nil
}

// MaxLength validates maximum string length
func MaxLength(fieldName string, value string, max int) error {
	if len(value) > max {
		return errors.InvalidFieldValue(fieldName, value, fmt.Sprintf("maximum length %d", max))
	}
	return nil
}

// StringLength validates string length is within range
func StringLength(fieldName string, value string, min, max int) error {
	if err := MinLength(fieldName, value, min); err != nil {
		return err
	}
	return MaxLength(fieldName, value, max)
}

// MinValue validates minimum numeric value
func MinValue(fieldName string, value, min float64) error {
	if value < min {
		return errors.InvalidFieldValue(fieldName, value, fmt.Sprintf("minimum value %v", min))
	}
	return nil
}

// MaxValue validates maximum numeric value
func MaxValue(fieldName string, value, max float64) error {
	if value > max {
		return errors.InvalidFieldValue(fieldName, value, fmt.Sprintf("maximum value %v", max))
	}
	return nil
}

// Range validates numeric value is within range
func Range(fieldName string, value, min, max float64) error {
	if err := MinValue(fieldName, value, min); err != nil {
		return err
	}
	return MaxValue(fieldName, value, max)
}

// Email validates email format
func Email(fieldName string, value string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(value) {
		return errors.InvalidFieldValue(fieldName, value, "valid email address")
	}
	return nil
}

// URL validates URL format
func URL(fieldName string, value string) error {
	u, err := url.Parse(value)
	if err != nil {
		return errors.InvalidFieldValue(fieldName, value, "valid URL")
	}
	if u.Scheme == "" || u.Host == "" {
		return errors.InvalidFieldValue(fieldName, value, "valid URL with scheme and host")
	}
	return nil
}

// UUID validates UUID format
func UUID(fieldName string, value string) error {
	_, err := uuid.Parse(value)
	if err != nil {
		return errors.InvalidFieldValue(fieldName, value, "valid UUID")
	}
	return nil
}

// IP validates IP address format
func IP(fieldName string, value string) error {
	if net.ParseIP(value) == nil {
		return errors.InvalidFieldValue(fieldName, value, "valid IP address")
	}
	return nil
}

// IPv4 validates IPv4 address format
func IPv4(fieldName string, value string) error {
	ip := net.ParseIP(value)
	if ip == nil || ip.To4() == nil {
		return errors.InvalidFieldValue(fieldName, value, "valid IPv4 address")
	}
	return nil
}

// IPv6 validates IPv6 address format
func IPv6(fieldName string, value string) error {
	ip := net.ParseIP(value)
	if ip == nil || ip.To4() != nil {
		return errors.InvalidFieldValue(fieldName, value, "valid IPv6 address")
	}
	return nil
}

// Port validates port number
func Port(fieldName string, value int) error {
	if value < 1 || value > 65535 {
		return errors.InvalidFieldValue(fieldName, value, "valid port (1-65535)")
	}
	return nil
}

// Hostname validates hostname format
func Hostname(fieldName string, value string) error {
	hostnameRegex := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9]?$`)
	parts := strings.Split(value, ".")
	
	for _, part := range parts {
		if !hostnameRegex.MatchString(part) {
			return errors.InvalidFieldValue(fieldName, value, "valid hostname")
		}
	}
	return nil
}

// FQDN validates fully qualified domain name
func FQDN(fieldName string, value string) error {
	if !strings.Contains(value, ".") {
		return errors.InvalidFieldValue(fieldName, value, "fully qualified domain name")
	}
	return Hostname(fieldName, value)
}

// Pattern validates string matches a regex pattern
func Pattern(fieldName string, value string, pattern string) error {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}
	if !regex.MatchString(value) {
		return errors.InvalidFieldValue(fieldName, value, fmt.Sprintf("match pattern %s", pattern))
	}
	return nil
}

// OneOf validates value is one of allowed values
func OneOf(fieldName string, value interface{}, allowed []interface{}) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return errors.InvalidFieldValue(fieldName, value, fmt.Sprintf("one of %v", allowed))
}

// StringOneOf validates string is one of allowed values
func StringOneOf(fieldName string, value string, allowed []string) error {
	for _, a := range allowed {
		if value == a {
			return nil
		}
	}
	return errors.InvalidFieldValue(fieldName, value, fmt.Sprintf("one of %v", allowed))
}

// NotEmpty validates that a collection is not empty
func NotEmpty(fieldName string, value interface{}) error {
	switch v := value.(type) {
	case string:
		if v == "" {
			return errors.InvalidFieldValue(fieldName, value, "non-empty value")
		}
	case []interface{}:
		if len(v) == 0 {
			return errors.InvalidFieldValue(fieldName, value, "non-empty collection")
		}
	case map[string]interface{}:
		if len(v) == 0 {
			return errors.InvalidFieldValue(fieldName, value, "non-empty map")
		}
	default:
		// For other slice types
		if fmt.Sprintf("%v", v) == "[]" {
			return errors.InvalidFieldValue(fieldName, value, "non-empty collection")
		}
	}
	return nil
}

// Future validates that a time is in the future
func Future(fieldName string, value time.Time) error {
	if !value.After(time.Now()) {
		return errors.InvalidFieldValue(fieldName, value.Format(time.RFC3339), "future date")
	}
	return nil
}

// Past validates that a time is in the past
func Past(fieldName string, value time.Time) error {
	if !value.Before(time.Now()) {
		return errors.InvalidFieldValue(fieldName, value.Format(time.RFC3339), "past date")
	}
	return nil
}

// DateRange validates that a time is within a range
func DateRange(fieldName string, value, min, max time.Time) error {
	if value.Before(min) || value.After(max) {
		return errors.InvalidFieldValue(fieldName, value.Format(time.RFC3339), 
			fmt.Sprintf("between %s and %s", min.Format(time.RFC3339), max.Format(time.RFC3339)))
	}
	return nil
}

// JSONPath validates JSON path format
func JSONPath(fieldName string, value string) error {
	jsonPathRegex := regexp.MustCompile(`^\$(\.[a-zA-Z_][a-zA-Z0-9_]*|\[[0-9]+\]|\[\*\])*$`)
	if !jsonPathRegex.MatchString(value) {
		return errors.InvalidFieldValue(fieldName, value, "valid JSON path")
	}
	return nil
}

// Base64 validates base64 encoding
func Base64(fieldName string, value string) error {
	base64Regex := regexp.MustCompile(`^[A-Za-z0-9+/]*={0,2}$`)
	if !base64Regex.MatchString(value) || len(value)%4 != 0 {
		return errors.InvalidFieldValue(fieldName, value, "valid base64 encoding")
	}
	return nil
}

// HexColor validates hex color format
func HexColor(fieldName string, value string) error {
	hexColorRegex := regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	if !hexColorRegex.MatchString(value) {
		return errors.InvalidFieldValue(fieldName, value, "valid hex color (#RRGGBB)")
	}
	return nil
}

// CreditCard validates credit card number using Luhn algorithm
func CreditCard(fieldName string, value string) error {
	// Remove spaces and dashes
	cleaned := strings.ReplaceAll(strings.ReplaceAll(value, " ", ""), "-", "")
	
	if len(cleaned) < 13 || len(cleaned) > 19 {
		return errors.InvalidFieldValue(fieldName, value, "valid credit card number")
	}
	
	// Luhn algorithm
	sum := 0
	alternate := false
	for i := len(cleaned) - 1; i >= 0; i-- {
		n := int(cleaned[i] - '0')
		if alternate {
			n *= 2
			if n > 9 {
				n = n%10 + 1
			}
		}
		sum += n
		alternate = !alternate
	}
	
	if sum%10 != 0 {
		return errors.InvalidFieldValue(fieldName, value, "valid credit card number")
	}
	
	return nil
}

// PhoneNumber validates phone number format (basic validation)
func PhoneNumber(fieldName string, value string) error {
	phoneRegex := regexp.MustCompile(`^[+]?[0-9]{1,4}?[-.\s]?[(]?[0-9]{1,3}[)]?[-.\s]?[0-9]{1,4}[-.\s]?[0-9]{1,4}[-.\s]?[0-9]{1,9}$`)
	if !phoneRegex.MatchString(value) {
		return errors.InvalidFieldValue(fieldName, value, "valid phone number")
	}
	return nil
}

// AlphaNumeric validates string contains only letters and numbers
func AlphaNumeric(fieldName string, value string) error {
	alphaNumRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !alphaNumRegex.MatchString(value) {
		return errors.InvalidFieldValue(fieldName, value, "alphanumeric characters only")
	}
	return nil
}

// NoSpecialChars validates string contains no special characters
func NoSpecialChars(fieldName string, value string) error {
	specialCharRegex := regexp.MustCompile(`[^a-zA-Z0-9\s]`)
	if specialCharRegex.MatchString(value) {
		return errors.InvalidFieldValue(fieldName, value, "no special characters")
	}
	return nil
}