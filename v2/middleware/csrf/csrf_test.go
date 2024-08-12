package csrf_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elisasre/go-common/v2/middleware/csrf"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestIgnoredMethods(t *testing.T) {
	r := gin.New()
	r.Use(csrf.New(nil))
	for _, method := range []string{"GET", "HEAD", "OPTIONS", "TRACE"} {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(method, "/ping", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, 404, w.Code)
	}
}

func TestCSRFFail(t *testing.T) {
	r := gin.New()
	r.Use(csrf.New(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ping", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}

func TestCSRFMachineUser(t *testing.T) {
	r := gin.New()
	r.Use(csrf.New(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ping", nil)
	req.Header.Add(csrf.Authorization, "foobar")
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestCSRFSucceeded(t *testing.T) {
	r := gin.New()
	r.Use(csrf.New(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ping", nil)
	req.Header.Add(csrf.TokenHeaderKey, "foobar")
	req.AddCookie(&http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestCSRFIncorrect(t *testing.T) {
	r := gin.New()
	r.Use(csrf.New(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ping", nil)
	req.Header.Add(csrf.TokenHeaderKey, "foobar")
	req.AddCookie(&http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar2"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}

func TestCSRFNoRefererSucceeded(t *testing.T) {
	r := gin.New()
	r.Use(csrf.New(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ping", nil)
	req.Header.Add(csrf.TokenHeaderKey, "foobar")
	req.Header.Add("X-Forwarded-Proto", "https")
	req.AddCookie(&http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}

func TestCSRFRefererInvalidURL(t *testing.T) {
	r := gin.New()
	r.Use(csrf.New(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ping", nil)
	req.Header.Add(csrf.TokenHeaderKey, "foobar")
	req.Header.Add("X-Forwarded-Proto", "https")
	req.Header.Add("Referer", "foo")
	req.AddCookie(&http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}

func TestCSRFRefererHTTPURL(t *testing.T) {
	r := gin.New()
	r.Use(csrf.New(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ping", nil)
	req.Header.Add(csrf.TokenHeaderKey, "foobar")
	req.Header.Add("X-Forwarded-Proto", "https")
	req.Header.Add("Referer", "http://foo.fi")
	req.AddCookie(&http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}

func TestCSRFRefererHTTPSURL(t *testing.T) {
	r := gin.New()
	r.Use(csrf.New(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "https://foo.fi/ping", nil)
	req.Header.Add(csrf.TokenHeaderKey, "foobar")
	req.Header.Add("X-Forwarded-Proto", "https")
	req.Header.Add("Referer", "https://foo.fi")
	req.AddCookie(&http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestCSRFDifferentDomainRefererHTTPSURL(t *testing.T) {
	r := gin.New()
	r.Use(csrf.New(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "https://foo.fi/ping", nil)
	req.Header.Add(csrf.TokenHeaderKey, "foobar")
	req.Header.Add("X-Forwarded-Proto", "https")
	req.Header.Add("Referer", "https://foo2.fi")
	req.AddCookie(&http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}

func TestCSRFAllowPaths(t *testing.T) {
	r := gin.New()
	r.Use(csrf.New([]string{"/pingpong"}))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "https://foo.fi/pingpong", nil)
	req.Header.Add(csrf.TokenHeaderKey, "foobar")
	req.Header.Add("X-Forwarded-Proto", "https")
	req.Header.Add("Referer", "https://foo2.fi")
	req.AddCookie(&http.Cookie{Name: csrf.TokenCookieKey, Value: "foobar"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func ExampleNew() {
	r := gin.New()
	excludePaths := []string{"/oauth2/token"}
	r.Use(csrf.New(excludePaths))
}
