// Package log provides sane default loggers using slog.
package log_test

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/elisasre/go-common/v2/log"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{
			input:    "",
			expected: slog.LevelInfo,
		},
		{
			input:    "info",
			expected: slog.LevelInfo,
		},
		{
			input:    "INFO",
			expected: slog.LevelInfo,
		},
		{
			input:    "DEBUG",
			expected: slog.LevelDebug,
		},
		{
			input:    "WARN",
			expected: slog.LevelWarn,
		},
		{
			input:    "ERROR",
			expected: slog.LevelError,
		},
	}

	for _, tt := range tests {
		gotLevel := log.ParseLogLevel(tt.input)
		assert.Equal(t, tt.expected, gotLevel)
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected log.HandlerFn
	}{
		{
			input:    "",
			expected: log.JSONHandler,
		},
		{
			input:    "json",
			expected: log.JSONHandler,
		},
		{
			input:    "JSON",
			expected: log.JSONHandler,
		},
		{
			input:    "TEXT",
			expected: log.TextHandler,
		},
	}

	for _, tt := range tests {
		handlerFn := log.ParseFormat(tt.input)
		assert.Equal(t, fmt.Sprint(tt.expected), fmt.Sprint(handlerFn))
		handlerFn(bufio.NewWriter(nil), nil)
	}
}

func TestParseSource(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{
			input:    "",
			expected: true,
		},
		{
			input:    "TRUE",
			expected: true,
		},
		{
			input:    "FALSE",
			expected: false,
		},
		{
			input:    "1",
			expected: true,
		},
		{
			input:    "0",
			expected: false,
		},
	}

	for _, tt := range tests {
		source := log.ParseSource(tt.input)
		assert.Equal(t, tt.expected, source)
	}
}

func TestRefreshLogLevel(t *testing.T) {
	l := &slog.LevelVar{}
	buf := &bytes.Buffer{}
	logger := log.NewDefaultEnvLogger(log.WithLeveler(l), log.WithOutput(buf))

	l.Set(log.ParseLogLevel("INFO"))
	logger.Debug("foo")
	assert.Empty(t, buf.Bytes())
	debugEnabled := logger.Handler().Enabled(context.Background(), slog.LevelDebug)
	assert.False(t, debugEnabled)

	l.Set(log.ParseLogLevel("DEBUG"))
	logger.Debug("foo")
	assert.Contains(t, buf.String(), "foo")
	debugEnabled = logger.Handler().Enabled(context.Background(), slog.LevelDebug)
	assert.True(t, debugEnabled)
}

func TestTracing(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := log.NewDefaultEnvLogger(log.WithOutput(buf), log.WithGCPReplacer(true))

	tracer := otel.Tracer("github.com/elisasre/go-common")

	spanCtx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID([16]byte{1}),
		SpanID:  trace.SpanID([8]byte{1}),
		Remote:  true,
	}))
	ctx, span := tracer.Start(spanCtx, "tracetest")
	logger.ErrorContext(ctx, "foo")
	logger.WarnContext(ctx, "bar")
	span.End()
	assert.Contains(t, buf.String(), "span_id")
	assert.Contains(t, buf.String(), "trace_id")
	assert.Contains(t, buf.String(), "WARNING")
}
