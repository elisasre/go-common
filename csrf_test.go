package common

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Debug is middleware used for testing purposes
func Debug() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Printf("%v %+v\n", isAPIUser(c), c.Request.Header)
		c.Next()
	}
}

func TestIgnoredMethods(t *testing.T) {
	r := gin.New()
	r.Use(CSRF(nil))
	for _, method := range ignoreMethods {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(method, "/ping", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, 404, w.Code)
	}
}

func TestCSRFFail(t *testing.T) {
	r := gin.New()
	r.Use(CSRF(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ping", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}

func TestCSRFMachineUser(t *testing.T) {
	r := gin.New()
	r.Use(CSRF(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ping", nil)
	req.Header.Add(Authorization, "foobar")
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestCSRFJWTMachineUser(t *testing.T) {
	r := gin.New()
	r.Use(CSRF(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ping", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: "foobar"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestCSRFSucceeded(t *testing.T) {
	r := gin.New()
	r.Use(CSRF(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ping", nil)
	req.Header.Add(Xcsrf, "foobar")
	req.AddCookie(&http.Cookie{Name: CsrfTokenKey, Value: "foobar"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestCSRFIncorrect(t *testing.T) {
	r := gin.New()
	r.Use(CSRF(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ping", nil)
	req.Header.Add(Xcsrf, "foobar")
	req.AddCookie(&http.Cookie{Name: CsrfTokenKey, Value: "foobar2"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}

func TestCSRFNoRefererSucceeded(t *testing.T) {
	r := gin.New()
	r.Use(CSRF(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ping", nil)
	req.Header.Add(Xcsrf, "foobar")
	req.Header.Add("X-Forwarded-Proto", "https")
	req.AddCookie(&http.Cookie{Name: CsrfTokenKey, Value: "foobar"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}

func TestCSRFRefererInvalidURL(t *testing.T) {
	r := gin.New()
	r.Use(CSRF(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ping", nil)
	req.Header.Add(Xcsrf, "foobar")
	req.Header.Add("X-Forwarded-Proto", "https")
	req.Header.Add("Referer", "foo")
	req.AddCookie(&http.Cookie{Name: CsrfTokenKey, Value: "foobar"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}

func TestCSRFRefererHTTPURL(t *testing.T) {
	r := gin.New()
	r.Use(CSRF(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/ping", nil)
	req.Header.Add(Xcsrf, "foobar")
	req.Header.Add("X-Forwarded-Proto", "https")
	req.Header.Add("Referer", "http://foo.fi")
	req.AddCookie(&http.Cookie{Name: CsrfTokenKey, Value: "foobar"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}

func TestCSRFRefererHTTPSURL(t *testing.T) {
	r := gin.New()
	r.Use(CSRF(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "https://foo.fi/ping", nil)
	req.Header.Add(Xcsrf, "foobar")
	req.Header.Add("X-Forwarded-Proto", "https")
	req.Header.Add("Referer", "https://foo.fi")
	req.AddCookie(&http.Cookie{Name: CsrfTokenKey, Value: "foobar"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestCSRFDifferentDomainRefererHTTPSURL(t *testing.T) {
	r := gin.New()
	r.Use(CSRF(nil))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "https://foo.fi/ping", nil)
	req.Header.Add(Xcsrf, "foobar")
	req.Header.Add("X-Forwarded-Proto", "https")
	req.Header.Add("Referer", "https://foo2.fi")
	req.AddCookie(&http.Cookie{Name: CsrfTokenKey, Value: "foobar"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 403, w.Code)
}

func TestCSRFAllowPaths(t *testing.T) {
	r := gin.New()
	r.Use(CSRF([]string{"/pingpong"}))
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "https://foo.fi/pingpong", nil)
	req.Header.Add(xcsrf, "foobar")
	req.Header.Add("X-Forwarded-Proto", "https")
	req.Header.Add("Referer", "https://foo2.fi")
	req.AddCookie(&http.Cookie{Name: csrfTokenKey, Value: "foobar"})
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}
