package common

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

const (
	badToken         = "CSRF token missing or incorrect." //nolint:gosec
	tokenMissing     = "CSRF cookie not set."             //nolint:gosec
	noReferer        = "Referer checking failed - no Referer."
	malformedReferer = "Referer checking failed - Referer is malformed."
	insecureReferer  = "Referer checking failed - Referer is insecure while host is secure."
	// CsrfTokenKey ...
	CsrfTokenKey = "csrftoken"
	// Xcsrf ...
	Xcsrf = "X-CSRF-Token"
	// Authorization ...
	Authorization = "Authorization"
)

var ignoreMethods = []string{"GET", "HEAD", "OPTIONS", "TRACE"}

// ErrorResponse provides HTTP error response
type ErrorResponse struct {
	Code    uint   `json:"code" example:"400"`
	Message string `json:"message" example:"Bad request"`
}

func getHeader(c *gin.Context) string {
	return c.Request.Header.Get(Xcsrf)
}

func isAPIUser(c *gin.Context) bool {
	return c.Request.Header.Get(Authorization) != ""
}

func isJWTMachineUser(c *gin.Context) bool {
	session, err := c.Request.Cookie("session")
	if err == nil && session.Value != "" {
		return true
	}
	return false
}

func getCookie(c *gin.Context) string {
	session, err := c.Request.Cookie(CsrfTokenKey)
	if err == nil && session.Value != "" {
		return session.Value
	}
	return ""
}

// CSRF is middleware for handling CSRF protection in gin
func CSRF(excludePaths []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// allow machineusers
		if isAPIUser(c) || isJWTMachineUser(c) {
			c.Next()
			return
		}

		csrfToken := getCookie(c)

		// Assume that anything not defined as 'safe' by RFC7231 needs protection
		if ContainsString(ignoreMethods, c.Request.Method) || ContainsString(excludePaths, c.Request.URL.Path) {
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
					Secure:   IsHTTPS(c.Request),
					MaxAge:   12 * 60 * 60,
					SameSite: http.SameSiteLaxMode,
				})
			}
			// Set the Vary: Cookie header to protect clients from caching the response.
			c.Header("Vary", "Cookie")
			c.Next()
			return
		}

		if IsHTTPS(c.Request) {
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

			if parsedURL.Scheme != "https" {
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
			c.JSON(403, ErrorResponse{Code: 403, Message: tokenMissing})
			c.Abort()
			return
		}

		if requestCSRFToken != csrfToken {
			c.JSON(403, ErrorResponse{Code: 403, Message: badToken})
			c.Abort()
			return
		}

		// process request
		c.Next()
	}
}
