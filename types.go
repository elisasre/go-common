package common

import (
	"fmt"
	"strings"
)

// String returns pointer to string.
func String(s string) *string {
	return &s
}

// StringValue returns string value from pointervalue.
func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// Int returns pointer to int.
func Int(value int) *int {
	return &value
}

// Int64 returns pointer to int64.
func Int64(value int64) *int64 {
	return &value
}

// MapToString modifies map to string array.
func MapToString(input map[string]string) []string {
	result := []string{}
	for key, val := range input {
		result = append(result, fmt.Sprintf("%s=%s", key, val))
	}
	return result
}

// Int64Value returns value from pointer.
func Int64Value(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

// Int32 returns pointer to int32.
func Int32(value int32) *int32 {
	return &value
}

// Int32Value returns value from pointer.
func Int32Value(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}

// UintValue returns value from pointer.
func UintValue(v *uint) uint {
	if v == nil {
		return 0
	}
	return *v
}

// Float64Value returns value from pointer.
func Float64Value(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

// Float64 returns pointer to float64.
func Float64(value float64) *float64 {
	return &value
}

// Bool returns a pointer to a bool.
func Bool(v bool) *bool {
	return &v
}

// BoolValue returns the value of bool pointer or false.
func BoolValue(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

// StringToBool returns boolean value from string.
func StringToBool(value string) bool {
	return strings.ToLower(value) == "true"
}

// StringEmpty returns boolean value if string is empty.
func StringEmpty(value string) bool {
	return value == ""
}
