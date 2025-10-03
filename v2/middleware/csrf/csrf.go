package csrf

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"slices"

	"github.com/elisasre/go-common/v2/httputil"
	"github.com/gin-gonic/gin"
)

const (
	// TokenCookieKey is the cookie name which contains the CSRF token.
	TokenCookieKey = "csrftoken"
	// TokenHeaderKey is the header name which contains the CSRF token.
	TokenHeaderKey = "X-CSRF-Token" //nolint: gosec
	// Authorization is the header name which contains the token.
	Authorization = "Authorization"
)

const (
	insecureReferer  = "Referer checking failed - Referer is insecure while host is secure."
	badTooken        = "CSRF token missing or incorrect."
	tokenMissing     = "CSRF cookie not set." //nolint: gosec
	noReferer        = "Referer checking failed - no Referer."
	malformedReferer = "Referer checking failed - Referer is malformed."
	protoHTTPS       = "https"
)

var ignoreMethods = []string{"GET", "HEAD", "OPTIONS", "TRACE"}

// New creates new CSRF middleware for gin.
// Deprecated: Use NewV2.
func New(excludePaths []string) gin.HandlerFunc {
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
					c.JSON(403, httputil.ErrorResponse{Code: 403, Message: malformedReferer})
					c.Abort()
					return
				}
				http.SetCookie(c.Writer, &http.Cookie{
					Name:     TokenCookieKey,
					Value:    val,
					Path:     "/",
					Domain:   c.Request.URL.Host,
					HttpOnly: false,
					Secure:   httputil.IsHTTPS(c.Request),
					MaxAge:   12 * 60 * 60,
					SameSite: http.SameSiteLaxMode,
				})
			}
			// Set the Vary: Cookie header to protect clients from caching the response.
			c.Header("Vary", "Cookie")
			c.Next()
			return
		}

		if httputil.IsHTTPS(c.Request) {
			referer := c.Request.Header.Get("Referer")
			if referer == "" {
				c.JSON(403, httputil.ErrorResponse{Code: 403, Message: noReferer})
				c.Abort()
				return
			}

			parsedURL, err := url.Parse(referer)
			if err != nil {
				c.JSON(403, httputil.ErrorResponse{Code: 403, Message: malformedReferer})
				c.Abort()
				return
			}

			if parsedURL.Scheme != protoHTTPS {
				c.JSON(403, httputil.ErrorResponse{Code: 403, Message: insecureReferer})
				c.Abort()
				return
			}

			if parsedURL.Host != c.Request.Host {
				msg := fmt.Sprintf("Referer checking failed - %s does not match any trusted origins.", parsedURL.Host)
				c.JSON(403, httputil.ErrorResponse{Code: 403, Message: msg})
				c.Abort()
				return
			}
		}

		requestCSRFToken := getHeader(c)
		if csrfToken == "" {
			c.JSON(403, httputil.ErrorResponse{Code: 403, Message: tokenMissing})
			c.Abort()
			return
		}

		if requestCSRFToken != csrfToken {
			c.JSON(403, httputil.ErrorResponse{Code: 403, Message: badTooken})
			c.Abort()
			return
		}

		// process request
		c.Next()
	}
}

// NewV2 creates new CSRF middleware for gin using Go 1.25's built-in CrossOriginProtection.
// trustedOrigins should contain the list of trusted origins (e.g., "https://example.com").
// excludePaths contains URL patterns that should bypass CSRF protection.
func NewV2(trustedOrigins []string, excludePaths []string) (gin.HandlerFunc, error) {
	cop := http.NewCrossOriginProtection()
	for _, origin := range trustedOrigins {
		if err := cop.AddTrustedOrigin(origin); err != nil {
			return nil, fmt.Errorf("failed to add '%s' as trustedOrigin: %w", origin, err)
		}
	}

	for _, path := range excludePaths {
		cop.AddInsecureBypassPattern(path)
	}

	return func(c *gin.Context) {
		if err := cop.Check(c.Request); err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, httputil.ErrorResponse{Code: 403, Message: err.Error()})
			return
		}
		c.Next()
	}, nil
}

// RandomToken returns random sha256 string.
func RandomToken() (string, error) {
	const randomLength = 32

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

func getHeader(c *gin.Context) string {
	return c.Request.Header.Get(TokenHeaderKey)
}

func isAPIUser(c *gin.Context) bool {
	return c.Request.Header.Get(Authorization) != ""
}

func getCookie(c *gin.Context) string {
	session, err := c.Request.Cookie(TokenCookieKey)
	if err == nil && session.Value != "" {
		return session.Value
	}
	return ""
}
