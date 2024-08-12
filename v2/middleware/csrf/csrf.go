package csrf

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	badTooken        = "CSRF token missing or incorrect."
	tookenMissing    = "CSRF cookie not set."
	noReferer        = "Referer checking failed - no Referer."
	malformedReferer = "Referer checking failed - Referer is malformed."
	insecureReferer  = "Referer checking failed - Referer is insecure while host is secure."
	// CsrfTokenKey is the cookie name which slices.Contains the CSRF token.
	CsrfTokenKey = "csrftoken"
	// Xcsrf is the header name which slices.Contains the CSRF token.
	Xcsrf = "X-CSRF-Token"
	// Authorization is the header name which slices.Contains the token.
	Authorization = "Authorization"

	https = "https"
)

var ignoreMethods = []string{"GET", "HEAD", "OPTIONS", "TRACE"}

func (e ErrorResponse) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

// ErrorResponse provides HTTP error response.
type ErrorResponse struct {
	Code      uint   `json:"code,omitempty" example:"400"`
	Message   string `json:"message" example:"Bad request"`
	ErrorType string `json:"error_type,omitempty" example:"invalid_scope"`
}

// CSRF is middleware for handling CSRF protection in gin.
func CSRF(excludePaths []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// allow machineuser
		if isAPIUser(c) {
			c.Next()
			return
		}

		csrfToken := getCookie(c)

		// Assume that anything not defined as 'safe' by RFC7231 needs protection
		if slices.Contains(ignoreMethods, c.Request.Method) || slices.Contains(excludePaths, c.Request.URL.Path) {
			// set cookie in response if not found
			if csrfToken == "" {
				val, err := RandomToken()
				if err != nil {
					c.JSON(403, ErrorResponse{Code: 403, Message: malformedReferer})
					c.Abort()
					return
				}
				http.SetCookie(c.Writer, &http.Cookie{
					Name:     CsrfTokenKey,
					Value:    val,
					Path:     "/",
					Domain:   c.Request.URL.Host,
					HttpOnly: false,
					Secure:   isHTTPS(c.Request),
					MaxAge:   12 * 60 * 60,
					SameSite: http.SameSiteLaxMode,
				})
			}
			// Set the Vary: Cookie header to protect clients from caching the response.
			c.Header("Vary", "Cookie")
			c.Next()
			return
		}

		if isHTTPS(c.Request) {
			referer := c.Request.Header.Get("Referer")
			if referer == "" {
				c.JSON(403, ErrorResponse{Code: 403, Message: noReferer})
				c.Abort()
				return
			}

			parsedURL, err := url.Parse(referer)
			if err != nil {
				c.JSON(403, ErrorResponse{Code: 403, Message: malformedReferer})
				c.Abort()
				return
			}

			if parsedURL.Scheme != https {
				c.JSON(403, ErrorResponse{Code: 403, Message: insecureReferer})
				c.Abort()
				return
			}

			if parsedURL.Host != c.Request.Host {
				msg := fmt.Sprintf("Referer checking failed - %s does not match any trusted origins.", parsedURL.Host)
				c.JSON(403, ErrorResponse{Code: 403, Message: msg})
				c.Abort()
				return
			}
		}

		requestCSRFToken := getHeader(c)
		if csrfToken == "" {
			c.JSON(403, ErrorResponse{Code: 403, Message: tookenMissing})
			c.Abort()
			return
		}

		if requestCSRFToken != csrfToken {
			c.JSON(403, ErrorResponse{Code: 403, Message: badTooken})
			c.Abort()
			return
		}

		// process request
		c.Next()
	}
}

const (
	randomLength = 32
	sentryKey    = "sentry"
)

// RandomToken returns random sha256 string.
func RandomToken() (string, error) {
	hash := sha256.New()
	r, err := randomString(randomLength)
	if err != nil {
		return "", err
	}
	hash.Write([]byte(r))
	bs := hash.Sum(nil)
	return fmt.Sprintf("%x", bs), nil
}

func randomString(n int) (string, error) {
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

func isHTTPS(r *http.Request) bool {
	switch {
	case r.URL.Scheme == https:
		return true
	case r.TLS != nil:
		return true
	case strings.HasPrefix(strings.ToLower(r.Proto), https):
		return true
	case r.Header.Get("X-Forwarded-Proto") == https:
		return true
	default:
		return false
	}
}

func getHeader(c *gin.Context) string {
	return c.Request.Header.Get(Xcsrf)
}

func isAPIUser(c *gin.Context) bool {
	return c.Request.Header.Get(Authorization) != ""
}

func getCookie(c *gin.Context) string {
	session, err := c.Request.Cookie(CsrfTokenKey)
	if err == nil && session.Value != "" {
		return session.Value
	}
	return ""
}
