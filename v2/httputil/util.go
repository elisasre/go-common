package httputil

import (
	"net/http"
	"strings"
)

func IsHTTPS(r *http.Request) bool {
	const protoHTTPS = "https"
	switch {
	case r.URL.Scheme == protoHTTPS:
		return true
	case r.TLS != nil:
		return true
	case strings.HasPrefix(strings.ToLower(r.Proto), protoHTTPS):
		return true
	case r.Header.Get("X-Forwarded-Proto") == protoHTTPS:
		return true
	default:
		return false
	}
}
