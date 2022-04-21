package common

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"net/http"
	"strings"
)

const (
	randomLength = 32
)

var characterRunes = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// RandomString returns a random string length of argument n.
func RandomString(n int) (string, error) {
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

// RandomToken returns random sha256 string.
func RandomToken() (string, error) {
	hash := sha256.New()
	r, err := RandomString(randomLength)
	if err != nil {
		return "", err
	}
	hash.Write([]byte(r))
	bs := hash.Sum(nil)
	return fmt.Sprintf("%x", bs), nil
}

// IsHTTPS is a helper function that evaluates the http.Request
// and returns True if the Request uses HTTPS. It is able to detect,
// using the X-Forwarded-Proto, if the original request was HTTPS and
// routed through a reverse proxy with SSL termination.
func IsHTTPS(r *http.Request) bool {
	switch {
	case r.URL.Scheme == "https":
		return true
	case r.TLS != nil:
		return true
	case strings.HasPrefix(r.Proto, "HTTPS"):
		return true
	case r.Header.Get("X-Forwarded-Proto") == "https":
		return true
	default:
		return false
	}
}

// MinUint ...
func MinUint(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}

// EnsureDot ...
func EnsureDot(input string) string {
	if !strings.HasSuffix(input, ".") {
		return fmt.Sprintf("%s.", input)
	}
	return input
}

// RemoveDot ...
func RemoveDot(input string) string {
	if strings.HasSuffix(input, ".") {
		return input[:len(input)-1]
	}
	return input
}
