package common

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// EnsureDot ensures that string has ending dot.
func EnsureDot(input string) string {
	if strings.HasSuffix(input, ".") {
		return input
	}
	return input + "."
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
	return v == "true" || v == "t" || v == "yes" || v == "y" || v == "on"
}

func RandomString(n int) (string, error) {
	characterRunes := []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	b := make([]byte, n)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(characterRunes))))
		if err != nil {
			return "", err
		}
		b[i] = characterRunes[num.Int64()]
	}

	return string(b), nil
}
