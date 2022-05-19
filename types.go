package common

import (
	"fmt"
	"strings"
)

var supportedtruevalues = []string{"true", "t", "yes", "y", "on"}

// Ptr returns pointer to any type.
func Ptr[T any](v T) *T {
	return &v
}

// PtrValue returns value of any type.
func PtrValue[T any](p *T) T {
	if p != nil {
		return *p
	}

	var v T
	return v
}

// String returns pointer to string.
func String(s string) *string {
	return Ptr(s)
}

// StringValue returns string value from pointervalue.
func StringValue(s *string) string {
	return PtrValue(s)
}

// Int returns pointer to int.
func Int(value int) *int {
	return Ptr(value)
}

// Int64 returns pointer to int64.
func Int64(value int64) *int64 {
	return Ptr(value)
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
	return PtrValue(v)
}

// Int32 returns pointer to int32.
func Int32(value int32) *int32 {
	return Ptr(value)
}

// Int32Value returns value from pointer.
func Int32Value(v *int32) int32 {
	return PtrValue(v)
}

// UintValue returns value from pointer.
func UintValue(v *uint) uint {
	return PtrValue(v)
}

// Float64Value returns value from pointer.
func Float64Value(v *float64) float64 {
	return PtrValue(v)
}

// Float64 returns pointer to float64.
func Float64(value float64) *float64 {
	return Ptr(value)
}

// Bool returns a pointer to a bool.
func Bool(v bool) *bool {
	return Ptr(v)
}

// BoolValue returns the value of bool pointer or false.
func BoolValue(v *bool) bool {
	return PtrValue(v)
}

// StringToBool returns boolean value from string.
func StringToBool(value string) bool {
	return ContainsString(supportedtruevalues, strings.ToLower(value))
}

// StringEmpty returns boolean value if string is empty.
func StringEmpty(value string) bool {
	return value == ""
}
