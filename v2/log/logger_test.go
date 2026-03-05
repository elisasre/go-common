// Package log provides sane default loggers using slog.
package log_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"

	"github.com/elisasre/go-common/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestGCPTraceContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := log.NewDefaultEnvLogger(
		log.WithOutput(buf),
		log.WithGCPReplacer(true),
		log.WithGCPTraceContext("my-project"),
	)

	tracer := otel.Tracer("github.com/elisasre/go-common")

	spanCtx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID([16]byte{1}),
		SpanID:  trace.SpanID([8]byte{1}),
		Remote:  true,
	}))
	ctx, span := tracer.Start(spanCtx, "tracetest")
	logger.InfoContext(ctx, "gcp trace test")
	span.End()

	entry := parseJSONLog(t, buf)
	assert.Equal(t, "projects/my-project/traces/01000000000000000000000000000000", entry["logging.googleapis.com/trace"])
	assert.Equal(t, "0100000000000000", entry["logging.googleapis.com/spanId"])
	assert.NotNil(t, entry["logging.googleapis.com/trace_sampled"])

	// Verify default keys are not present.
	assert.Nil(t, entry["trace_id"])
	assert.Nil(t, entry["span_id"])
}

func TestCustomTraceContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := log.NewDefaultEnvLogger(
		log.WithOutput(buf),
		log.WithTraceContext(log.TraceContext{
			TraceIDKey:      "custom_trace",
			SpanIDKey:       "custom_span",
			TraceSampledKey: "custom_sampled",
		}),
	)

	spanCtx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID([16]byte{1}),
		SpanID:  trace.SpanID([8]byte{1}),
		Remote:  true,
	}))
	tracer := otel.Tracer("github.com/elisasre/go-common")
	ctx, span := tracer.Start(spanCtx, "tracetest")
	logger.InfoContext(ctx, "custom trace test")
	span.End()

	entry := parseJSONLog(t, buf)
	assert.NotNil(t, entry["custom_trace"])
	assert.NotNil(t, entry["custom_span"])
	assert.NotNil(t, entry["custom_sampled"])
}

func TestTracingWithAttrs(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := log.NewDefaultEnvLogger(
		log.WithOutput(buf),
		log.WithGCPTraceContext("my-project"),
	)

	child := logger.With(slog.String("request_id", "abc123"))

	spanCtx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID([16]byte{1}),
		SpanID:  trace.SpanID([8]byte{1}),
		Remote:  true,
	}))
	tracer := otel.Tracer("github.com/elisasre/go-common")
	ctx, span := tracer.Start(spanCtx, "tracetest")
	child.InfoContext(ctx, "with attrs test")
	span.End()

	entry := parseJSONLog(t, buf)
	assert.Equal(t, "abc123", entry["request_id"])
	assert.Equal(t, "projects/my-project/traces/01000000000000000000000000000000", entry["logging.googleapis.com/trace"])
	assert.Equal(t, "0100000000000000", entry["logging.googleapis.com/spanId"])
}

func TestTracingWithGroup(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := log.NewDefaultEnvLogger(
		log.WithOutput(buf),
		log.WithGCPTraceContext("my-project"),
	)

	child := logger.WithGroup("req").With(slog.String("id", "abc123"))

	spanCtx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID([16]byte{1}),
		SpanID:  trace.SpanID([8]byte{1}),
		Remote:  true,
	}))
	tracer := otel.Tracer("github.com/elisasre/go-common")
	ctx, span := tracer.Start(spanCtx, "tracetest")
	child.InfoContext(ctx, "with group test")
	span.End()

	entry := parseJSONLog(t, buf)
	// User attributes should be scoped under the group.
	reqGroup, ok := entry["req"].(map[string]any)
	require.True(t, ok, "expected 'req' group in log output")
	assert.Equal(t, "abc123", reqGroup["id"])
	// Trace attributes should remain top-level, not nested under the group.
	assert.Equal(t, "projects/my-project/traces/01000000000000000000000000000000", entry["logging.googleapis.com/trace"])
	assert.Equal(t, "0100000000000000", entry["logging.googleapis.com/spanId"])
	assert.Nil(t, reqGroup["logging.googleapis.com/trace"])
	assert.Nil(t, reqGroup["logging.googleapis.com/spanId"])
}

func TestTracingNoSpanInContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := log.NewDefaultEnvLogger(
		log.WithOutput(buf),
		log.WithGCPTraceContext("my-project"),
	)

	logger.InfoContext(context.Background(), "no span")

	entry := parseJSONLog(t, buf)
	assert.Equal(t, "no span", entry["msg"])
	assert.Nil(t, entry["logging.googleapis.com/trace"])
	assert.Nil(t, entry["logging.googleapis.com/spanId"])
	assert.Nil(t, entry["logging.googleapis.com/trace_sampled"])
	assert.Nil(t, entry["trace_id"])
	assert.Nil(t, entry["span_id"])
}

func TestWithGCP(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := log.NewDefaultEnvLogger(
		log.WithOutput(buf),
		log.WithGCP("my-project"),
	)

	spanCtx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID([16]byte{1}),
		SpanID:  trace.SpanID([8]byte{1}),
		Remote:  true,
	}))
	tracer := otel.Tracer("github.com/elisasre/go-common")
	ctx, span := tracer.Start(spanCtx, "tracetest")
	logger.WarnContext(ctx, "gcp combined test")
	span.End()

	entry := parseJSONLog(t, buf)
	// GCP replacer fields.
	assert.Equal(t, "WARNING", entry["severity"])
	assert.Equal(t, "gcp combined test", entry["message"])
	assert.NotNil(t, entry["timestamp"])
	assert.Nil(t, entry["level"])
	assert.Nil(t, entry["msg"])
	// GCP trace context fields.
	assert.Equal(t, "projects/my-project/traces/01000000000000000000000000000000", entry["logging.googleapis.com/trace"])
	assert.Equal(t, "0100000000000000", entry["logging.googleapis.com/spanId"])
	// GCP source location.
	srcLoc, ok := entry["logging.googleapis.com/sourceLocation"].(map[string]any)
	require.True(t, ok, "expected 'logging.googleapis.com/sourceLocation' group in log output")
	assert.Equal(t, "logger_test.go", srcLoc["file"])
	assert.NotEmpty(t, srcLoc["line"])
	assert.NotEmpty(t, srcLoc["function"])
	// line should be a string per GCP spec.
	_, isString := srcLoc["line"].(string)
	assert.True(t, isString, "expected 'line' to be a string")
	// Standard source key should not be present.
	assert.Nil(t, entry["source"])
}

func TestGCPSeverityLevels(t *testing.T) {
	tests := []struct {
		level    slog.Level
		expected string
	}{
		{slog.LevelDebug, "DEBUG"},
		{slog.LevelInfo, "INFO"},
		{slog.LevelWarn, "WARNING"},
		{slog.LevelError, "ERROR"},
		{slog.LevelError + 4, "CRITICAL"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := log.NewDefaultEnvLogger(
				log.WithOutput(buf),
				log.WithGCPReplacer(true),
				log.WithLeveler(slog.LevelDebug),
			)
			logger.Log(context.Background(), tt.level, "test")
			entry := parseJSONLog(t, buf)
			assert.Equal(t, tt.expected, entry["severity"])
		})
	}
}

func TestReplaceAttrComposition(t *testing.T) {
	buf := &bytes.Buffer{}

	// Use WithShortSource (from defaults) + a custom replacer that adds a tag.
	// Both should apply since we use addReplacer for WithShortSource
	// and WithGCPReplacer, and only WithReplacer resets.
	logger := log.NewDefaultEnvLogger(
		log.WithOutput(buf),
		log.WithGCPReplacer(true),
	)
	logger.Info("composition test")
	entry := parseJSONLog(t, buf)

	// GCP replacer should have applied (message instead of msg).
	assert.Equal(t, "composition test", entry["message"])
	assert.Equal(t, "INFO", entry["severity"])
	// Source should be GCP sourceLocation (WithGCPReplacer includes short source).
	srcLoc, ok := entry["logging.googleapis.com/sourceLocation"].(map[string]any)
	require.True(t, ok, "expected sourceLocation")
	// File should be short (just filename, not full path) since WithShortSource
	// from defaults runs first, then WithGCPReplacer. Since GCPReplacer replaces the
	// attr entirely, the short source from defaults doesn't matter here — GCPReplacer
	// handles short internally.
	assert.Equal(t, "logger_test.go", srcLoc["file"])
}

func TestWithReplacerResetsChain(t *testing.T) {
	buf := &bytes.Buffer{}

	// WithReplacer should reset the chain — GCP replacer should NOT apply.
	logger := log.NewDefaultEnvLogger(
		log.WithOutput(buf),
		log.WithGCPReplacer(true),
		log.WithReplacer(func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				a.Key = "custom_msg"
			}
			return a
		}),
	)
	logger.Info("replacer reset test")
	entry := parseJSONLog(t, buf)

	// Custom replacer should have applied.
	assert.Equal(t, "replacer reset test", entry["custom_msg"])
	// GCP replacer should NOT have applied (it was reset).
	assert.Nil(t, entry["severity"])
	assert.Nil(t, entry["message"])
}

func TestCustomTraceFormatters(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := log.NewDefaultEnvLogger(
		log.WithOutput(buf),
		log.WithTraceContext(log.TraceContext{
			TraceIDKey: "custom_trace",
			SpanIDKey:  "custom_span",
			TraceIDFormatter: func(id trace.TraceID) string {
				return "trace-" + id.String()
			},
			SpanIDFormatter: func(id trace.SpanID) string {
				return "span-" + id.String()
			},
		}),
	)

	spanCtx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID([16]byte{1}),
		SpanID:  trace.SpanID([8]byte{2}),
		Remote:  true,
	}))
	tracer := otel.Tracer("github.com/elisasre/go-common")
	ctx, span := tracer.Start(spanCtx, "tracetest")
	logger.InfoContext(ctx, "formatter test")
	span.End()

	entry := parseJSONLog(t, buf)
	assert.Equal(t, "trace-01000000000000000000000000000000", entry["custom_trace"])
	assert.Equal(t, "span-0200000000000000", entry["custom_span"])
}

func TestWithGCPReplacerFullPath(t *testing.T) {
	buf := &bytes.Buffer{}
	// Reset the default WithShortSource replacer, then add GCPReplacer with full paths.
	logger := log.NewDefaultEnvLogger(
		log.WithOutput(buf),
		log.WithReplacer(func(_ []string, a slog.Attr) slog.Attr { return a }),
		log.WithGCPReplacer(false),
	)
	logger.Info("full path test")
	entry := parseJSONLog(t, buf)

	assert.Equal(t, "full path test", entry["message"])
	assert.Equal(t, "INFO", entry["severity"])
	srcLoc, ok := entry["logging.googleapis.com/sourceLocation"].(map[string]any)
	require.True(t, ok, "expected sourceLocation")
	file, _ := srcLoc["file"].(string)
	// Full path should contain directory separators.
	assert.Contains(t, file, "/")
}

func TestWithTraceContextLastWins(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := log.NewDefaultEnvLogger(
		log.WithOutput(buf),
		log.WithTraceContext(log.TraceContext{TraceIDKey: "first"}),
		log.WithTraceContext(log.TraceContext{TraceIDKey: "second"}),
	)

	spanCtx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID([16]byte{1}),
		SpanID:  trace.SpanID([8]byte{1}),
		Remote:  true,
	}))
	tracer := otel.Tracer("github.com/elisasre/go-common")
	ctx, span := tracer.Start(spanCtx, "tracetest")
	logger.InfoContext(ctx, "last wins test")
	span.End()

	entry := parseJSONLog(t, buf)
	assert.NotNil(t, entry["second"])
	assert.Nil(t, entry["first"])
}

func TestWithGroupAndAttrsChain(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := log.NewDefaultEnvLogger(
		log.WithOutput(buf),
		log.WithGCPTraceContext("my-project"),
	)

	// Build a chain: root attrs -> group -> group attrs.
	child := logger.With(slog.String("service", "api")).WithGroup("req").With(slog.String("id", "abc123"))

	spanCtx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID([16]byte{1}),
		SpanID:  trace.SpanID([8]byte{1}),
		Remote:  true,
	}))
	tracer := otel.Tracer("github.com/elisasre/go-common")
	ctx, span := tracer.Start(spanCtx, "tracetest")
	child.InfoContext(ctx, "chain test")
	span.End()

	entry := parseJSONLog(t, buf)
	// Root-level attrs.
	assert.Equal(t, "api", entry["service"])
	// Trace attrs must be top-level.
	assert.Equal(t, "projects/my-project/traces/01000000000000000000000000000000", entry["logging.googleapis.com/trace"])
	assert.Equal(t, "0100000000000000", entry["logging.googleapis.com/spanId"])
	// Grouped attrs.
	reqGroup, ok := entry["req"].(map[string]any)
	require.True(t, ok, "expected 'req' group")
	assert.Equal(t, "abc123", reqGroup["id"])
	// Trace attrs should not be in the group.
	assert.Nil(t, reqGroup["logging.googleapis.com/trace"])
}

// parseJSONLog parses the last JSON log line from the buffer.
func parseJSONLog(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	require.NotEmpty(t, lines, "expected at least one log line")
	var entry map[string]any
	require.NoError(t, json.Unmarshal(lines[len(lines)-1], &entry))
	return entry
}
