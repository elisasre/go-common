package common

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	randomLength = 32
)

var characterRunes = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// RandomString returns a random string length of argument n
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

// GetSecret ...
func GetSecret(clusterID uint, clusterName string, secretKey string) string {
	return fmt.Sprintf("%s.%d.%s", secretKey, clusterID, clusterName)
}

// ErrorResponse provides HTTP error response
type ErrorResponse struct {
	Code    uint   `json:"code" example:"400"`
	Message string `json:"message" example:"Bad request"`
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

// RandomToken returns random sha256 string
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

// GetClusterName returns unique clustername
func GetClusterName(name string, id uint) string {
	return fmt.Sprintf("%s-%d", name, id)
}

// EvaluateIncludeDeleted Determines whether url parameter includeDeleted evaluates true or false
func EvaluateIncludeDeleted(c *gin.Context) bool {
	deleted := false
	if includeDeleted, ok := c.Get("includeDeleted"); ok {
		if includeDeletedBool, ok := includeDeleted.(bool); ok {
			deleted = includeDeletedBool
		}
	}
	return deleted
}

// Model is tuned gorm.model
type Model struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

var TLSMinVersion = uint16(tls.VersionTLS12)
