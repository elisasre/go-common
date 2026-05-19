package ctxlog_test

import (
	"context"
	"log/slog"
	"os"

	"github.com/elisasre/go-common/v2/ctxlog"
)

// dropTime removes the volatile top-level time attribute so example output is
// deterministic.
func dropTime(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 && a.Key == slog.TimeKey {
		return slog.Attr{}
	}
	return a
}

func ExampleInfo() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: dropTime}))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	ctxlog.Info(ctx, "user logged in", "user", "alice")
	// Output: level=INFO msg="user logged in" user=alice
}

func ExampleWith() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: dropTime}))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	// Bind request-scoped attributes once; every later call carries them.
	ctx = ctxlog.With(ctx, "request_id", "r-123")
	ctxlog.Info(ctx, "handling request")
	ctxlog.Warn(ctx, "slow response")
	// Output:
	// level=INFO msg="handling request" request_id=r-123
	// level=WARN msg="slow response" request_id=r-123
}

func ExampleLogAttrs() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{ReplaceAttr: dropTime}))
	ctx := ctxlog.WithLogger(context.Background(), logger)

	// LogAttrs takes typed slog.Attr values: compile-checked, no !BADKEY risk.
	ctxlog.LogAttrs(ctx, slog.LevelError, "payment failed",
		slog.String("order", "o-9"),
		slog.Int("attempt", 3),
	)
	// Output: level=ERROR msg="payment failed" order=o-9 attempt=3
}
