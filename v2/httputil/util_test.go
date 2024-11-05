package httputil

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsHTTPS(t *testing.T) {
	tests := []struct {
		name     string
		r        *http.Request
		expected bool
	}{
		{
			name:     "HTTPS scheme",
			r:        &http.Request{URL: &url.URL{Scheme: "https"}},
			expected: true,
		},
		{
			name:     "is TLS",
			r:        &http.Request{TLS: &tls.ConnectionState{}},
			expected: true,
		},
		{
			name:     "HTTPS proto",
			r:        &http.Request{Proto: "HTTPS/1.1"},
			expected: true,
		},
		{
			name:     "X-Forwarded-Proto",
			r:        &http.Request{Header: http.Header{"X-Forwarded-Proto": []string{"https"}}},
			expected: true,
		},
		{
			name:     "not HTTPS",
			r:        &http.Request{},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsHTTPS(tt.r)
			require.Equal(t, tt.expected, got)
		})
	}
}
