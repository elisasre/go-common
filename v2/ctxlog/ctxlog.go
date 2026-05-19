// Package ctxlog provides context-based logger propagation.
//
// It is a thin convenience layer over log/slog: a *slog.Logger is carried in
// the context, and the package-level logging functions retrieve it (falling
// back to slog.Default()) and emit records with the calling code's source
// location. This lets applications thread a request-scoped logger through
// ctx instead of re-implementing the same "stash logger / pull logger"
// boilerplate everywhere.
//
// ctxlog does not itself add trace_id/span_id attributes. It only forwards
// the context into the handler. Trace enrichment happens when the stored (or
// default) logger's handler reads the span context from ctx — for example the
// handler installed by github.com/elisasre/go-common/v2/log.NewDefaultEnvLogger,
// which is also why FromContext falls back to slog.Default().
//
// ctxlog is a leaf API: source location is captured for the direct caller.
// Wrapping these functions makes every record point at the wrapper; build a
// custom facade on slog.Handler instead.
//
// All functions are safe for concurrent use and tolerate a nil context
// (treated as context.Background()), so a logging call never panics on a
// missing context.
//
// # Call-site safety
//
// The ...any family (Debug, Info, Warn, Error, Log, With) is ergonomic but
// statically unchecked: a mismatched pair yields a !BADKEY attribute at run
// time, not a compile error — the same trade-off as slog.Info vs LogAttrs.
// sloglint cannot see ctxlog calls and no linter can check this family
// correctly (ctxlog.Info(ctx, "m", slog.String("k", v)) is valid but would
// be rejected). Where correctness must be guaranteed use LogAttrs or
// WithAttrs: they take ...slog.Attr, so the compiler rejects a bad call.
//
// # Operational notes
//
// Call With, WithAttrs and WithGroup once per scope, not per loop iteration:
// each adds a context node and wraps the handler (O(n) in a hot loop).
//
// Handler errors are dropped, as in slog.Logger; ctxlog gives no delivery
// guarantee. Enforce delivery at the handler/transport layer if required.
package ctxlog

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

type contextKey struct{}

// WithLogger stores a *slog.Logger in the context.
// A nil ctx is treated as context.Background().
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext retrieves the *slog.Logger from the context.
// Falls back to slog.Default() if ctx is nil, no logger is stored, or a nil
// logger was stored via WithLogger(ctx, nil).
func FromContext(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if l, ok := ctx.Value(contextKey{}).(*slog.Logger); ok && l != nil {
			return l
		}
	}
	return slog.Default()
}

// logCtx builds the record itself so the captured source location points at
// the caller of the exported wrapper rather than at this file. It must be
// called directly by an exported logging function: the runtime.Callers skip
// of 3 is calibrated for the chain
//
//	caller -> ctxlog.<Wrapper> -> logCtx -> runtime.Callers
//
// skipping [runtime.Callers, logCtx, <Wrapper>]. Do not introduce an
// intermediate frame between an exported function and logCtx.
func logCtx(ctx context.Context, level slog.Level, msg string, args ...any) { //nolint:contextcheck // Background is substituted only for a nil ctx (defensive guard)
	if ctx == nil {
		ctx = context.Background()
	}
	l := FromContext(ctx)
	if !l.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip [Callers, logCtx, exported wrapper]
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(args...)
	_ = l.Handler().Handle(ctx, r)
}

// logAttrsCtx is logCtx for the typed-Attr path, kept separate so LogAttrs
// avoids boxing attributes into any. The same skip-3 calibration applies.
func logAttrsCtx(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) { //nolint:contextcheck // Background is substituted only for a nil ctx (defensive guard)
	if ctx == nil {
		ctx = context.Background()
	}
	l := FromContext(ctx)
	if !l.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip [Callers, logAttrsCtx, exported wrapper]
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.AddAttrs(attrs...)
	_ = l.Handler().Handle(ctx, r)
}

// Debug logs at LevelDebug using the logger from ctx.
func Debug(ctx context.Context, msg string, args ...any) {
	logCtx(ctx, slog.LevelDebug, msg, args...)
}

// Info logs at LevelInfo using the logger from ctx.
func Info(ctx context.Context, msg string, args ...any) {
	logCtx(ctx, slog.LevelInfo, msg, args...)
}

// Warn logs at LevelWarn using the logger from ctx.
func Warn(ctx context.Context, msg string, args ...any) {
	logCtx(ctx, slog.LevelWarn, msg, args...)
}

// Error logs at LevelError using the logger from ctx.
func Error(ctx context.Context, msg string, args ...any) {
	logCtx(ctx, slog.LevelError, msg, args...)
}

// Log logs at a caller-supplied level using the logger from ctx.
func Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	logCtx(ctx, level, msg, args...)
}

// LogAttrs logs at the given level using the logger from ctx.
func LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	logAttrsCtx(ctx, level, msg, attrs...)
}

// Enabled reports whether a record at the given level would be emitted by the
// logger in ctx. Use it to guard construction of expensive log arguments.
func Enabled(ctx context.Context, level slog.Level) bool { //nolint:contextcheck // Background is substituted only for a nil ctx (defensive guard)
	if ctx == nil {
		ctx = context.Background()
	}
	return FromContext(ctx).Enabled(ctx, level)
}

// With returns a copy of ctx whose logger has the given attributes added.
// It is the context-scoped analog of slog.Logger.With and derives from
// FromContext(ctx). With no args it returns ctx unchanged.
//
// The returned context must be used. Like context.WithValue, discarding the
// result is a silent no-op — the attributes are lost, not added in place.
//
// The logger is snapshotted at call time: if none is stored in ctx the
// current slog.Default() is captured, so call this only after the
// application has installed its real default logger.
func With(ctx context.Context, args ...any) context.Context {
	if len(args) == 0 {
		return ctx
	}
	return WithLogger(ctx, FromContext(ctx).With(args...))
}

// WithAttrs returns a copy of ctx whose logger has the given typed attributes
// added. It is the compile-checked analog of With (no !BADKEY risk) and the
// context-binding counterpart of LogAttrs. With no attrs it returns ctx
// unchanged.
//
// The returned context must be used; discarding the result is a silent no-op.
// The same snapshot semantics as With apply.
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	return WithLogger(ctx, slog.New(FromContext(ctx).Handler().WithAttrs(attrs)))
}

// WithGroup returns a copy of ctx whose logger starts a group with the given
// name. It is the context-scoped analog of slog.Logger.WithGroup and derives
// from FromContext(ctx). An empty name returns ctx unchanged, mirroring
// slog.Logger.WithGroup.
//
// The returned context must be used; discarding the result is a silent no-op.
// The same snapshot semantics as With apply.
func WithGroup(ctx context.Context, name string) context.Context {
	if name == "" {
		return ctx
	}
	return WithLogger(ctx, FromContext(ctx).WithGroup(name))
}
