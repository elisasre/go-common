package csrf_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elisasre/go-common/v2/middleware/csrf"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSRF(t *testing.T) {
	const testPath = "/ping"

	tests := []struct {
		name               string
		method             string
		url                string
		headers            map[string]string
		cookie             *http.Cookie
		excludePaths       []string
		expectedStatusCode int
		expectedBody       string
	}{
		{
			name:               "Ignore method GET",
			method:             "GET",
			url:                testPath,
			expectedStatusCode: 200,
			expectedBody:       `hello from handler`,
		},
		{
			name:               "Ignore method HEAD",
			method:             "HEAD",
			url:                testPath,
			expectedStatusCode: 200,
			expectedBody:       `hello from handler`,
		},
		{
			name:               "Ignore method OPTIONS",
			method:             "OPTIONS",
			url:                testPath,
			expectedStatusCode: 200,
			expectedBody:       `hello from handler`,
		},
		{
			name:               "Ignore method TRACE",
			method:             "TRACE",
			url:                testPath,
			expectedStatusCode: 200,
			expectedBody:       `hello from handler`,
		},
		{
			name:               "No Cookie",
			method:             "POST",
			url:                testPath,
			expectedStatusCode: 403,
			expectedBody:       `{"code":403,"message":"CSRF cookie not set."}`,
		},
		{
			name:               "Allow anything with Auth header",
			method:             "POST",
			url:                testPath,
			headers:            map[string]string{csrf.Authorization: "hacked"},
			expectedStatusCode: 200,
			expectedBody:       `hello from handler`,
		},
		{
			name:               "Valid CSRF",
			method:             "POST",
			url:                testPath,
			headers:            map[string]string{csrf.TokenHeaderKey: "foobar"},
			cookie:             &http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar"},
			expectedStatusCode: 200,
			expectedBody:       `hello from handler`,
		},
		{
			name:               "Invalid CSRF",
			method:             "POST",
			url:                testPath,
			headers:            map[string]string{csrf.TokenHeaderKey: "foobar"},
			cookie:             &http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar2"},
			expectedStatusCode: 403,
			expectedBody:       `{"code":403,"message":"CSRF token missing or incorrect."}`,
		},
		{
			name:               "Valid CSRF without referer",
			method:             "POST",
			url:                testPath,
			headers:            map[string]string{csrf.TokenHeaderKey: "foobar", "X-Forwarded-Proto": "https"},
			cookie:             &http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar"},
			expectedStatusCode: 403,
			expectedBody:       `{"code":403,"message":"Referer checking failed - no Referer."}`,
		},
		{
			name:               "Valid CSRF non matching referer",
			method:             "POST",
			url:                testPath,
			headers:            map[string]string{csrf.TokenHeaderKey: "foobar", "X-Forwarded-Proto": "https", "Referer": "foo"},
			cookie:             &http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar"},
			expectedStatusCode: 403,
			expectedBody:       `{"code":403,"message":"Referer checking failed - Referer is insecure while host is secure."}`,
		},
		{
			name:               "Valid CSRF HTTP url",
			method:             "POST",
			url:                testPath,
			headers:            map[string]string{csrf.TokenHeaderKey: "foobar", "X-Forwarded-Proto": "https", "Referer": "http://foo.fi"},
			cookie:             &http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar"},
			expectedStatusCode: 403,
			expectedBody:       `{"code":403,"message":"Referer checking failed - Referer is insecure while host is secure."}`,
		},
		{
			name:               "Valid CSRF HTTPS url",
			method:             "POST",
			url:                "https://foo.fi" + testPath,
			headers:            map[string]string{csrf.TokenHeaderKey: "foobar", "X-Forwarded-Proto": "https", "Referer": "https://foo.fi"},
			cookie:             &http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar"},
			expectedStatusCode: 200,
			expectedBody:       `hello from handler`,
		},
		{
			name:               "Unmatching domains",
			method:             "POST",
			url:                testPath,
			headers:            map[string]string{csrf.TokenHeaderKey: "foobar", "X-Forwarded-Proto": "https", "Referer": "https://foo2.fi"},
			cookie:             &http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar"},
			expectedStatusCode: 403,
			expectedBody:       `{"code":403,"message":"Referer checking failed - foo2.fi does not match any trusted origins."}`,
		},
		{
			name:               "Exclude path with unmatching domains",
			method:             "POST",
			url:                "https://foo.fi" + testPath,
			excludePaths:       []string{testPath},
			headers:            map[string]string{csrf.TokenHeaderKey: "foobar", "X-Forwarded-Proto": "https", "Referer": "https://foo2.fi"},
			cookie:             &http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar"},
			expectedStatusCode: 200,
			expectedBody:       `hello from handler`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, nil)
			require.NoErrorf(t, err, "failed to create request with method: %s and url: %s", tt.method, tt.url)

			for k, v := range tt.headers {
				req.Header.Add(k, v)
			}
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			r := gin.New()
			r.Use(csrf.New(tt.excludePaths))
			r.Handle(tt.method, testPath, func(c *gin.Context) {
				_, err := c.Writer.WriteString("hello from handler")
				assert.NoError(t, err)
			})

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatusCode, w.Code)
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestNewV2(t *testing.T) {
	const testPath = "/ping"

	tests := []struct {
		name               string
		trustedOrigins     []string
		excludePaths       []string
		method             string
		url                string
		headers            map[string]string
		expectedStatusCode int
		expectedBody       string
	}{
		{
			name:               "Ignore method GET",
			trustedOrigins:     []string{},
			excludePaths:       []string{},
			method:             "GET",
			url:                testPath,
			expectedStatusCode: 200,
			expectedBody:       "hello from handler",
		},
		{
			name:               "Ignore method HEAD",
			trustedOrigins:     []string{},
			excludePaths:       []string{},
			method:             "HEAD",
			url:                testPath,
			expectedStatusCode: 200,
			expectedBody:       "hello from handler",
		},
		{
			name:               "Ignore method OPTIONS",
			trustedOrigins:     []string{},
			excludePaths:       []string{},
			method:             "OPTIONS",
			url:                testPath,
			expectedStatusCode: 200,
			expectedBody:       "hello from handler",
		},
		{
			name:               "Allow anything with Auth header",
			trustedOrigins:     []string{},
			excludePaths:       []string{},
			method:             "POST",
			url:                testPath,
			headers:            map[string]string{csrf.Authorization: "Bearer token"},
			expectedStatusCode: 200,
			expectedBody:       "hello from handler",
		},
		{
			name:               "POST without headers is allowed as same-origin",
			trustedOrigins:     []string{},
			excludePaths:       []string{},
			method:             "POST",
			url:                testPath,
			expectedStatusCode: 200,
			expectedBody:       "hello from handler",
		},
		{
			name:               "POST with trusted origin succeeds",
			trustedOrigins:     []string{"https://example.com"},
			excludePaths:       []string{},
			method:             "POST",
			url:                "https://example.com" + testPath,
			headers:            map[string]string{"Origin": "https://example.com"},
			expectedStatusCode: 200,
			expectedBody:       "hello from handler",
		},
		{
			name:               "POST with untrusted origin fails",
			trustedOrigins:     []string{"https://example.com"},
			excludePaths:       []string{},
			method:             "POST",
			url:                testPath,
			headers:            map[string]string{"Origin": "https://attacker.com"},
			expectedStatusCode: 403,
		},
		{
			name:               "POST to excluded path bypasses check",
			trustedOrigins:     []string{},
			excludePaths:       []string{testPath},
			method:             "POST",
			url:                testPath,
			expectedStatusCode: 200,
			expectedBody:       "hello from handler",
		},
		{
			name:               "POST with multiple trusted origins",
			trustedOrigins:     []string{"https://example.com", "https://trusted.com"},
			excludePaths:       []string{},
			method:             "POST",
			url:                "https://trusted.com" + testPath,
			headers:            map[string]string{"Origin": "https://trusted.com"},
			expectedStatusCode: 200,
			expectedBody:       "hello from handler",
		},
		{
			name:               "PUT request follows same rules",
			trustedOrigins:     []string{"https://example.com"},
			excludePaths:       []string{},
			method:             "PUT",
			url:                "https://example.com" + testPath,
			headers:            map[string]string{"Origin": "https://example.com"},
			expectedStatusCode: 200,
			expectedBody:       "hello from handler",
		},
		{
			name:               "DELETE request follows same rules",
			trustedOrigins:     []string{"https://example.com"},
			excludePaths:       []string{},
			method:             "DELETE",
			url:                "https://example.com" + testPath,
			headers:            map[string]string{"Origin": "https://example.com"},
			expectedStatusCode: 200,
			expectedBody:       "hello from handler",
		},
		{
			name:               "PATCH request follows same rules",
			trustedOrigins:     []string{"https://example.com"},
			excludePaths:       []string{},
			method:             "PATCH",
			url:                "https://example.com" + testPath,
			headers:            map[string]string{"Origin": "https://example.com"},
			expectedStatusCode: 200,
			expectedBody:       "hello from handler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, nil)
			require.NoErrorf(t, err, "failed to create request with method: %s and url: %s", tt.method, tt.url)

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			r := gin.New()
			middleware, err := csrf.NewV2(tt.trustedOrigins, tt.excludePaths)
			require.NoError(t, err, "failed to create NewV2 middleware")
			r.Use(middleware)
			r.Handle(tt.method, testPath, func(c *gin.Context) {
				_, err := c.Writer.WriteString("hello from handler")
				assert.NoError(t, err)
			})

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatusCode, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestNewV2Errors(t *testing.T) {
	tests := []struct {
		name           string
		trustedOrigins []string
		excludePaths   []string
		errorContains  string
	}{
		{
			name:           "Invalid origin format",
			trustedOrigins: []string{"not-a-valid-url"},
			excludePaths:   []string{},
			errorContains:  "failed to add 'not-a-valid-url' as trustedOrigin",
		},
		{
			name:           "Mixed valid and invalid origins",
			trustedOrigins: []string{"https://example.com", "invalid"},
			excludePaths:   []string{},
			errorContains:  "failed to add 'invalid' as trustedOrigin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware, err := csrf.NewV2(tt.trustedOrigins, tt.excludePaths)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContains)
			assert.Nil(t, middleware)
		})
	}
}

func ExampleNew() {
	r := gin.New()
	excludePaths := []string{"/oauth2/token"}
	r.Use(csrf.New(excludePaths))
}
