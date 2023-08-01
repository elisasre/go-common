package common

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	randomLength = 32
	sentryKey    = "sentry"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	log.Logger = zerolog.New(os.Stderr)
	log.Logger = log.With().Logger()
}

var characterRunes = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// RandomString returns a random string length of argument n.
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

// RandomToken returns random sha256 string.
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

// MinUint calculates Min from a, b.
func MinUint(a, b uint) uint {
	if a < b {
		return a
	}
	return b
}

// EnsureDot ensures that string has ending dot.
func EnsureDot(input string) string {
	if !strings.HasSuffix(input, ".") {
		return fmt.Sprintf("%s.", input)
	}
	return input
}

// RemoveDot removes suffix dot from string if it exists.
func RemoveDot(input string) string {
	if strings.HasSuffix(input, ".") {
		return input[:len(input)-1]
	}
	return input
}

type ConfigLock struct {
	lock sync.Mutex
}

// LoadAndListenConfig loads config file to struct and listen changes in it.
func LoadAndListenConfig(path string, l *ConfigLock, obj interface{}, onUpdate func(oldObj interface{})) error {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("unable to read config: %w", err)
	}
	l.lock.Lock()
	if err := v.Unmarshal(&obj); err != nil {
		return fmt.Errorf("unable to marshal config: %w", err)
	}
	l.lock.Unlock()
	log.Info().
		Str("path", v.ConfigFileUsed()).
		Msg("config loaded")
	v.OnConfigChange(func(e fsnotify.Event) {
		log.Info().
			Str("path", e.Name).
			Msg("config reloaded")
		oldObj := reflect.Indirect(reflect.ValueOf(obj)).Interface()
		l.lock.Lock()
		if err := v.Unmarshal(&obj); err != nil {
			log.Fatal().
				Str("path", e.Name).
				Msgf("unable to marshal config: %v", err)
		}
		l.lock.Unlock()
		if onUpdate != nil {
			onUpdate(oldObj)
		}
	})
	v.WatchConfig()
	return nil
}

// RecoverWithContext recovers from panic and sends it to Sentry.
func RecoverWithContext(ctx context.Context, transaction *sentry.Span) {
	if transaction != nil {
		transaction.Finish()
	}
	if err := recover(); err != nil {
		defer sentry.RecoverWithContext(ctx)
		panic(err)
	}
}

// MakeSentryTransaction creates Sentry transaction.
func MakeSentryTransaction(ctx context.Context, name string, opts ...sentry.SpanOption) (context.Context, *sentry.Span, *sentry.Hub) {
	var hub *sentry.Hub
	ctx, hub = setHubToContext(ctx)
	options := []sentry.SpanOption{
		sentry.WithOpName(name),
	}
	options = append(options, opts...)
	transaction := sentry.StartTransaction(ctx,
		name,
		options...,
	)
	return transaction.Context(), transaction, hub
}

func setHubToContext(ctx context.Context) (context.Context, *sentry.Hub) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
		ctx = sentry.SetHubOnContext(ctx, hub)
	}
	return ctx, hub
}

// sentrySpanTracer middleware for sentry span time reporting.
func sentrySpanTracer() gin.HandlerFunc {
	return func(c *gin.Context) {
		span := sentry.StartSpan(c.Request.Context(), c.HandlerName())
		defer span.Finish()
		c.Next()
	}
}

// MakeSpan makes new sentry span.
func MakeSpan(ctx context.Context, skip int) *sentry.Span {
	pc, _, _, _ := runtime.Caller(skip) //nolint:dogsled
	tmp := runtime.FuncForPC(pc)
	spanName := "nil"
	if tmp != nil {
		spanName = tmp.Name()
	}
	span := sentry.StartSpan(ctx, spanName)
	return span
}

// GET wrapper to include sentrySpanTracer as last middleware.
func GET(group *gin.RouterGroup, relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return group.Handle(http.MethodGet, relativePath, addSpanTracer(handlers)...)
}

// PUT wrapper to include sentrySpanTracer as last middleware.
func PUT(group *gin.RouterGroup, relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return group.Handle(http.MethodPut, relativePath, addSpanTracer(handlers)...)
}

// POST wrapper to include sentrySpanTracer as last middleware.
func POST(group *gin.RouterGroup, relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return group.Handle(http.MethodPost, relativePath, addSpanTracer(handlers)...)
}

// DELETE wrapper to include sentrySpanTracer as last middleware.
func DELETE(group *gin.RouterGroup, relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return group.Handle(http.MethodDelete, relativePath, addSpanTracer(handlers)...)
}

// PATCH wrapper to include sentrySpanTracer as last middleware.
func PATCH(group *gin.RouterGroup, relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return group.Handle(http.MethodPatch, relativePath, addSpanTracer(handlers)...)
}

func addSpanTracer(handlers []gin.HandlerFunc) []gin.HandlerFunc {
	lastElement := handlers[len(handlers)-1]
	handlers = handlers[:len(handlers)-1]
	handlers = append(handlers, sentrySpanTracer(), lastElement)
	return handlers
}
