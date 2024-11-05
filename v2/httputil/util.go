package httputil

import (
	"fmt"
	"net/http"
	"strings"
)

func IsHTTPS(r *http.Request) bool {
	const protoHTTPS = "https"
	switch {
	case r.URL != nil && r.URL.Scheme == protoHTTPS:
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

func (e ErrorResponse) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

// ErrorResponse provides HTTP error response.
type ErrorResponse struct {
	Code      uint              `json:"code,omitempty" example:"400"`
	Message   string            `json:"message" example:"Bad request"`
	ErrorType string            `json:"error_type,omitempty" example:"invalid_scope"`
	Params    map[string]string `json:"params,omitempty"`
}
