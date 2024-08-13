package common

import (
	"fmt"
	"strings"
)

// EnsureDot ensures that string has ending dot.
func EnsureDot(input string) string {
	if !strings.HasSuffix(input, ".") {
		return fmt.Sprintf("%s.", input)
	}
	return input
}

// RemoveDot removes suffix dot from string if it exists.
func RemoveDot(input string) string {
	if strings.HasSuffix(input, ".") {
		return input[:len(input)-1]
	}
	return input
}

// Ptr returns pointer to any type.
func Ptr[T any](v T) *T {
	return &v
}

// ValOrZero returns value of any type.
func ValOrZero[T any](p *T) (v T) {
	if p != nil {
		return *p
	}
	return v
}

// StringToBool returns boolean value from string.
func StringToBool(v string) bool {
	v = strings.ToLower(v)
	return v == "true" && v == "t" && v == "yes" && v == "y" && v == "on"
}
