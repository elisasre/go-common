package ctxlog_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/elisasre/go-common/v2/ctxlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseJSONLog(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	return m
}

func TestFromContext_Default(t *testing.T) {
	ctx := context.Background()
	logger := ctxlog.FromContext(ctx)
	assert.Equal(t, slog.Default(), logger)
}

func TestFromContext_WithLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	got := ctxlog.FromContext(ctx)
	assert.Equal(t, logger, got)
}

func TestDebug(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	ctxlog.Debug(ctx, "debug msg", "key", "val")

	m := parseJSONLog(t, &buf)
	assert.Equal(t, "DEBUG", m["level"])
	assert.Equal(t, "debug msg", m["msg"])
	assert.Equal(t, "val", m["key"])
}

func TestInfo(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	ctxlog.Info(ctx, "info msg", "key", "val")

	m := parseJSONLog(t, &buf)
	assert.Equal(t, "INFO", m["level"])
	assert.Equal(t, "info msg", m["msg"])
	assert.Equal(t, "val", m["key"])
}

func TestWarn(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	ctxlog.Warn(ctx, "warn msg", "key", "val")

	m := parseJSONLog(t, &buf)
	assert.Equal(t, "WARN", m["level"])
	assert.Equal(t, "warn msg", m["msg"])
	assert.Equal(t, "val", m["key"])
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	ctxlog.Error(ctx, "error msg", "key", "val")

	m := parseJSONLog(t, &buf)
	assert.Equal(t, "ERROR", m["level"])
	assert.Equal(t, "error msg", m["msg"])
	assert.Equal(t, "val", m["key"])
}

func TestLog(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	ctxlog.Log(ctx, slog.LevelWarn, "log msg", "key", "val")

	m := parseJSONLog(t, &buf)
	assert.Equal(t, "WARN", m["level"])
	assert.Equal(t, "log msg", m["msg"])
	assert.Equal(t, "val", m["key"])
}

func TestLogAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	ctxlog.LogAttrs(ctx, slog.LevelWarn, "attrs msg", slog.String("key", "val"))

	m := parseJSONLog(t, &buf)
	assert.Equal(t, "WARN", m["level"])
	assert.Equal(t, "attrs msg", m["msg"])
	assert.Equal(t, "val", m["key"])
}

// TestSource_PointsAtCallSite is the regression gate for the PC-skip bug:
// the source location must point at the calling file, never at ctxlog.go.
// Every exported logging function is covered because each is an independent
// entry point into the skip-3 chain — a refactor that adds a frame to any one
// of them must fail here rather than ship a silently-wrong source location.
// It deliberately asserts only file and function (not an exact line) so that
// reformatting or editing this test does not produce confusing failures.
func TestSource_PointsAtCallSite(t *testing.T) {
	calls := []struct {
		name string
		emit func(ctx context.Context)
	}{
		{"Debug", func(ctx context.Context) { ctxlog.Debug(ctx, "m") }},
		{"Info", func(ctx context.Context) { ctxlog.Info(ctx, "m") }},
		{"Warn", func(ctx context.Context) { ctxlog.Warn(ctx, "m") }},
		{"Error", func(ctx context.Context) { ctxlog.Error(ctx, "m") }},
		{"Log", func(ctx context.Context) { ctxlog.Log(ctx, slog.LevelInfo, "m") }},
		{"LogAttrs", func(ctx context.Context) { ctxlog.LogAttrs(ctx, slog.LevelInfo, "m") }},
	}

	for _, c := range calls {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelDebug, // ensure Debug emits too
			}))
			ctx := ctxlog.WithLogger(context.Background(), logger)

			c.emit(ctx)

			m := parseJSONLog(t, &buf)
			src, ok := m["source"].(map[string]any)
			require.True(t, ok, "source attribute missing: %v", m)

			file, _ := src["file"].(string)
			assert.Equal(t, "ctxlog_test.go", filepath.Base(file),
				"source.file must be the caller's file, not the ctxlog package")

			fn, _ := src["function"].(string)
			assert.Contains(t, fn, "TestSource_PointsAtCallSite",
				"source.function should be the caller, got %q", fn)
		})
	}
}

func TestWithLogger_TypedNil(t *testing.T) {
	ctx := ctxlog.WithLogger(context.Background(), nil)

	assert.Equal(t, slog.Default(), ctxlog.FromContext(ctx))
	assert.NotPanics(t, func() {
		ctxlog.Info(ctx, "no panic with nil-stored logger")
	})
}

func TestEnabledShortCircuit(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	ctxlog.Debug(ctx, "should be dropped")

	assert.Zero(t, buf.Len(), "debug below handler level must not emit")
}

// TestLogAttrs_Disabled covers logAttrsCtx's Enabled short-circuit, the
// typed-Attr counterpart of TestEnabledShortCircuit.
func TestLogAttrs_Disabled(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	ctxlog.LogAttrs(ctx, slog.LevelInfo, "should be dropped", slog.String("k", "v"))

	assert.Zero(t, buf.Len(), "LogAttrs below handler level must not emit")
}

func TestWith(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	assert.Equal(t, ctx, ctxlog.With(ctx), "With with no args must return ctx unchanged")

	ctx = ctxlog.With(ctx, "rid", "abc")
	ctx = ctxlog.With(ctx, "k2", "v2")
	ctxlog.Info(ctx, "m")

	m := parseJSONLog(t, &buf)
	assert.Equal(t, "abc", m["rid"], "chained With must retain earlier attrs")
	assert.Equal(t, "v2", m["k2"])
}

func TestWith_DefaultFallback(t *testing.T) {
	assert.NotPanics(t, func() {
		ctx := ctxlog.With(context.Background(), "k", "v")
		ctxlog.Info(ctx, "m")
	})
}

func TestWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	assert.Equal(t, ctx, ctxlog.WithAttrs(ctx), "WithAttrs with no attrs must return ctx unchanged")

	ctx = ctxlog.WithAttrs(ctx, slog.String("rid", "abc"))
	ctx = ctxlog.WithAttrs(ctx, slog.Int("attempt", 2))
	ctxlog.Info(ctx, "m")

	m := parseJSONLog(t, &buf)
	assert.Equal(t, "abc", m["rid"], "chained WithAttrs must retain earlier attrs")
	assert.EqualValues(t, 2, m["attempt"])
}

func TestWithGroup(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	gctx := ctxlog.WithGroup(ctx, "g")
	ctxlog.Info(gctx, "m", "k", "v")

	m := parseJSONLog(t, &buf)
	group, ok := m["g"].(map[string]any)
	require.True(t, ok, "expected nested group %q in %v", "g", m)
	assert.Equal(t, "v", group["k"])
}

func TestWithGroup_EmptyName(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	assert.Equal(t, ctx, ctxlog.WithGroup(ctx, ""),
		"WithGroup with empty name must return ctx unchanged")

	ctx = ctxlog.WithGroup(ctx, "")
	ctxlog.Info(ctx, "m", "k", "v")

	m := parseJSONLog(t, &buf)
	assert.Equal(t, "v", m["k"], "empty group name keeps attrs top-level")
}

func TestNilContext(t *testing.T) {
	var nilCtx context.Context // typed nil: the case under test

	assert.Equal(t, slog.Default(), ctxlog.FromContext(nilCtx))
	assert.NotPanics(t, func() {
		ctxlog.Info(nilCtx, "no panic on nil ctx")
		ctxlog.LogAttrs(nilCtx, slog.LevelInfo, "no panic", slog.Int("n", 1))
		ctx := ctxlog.WithLogger(nilCtx, slog.Default())
		ctxlog.With(ctx, "k", "v")
		ctxlog.WithGroup(ctx, "g")
		_ = ctxlog.Enabled(nilCtx, slog.LevelInfo)
	})
}

func TestEnabled(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn})) //nolint:sloglint // real level-gating handler required; slog.DiscardHandler.Enabled is always false
	ctx := ctxlog.WithLogger(context.Background(), logger)

	assert.False(t, ctxlog.Enabled(ctx, slog.LevelInfo))
	assert.True(t, ctxlog.Enabled(ctx, slog.LevelError))
}

// TestDisabledPathZeroAlloc guards the Enabled short-circuit: a below-level
// call with no args must not allocate, so logging stays cheap on hot paths.
func TestDisabledPathZeroAlloc(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn})) //nolint:sloglint // real level-gating handler required; slog.DiscardHandler.Enabled is always false
	ctx := ctxlog.WithLogger(context.Background(), logger)

	avg := testing.AllocsPerRun(100, func() {
		ctxlog.Debug(ctx, "skipped")
	})
	assert.Zero(t, avg, "disabled-level fast path must not allocate")
}

// countingWriter counts Write calls. slog's JSON handler emits one Write per
// record, so the count equals the number of records written.
type countingWriter struct{ n atomic.Int64 }

func (w *countingWriter) Write(p []byte) (int, error) {
	w.n.Add(1)
	return len(p), nil
}

// TestConcurrentUse backs the "safe for concurrent use" doc claim: many
// goroutines derive their own context with With and log on it simultaneously.
// Run with -race; every record must be emitted exactly once.
func TestConcurrentUse(t *testing.T) {
	w := &countingWriter{}
	logger := slog.New(slog.NewJSONHandler(w, nil))
	base := ctxlog.WithLogger(context.Background(), logger)

	const goroutines, perG = 50, 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := range goroutines {
		go func() {
			defer wg.Done()
			gctx := ctxlog.With(base, "goroutine", g)
			for i := range perG {
				ctxlog.Info(gctx, "concurrent", "i", i)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(goroutines*perG), w.n.Load())
}

func BenchmarkInfo(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil)) //nolint:sloglint // benchmark must exercise the real encode path, not a no-op handler
	ctx := ctxlog.WithLogger(context.Background(), logger)
	b.ReportAllocs()
	for b.Loop() {
		ctxlog.Info(ctx, "msg", "key", "val")
	}
}

func BenchmarkInfoDisabled(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelWarn})) //nolint:sloglint // real level-gating handler required; slog.DiscardHandler.Enabled is always false
	ctx := ctxlog.WithLogger(context.Background(), logger)
	b.ReportAllocs()
	for b.Loop() {
		ctxlog.Debug(ctx, "skipped")
	}
}
