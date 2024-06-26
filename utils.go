package common

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

const (
	randomLength = 32
	sentryKey    = "sentry"
)

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

// LoadAndListenConfig loads config file to struct and listen changes in it.
// User of this function should make sure to protect application state by mutex
// if changing configuration on thr flight might cause date race or other problems
// in application using this functionality.
//
// NOTES:
// When application is run by orchestrator like k8s applying configuration changes by starting
// new instance should be preferred if possible. That way we avoid reimplementing state management
// inside application which is already done by k8s. However for applications with big internal caches
// or otherwise stateful implementations this functionality can offer huge performance benefits.
func LoadAndListenConfig[Conf any](path string, c Conf, onUpdate func(c Conf)) (Conf, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return c, fmt.Errorf("unable to read config: %w", err)
	}
	if err := v.Unmarshal(&c); err != nil {
		return c, fmt.Errorf("unable to marshal config: %w", err)
	}

	slog.Info("config loaded",
		slog.String("path", v.ConfigFileUsed()),
	)
	v.OnConfigChange(func(e fsnotify.Event) {
		slog.Info("config reloaded",
			slog.String("path", e.Name),
			slog.String("operation", e.Op.String()),
		)
		cc := c
		if err := v.Unmarshal(&cc); err != nil {
			slog.Error("unable to marshal config",
				slog.String("path", e.Name),
				slog.String("error", err.Error()),
			)
		}
		if onUpdate != nil {
			onUpdate(cc)
		}
	})
	v.WatchConfig()

	return c, nil
}

var output io.Writer = os.Stdout

// RecoverWithContext recovers from panic and sends it to Sentry.
func RecoverWithContext(ctx context.Context, transaction *sentry.Span) {
	if transaction != nil {
		transaction.Finish()
	}
	if err := recover(); err != nil {
		defer sentry.RecoverWithContext(ctx)
		fmt.Fprintf(output, "panic: %s\n%s\n", err, string(debug.Stack()))
		panic(err)
	}
}

// SentryErr sends error to Sentry.
func SentryErr(ctx context.Context, err error) {
	_, hub := setHubToContext(ctx)
	hub.CaptureException(err)
	slog.Error(err.Error()) //nolint: sloglint
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

func GetFreeLocalhostTCPPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("failed to find free port to listen on: %w", err)
	}
	defer listener.Close()

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("failed to get port from listener")
	}

	return tcpAddr.Port, nil
}
