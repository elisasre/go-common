package sentryutil

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

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
