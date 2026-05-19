package ctxlog_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/elisasre/go-common/v2/ctxlog"
	"github.com/elisasre/go-common/v2/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// TestTraceEnrichment locks in the documented dependency: ctxlog does not add
// trace fields itself, but when the logger is the one installed by
// log.NewDefaultEnvLogger (which wraps the handler in spanContextLogHandler),
// trace_id/span_id/trace_sampled flow through because ctxlog forwards ctx
// into Handler().Handle(ctx, r).
func TestTraceEnrichment(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	// NewDefaultEnvLogger reads these; pin them so the test asserts ctxlog
	// behavior, not the ambient shell/CI environment.
	t.Setenv("LOG_FORMAT", "JSON")
	t.Setenv("LOG_LEVEL", "INFO")

	var buf bytes.Buffer
	logger := log.NewDefaultEnvLogger(log.WithOutput(&buf))

	traceID, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("0102030405060708")
	require.NoError(t, err)

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	require.True(t, sc.IsValid())

	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	ctx = ctxlog.WithLogger(ctx, logger)

	ctxlog.Info(ctx, "traced message")

	m := parseJSONLog(t, &buf)
	assert.Equal(t, traceID.String(), m[log.TraceID])
	assert.Equal(t, spanID.String(), m[log.SpanID])
	assert.Equal(t, true, m[log.TraceSampled])
}
