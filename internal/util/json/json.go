package json

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/promptshield/promptshield/internal/shared/errors"
)

// Marshal marshals value to JSON bytes
func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// MarshalIndent marshals value to indented JSON bytes
func MarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

// Unmarshal unmarshals JSON bytes to value
func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// UnmarshalString unmarshals JSON string to value
func UnmarshalString(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}

// MarshalToString marshals value to JSON string
func MarshalToString(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// MarshalToIndentedString marshals value to indented JSON string
func MarshalToIndentedString(v interface{}, prefix, indent string) (string, error) {
	bytes, err := json.MarshalIndent(v, prefix, indent)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// PrettyPrint marshals value to pretty-printed JSON string
func PrettyPrint(v interface{}) (string, error) {
	return MarshalToIndentedString(v, "", "  ")
}

// Compact removes insignificant space characters from JSON
func Compact(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// CompactString removes insignificant space characters from JSON string
func CompactString(data string) (string, error) {
	compacted, err := Compact([]byte(data))
	if err != nil {
		return "", err
	}
	return string(compacted), nil
}

// Indent adds indentation to JSON
func Indent(data []byte, prefix, indent string) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, prefix, indent); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// IndentString adds indentation to JSON string
func IndentString(data string, prefix, indent string) (string, error) {
	indented, err := Indent([]byte(data), prefix, indent)
	if err != nil {
		return "", err
	}
	return string(indented), nil
}

// IsValid reports whether data is a valid JSON encoding
func IsValid(data []byte) bool {
	return json.Valid(data)
}

// IsValidString reports whether data string is valid JSON
func IsValidString(data string) bool {
	return json.Valid([]byte(data))
}

// Encoder wraps json.Encoder with additional functionality
type Encoder struct {
	*json.Encoder
	writer io.Writer
}

// NewEncoder creates a new JSON encoder
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		Encoder: json.NewEncoder(w),
		writer:  w,
	}
}

// SetEscapeHTML sets whether to escape HTML characters
func (e *Encoder) SetEscapeHTML(on bool) {
	e.Encoder.SetEscapeHTML(on)
}

// SetIndent sets the indentation for encoding
func (e *Encoder) SetIndent(prefix, indent string) {
	e.Encoder.SetIndent(prefix, indent)
}

// Decoder wraps json.Decoder with additional functionality
type Decoder struct {
	*json.Decoder
	reader io.Reader
}

// NewDecoder creates a new JSON decoder
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		Decoder: json.NewDecoder(r),
		reader:  r,
	}
}

// DisallowUnknownFields disallows unknown fields
func (d *Decoder) DisallowUnknownFields() {
	d.Decoder.DisallowUnknownFields()
}

// UseNumber uses json.Number for numeric values
func (d *Decoder) UseNumber() {
	d.Decoder.UseNumber()
}

// RawMessage represents a raw JSON message
type RawMessage = json.RawMessage

// Number represents a JSON number
type Number = json.Number

// ToMap converts JSON to map[string]interface{}
func ToMap(data []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ToMapString converts JSON string to map[string]interface{}
func ToMapString(data string) (map[string]interface{}, error) {
	return ToMap([]byte(data))
}

// FromMap converts map to JSON bytes
func FromMap(m map[string]interface{}) ([]byte, error) {
	return json.Marshal(m)
}

// FromMapString converts map to JSON string
func FromMapString(m map[string]interface{}) (string, error) {
	bytes, err := FromMap(m)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ToSlice converts JSON array to []interface{}
func ToSlice(data []byte) ([]interface{}, error) {
	var result []interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ToSliceString converts JSON array string to []interface{}
func ToSliceString(data string) ([]interface{}, error) {
	return ToSlice([]byte(data))
}

// FromSlice converts slice to JSON bytes
func FromSlice(slice []interface{}) ([]byte, error) {
	return json.Marshal(slice)
}

// FromSliceString converts slice to JSON string
func FromSliceString(slice []interface{}) (string, error) {
	bytes, err := FromSlice(slice)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Get gets a value from JSON using a path (e.g., "user.profile.name")
func Get(data []byte, path string) (interface{}, error) {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return GetFromObject(obj, path)
}

// GetString gets a value from JSON string using a path
func GetString(data string, path string) (interface{}, error) {
	return Get([]byte(data), path)
}

// GetFromObject gets a value from an object using a path
func GetFromObject(obj interface{}, path string) (interface{}, error) {
	if path == "" {
		return obj, nil
	}
	
	parts := strings.Split(path, ".")
	current := obj
	
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			var exists bool
			current, exists = v[part]
			if !exists {
				return nil, fmt.Errorf("path not found: %s", path)
			}
		case []interface{}:
			index, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid array index: %s", part)
			}
			if index < 0 || index >= len(v) {
				return nil, fmt.Errorf("array index out of bounds: %d", index)
			}
			current = v[index]
		default:
			return nil, fmt.Errorf("cannot traverse non-object/non-array at path: %s", path)
		}
	}
	
	return current, nil
}

// Set sets a value in JSON using a path
func Set(data []byte, path string, value interface{}) ([]byte, error) {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	
	if err := SetInObject(&obj, path, value); err != nil {
		return nil, err
	}
	
	return json.Marshal(obj)
}

// SetString sets a value in JSON string using a path
func SetString(data string, path string, value interface{}) (string, error) {
	result, err := Set([]byte(data), path, value)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// SetInObject sets a value in an object using a path
func SetInObject(obj *interface{}, path string, value interface{}) error {
	if path == "" {
		*obj = value
		return nil
	}
	
	parts := strings.Split(path, ".")
	current := obj
	
	for i, part := range parts {
		isLast := i == len(parts)-1
		
		switch v := (*current).(type) {
		case map[string]interface{}:
			if isLast {
				v[part] = value
			} else {
				if _, exists := v[part]; !exists {
					v[part] = make(map[string]interface{})
				}
				next := v[part]
				current = &next
				v[part] = next
			}
		case []interface{}:
			index, err := strconv.Atoi(part)
			if err != nil {
				return fmt.Errorf("invalid array index: %s", part)
			}
			if index < 0 || index >= len(v) {
				return fmt.Errorf("array index out of bounds: %d", index)
			}
			if isLast {
				v[index] = value
			} else {
				next := v[index]
				current = &next
				v[index] = next
			}
		case nil:
			// Create new map
			newMap := make(map[string]interface{})
			*current = newMap
			if isLast {
				newMap[part] = value
			} else {
				newMap[part] = make(map[string]interface{})
				next := newMap[part]
				current = &next
				newMap[part] = next
			}
		default:
			return fmt.Errorf("cannot set value in non-object/non-array at path: %s", path)
		}
	}
	
	return nil
}

// Delete removes a value from JSON using a path
func Delete(data []byte, path string) ([]byte, error) {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	
	if err := DeleteFromObject(&obj, path); err != nil {
		return nil, err
	}
	
	return json.Marshal(obj)
}

// DeleteString removes a value from JSON string using a path
func DeleteString(data string, path string) (string, error) {
	result, err := Delete([]byte(data), path)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// DeleteFromObject removes a value from an object using a path
func DeleteFromObject(obj *interface{}, path string) error {
	if path == "" {
		return fmt.Errorf("cannot delete root object")
	}
	
	parts := strings.Split(path, ".")
	current := *obj
	
	// Navigate to parent
	for i, part := range parts[:len(parts)-1] {
		switch v := current.(type) {
		case map[string]interface{}:
			var exists bool
			current, exists = v[part]
			if !exists {
				return fmt.Errorf("path not found: %s", strings.Join(parts[:i+1], "."))
			}
		case []interface{}:
			index, err := strconv.Atoi(part)
			if err != nil {
				return fmt.Errorf("invalid array index: %s", part)
			}
			if index < 0 || index >= len(v) {
				return fmt.Errorf("array index out of bounds: %d", index)
			}
			current = v[index]
		default:
			return fmt.Errorf("cannot traverse non-object/non-array")
		}
	}
	
	// Delete from parent
	lastPart := parts[len(parts)-1]
	switch v := current.(type) {
	case map[string]interface{}:
		delete(v, lastPart)
	case []interface{}:
		index, err := strconv.Atoi(lastPart)
		if err != nil {
			return fmt.Errorf("invalid array index: %s", lastPart)
		}
		if index < 0 || index >= len(v) {
			return fmt.Errorf("array index out of bounds: %d", index)
		}
		// Remove element from slice
		copy(v[index:], v[index+1:])
		v = v[:len(v)-1]
	default:
		return fmt.Errorf("cannot delete from non-object/non-array")
	}
	
	return nil
}

// Merge merges two JSON objects
func Merge(data1, data2 []byte) ([]byte, error) {
	var obj1, obj2 map[string]interface{}
	
	if err := json.Unmarshal(data1, &obj1); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data2, &obj2); err != nil {
		return nil, err
	}
	
	merged := MergeMaps(obj1, obj2)
	return json.Marshal(merged)
}

// MergeString merges two JSON object strings
func MergeString(data1, data2 string) (string, error) {
	result, err := Merge([]byte(data1), []byte(data2))
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// MergeMaps merges two maps recursively
func MergeMaps(map1, map2 map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Copy map1
	for k, v := range map1 {
		result[k] = v
	}
	
	// Merge map2
	for k, v := range map2 {
		if existing, exists := result[k]; exists {
			if existingMap, ok := existing.(map[string]interface{}); ok {
				if vMap, ok := v.(map[string]interface{}); ok {
					result[k] = MergeMaps(existingMap, vMap)
					continue
				}
			}
		}
		result[k] = v
	}
	
	return result
}

// Clone creates a deep copy of JSON data
func Clone(data []byte) ([]byte, error) {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	return json.Marshal(obj)
}

// CloneString creates a deep copy of JSON string
func CloneString(data string) (string, error) {
	result, err := Clone([]byte(data))
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// Keys returns all keys from a JSON object
func Keys(data []byte) ([]string, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	return keys, nil
}

// KeysString returns all keys from a JSON object string
func KeysString(data string) ([]string, error) {
	return Keys([]byte(data))
}

// Values returns all values from a JSON object
func Values(data []byte) ([]interface{}, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	
	values := make([]interface{}, 0, len(obj))
	for _, v := range obj {
		values = append(values, v)
	}
	return values, nil
}

// ValuesString returns all values from a JSON object string
func ValuesString(data string) ([]interface{}, error) {
	return Values([]byte(data))
}

// HasKey checks if a JSON object has a specific key
func HasKey(data []byte, key string) (bool, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return false, err
	}
	
	_, exists := obj[key]
	return exists, nil
}

// HasKeyString checks if a JSON object string has a specific key
func HasKeyString(data string, key string) (bool, error) {
	return HasKey([]byte(data), key)
}

// Size returns the size of a JSON object or array
func Size(data []byte) (int, error) {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return 0, err
	}
	
	switch v := obj.(type) {
	case map[string]interface{}:
		return len(v), nil
	case []interface{}:
		return len(v), nil
	default:
		return 0, fmt.Errorf("not an object or array")
	}
}

// SizeString returns the size of a JSON object or array string
func SizeString(data string) (int, error) {
	return Size([]byte(data))
}

// IsEmpty checks if a JSON object or array is empty
func IsEmpty(data []byte) (bool, error) {
	size, err := Size(data)
	if err != nil {
		return false, err
	}
	return size == 0, nil
}

// IsEmptyString checks if a JSON object or array string is empty
func IsEmptyString(data string) (bool, error) {
	return IsEmpty([]byte(data))
}

// ConvertToType converts JSON value to specific Go type
func ConvertToType(value interface{}, targetType reflect.Type) (interface{}, error) {
	if value == nil {
		return reflect.Zero(targetType).Interface(), nil
	}
	
	valueType := reflect.TypeOf(value)
	if valueType == targetType {
		return value, nil
	}
	
	switch targetType.Kind() {
	case reflect.String:
		return fmt.Sprintf("%v", value), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if num, ok := value.(json.Number); ok {
			return strconv.ParseInt(string(num), 10, 64)
		}
		if str, ok := value.(string); ok {
			return strconv.ParseInt(str, 10, 64)
		}
		if f, ok := value.(float64); ok {
			return int64(f), nil
		}
	case reflect.Float32, reflect.Float64:
		if num, ok := value.(json.Number); ok {
			return strconv.ParseFloat(string(num), 64)
		}
		if str, ok := value.(string); ok {
			return strconv.ParseFloat(str, 64)
		}
	case reflect.Bool:
		if str, ok := value.(string); ok {
			return strconv.ParseBool(str)
		}
	}
	
	return nil, fmt.Errorf("cannot convert %T to %s", value, targetType)
}

// FilterKeys filters JSON object by keys
func FilterKeys(data []byte, keys []string) ([]byte, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}
	
	filtered := make(map[string]interface{})
	keySet := make(map[string]bool)
	for _, key := range keys {
		keySet[key] = true
	}
	
	for k, v := range obj {
		if keySet[k] {
			filtered[k] = v
		}
	}
	
	return json.Marshal(filtered)
}

// FilterKeysString filters JSON object string by keys
func FilterKeysString(data string, keys []string) (string, error) {
	result, err := FilterKeys([]byte(data), keys)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// SafeUnmarshal unmarshals JSON with validation
func SafeUnmarshal(data []byte, v interface{}) error {
	if !IsValid(data) {
		return errors.InvalidRequestFormat("application/json", "invalid JSON")
	}
	
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	
	if err := decoder.Decode(v); err != nil {
		return errors.ValidationFailed("json", err.Error())
	}
	
	return nil
}

// SafeUnmarshalString safely unmarshals JSON string with validation
func SafeUnmarshalString(data string, v interface{}) error {
	return SafeUnmarshal([]byte(data), v)
}