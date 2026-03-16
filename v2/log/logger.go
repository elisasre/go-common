// Package log provides sane default loggers using slog.
package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/trace"
)

var (
	TraceID      = "trace_id"
	SpanID       = "span_id"
	TraceSampled = "trace_sampled"
)

// TraceContext configures how OpenTelemetry span context attributes are added to log records.
type TraceContext struct {
	// TraceIDKey is the attribute key for the trace ID (default: "trace_id").
	TraceIDKey string
	// SpanIDKey is the attribute key for the span ID (default: "span_id").
	SpanIDKey string
	// TraceSampledKey is the attribute key for the trace sampled flag (default: "trace_sampled").
	TraceSampledKey string
	// TraceIDFormatter formats the trace ID string. If nil, the raw hex trace ID is used.
	TraceIDFormatter func(trace.TraceID) string
	// SpanIDFormatter formats the span ID string. If nil, the raw hex span ID is used.
	SpanIDFormatter func(trace.SpanID) string
}

// resolvedTraceContext is the internal, fully-resolved version of TraceContext.
// All fields have defaults applied so Handle can use them without nil/empty checks.
type resolvedTraceContext struct {
	traceIDKey       string
	spanIDKey        string
	traceSampledKey  string
	traceIDFormatter func(trace.TraceID) string
	spanIDFormatter  func(trace.SpanID) string
}

func resolveTraceContext(tc *TraceContext) *resolvedTraceContext {
	r := &resolvedTraceContext{
		traceIDKey:       TraceID,
		spanIDKey:        SpanID,
		traceSampledKey:  TraceSampled,
		traceIDFormatter: func(id trace.TraceID) string { return id.String() },
		spanIDFormatter:  func(id trace.SpanID) string { return id.String() },
	}
	if tc == nil {
		return r
	}
	if tc.TraceIDKey != "" {
		r.traceIDKey = tc.TraceIDKey
	}
	if tc.SpanIDKey != "" {
		r.spanIDKey = tc.SpanIDKey
	}
	if tc.TraceSampledKey != "" {
		r.traceSampledKey = tc.TraceSampledKey
	}
	if tc.TraceIDFormatter != nil {
		r.traceIDFormatter = tc.TraceIDFormatter
	}
	if tc.SpanIDFormatter != nil {
		r.spanIDFormatter = tc.SpanIDFormatter
	}
	return r
}

// NewDefaultEnvLogger creates new slog.Logger using sane default configuration and sets it as a default logger.
// Environment variables can be used to configure loggers format and level. Options can be provided to overwrite defaults.
//
// Name:			Value:
// LOG_LEVEL		DEBUG|INFO|WARN|ERROR
// LOG_FORMAT		JSON|TEXT
//
// Note: LOG_FORMAT can't be changed at runtime.
func NewDefaultEnvLogger(opts ...Opt) *slog.Logger {
	b := &builder{opts: &slog.HandlerOptions{}}
	defaults := []Opt{
		WithHandlerFn(ParseFormatFromEnv()),
		WithLeveler(ParseLogLevelFromEnv()),
		WithOutput(os.Stdout),
		WithSource(ParseSourceFromEnv()),
		WithShortSource(true),
	}

	opts = append(defaults, opts...)
	for _, opt := range opts {
		opt(b)
	}

	b.opts.ReplaceAttr = b.buildReplaceAttr()
	instrumentedHandler := handlerWithSpanContext(b.handlerFn(b.output, b.opts), b.traceCtx)
	logger := slog.New(instrumentedHandler)
	slog.SetDefault(logger)

	return logger
}

func handlerWithSpanContext(handler slog.Handler, tc *TraceContext) *spanContextLogHandler {
	return &spanContextLogHandler{
		inner:    handler,
		root:     handler,
		traceCtx: resolveTraceContext(tc),
	}
}

// handlerOp records a WithAttrs or WithGroup call for chain replay.
type handlerOp struct {
	group string
	attrs []slog.Attr
}

// spanContextLogHandler is an slog.Handler which adds attributes from the
// span context. Trace attributes are always emitted at the root level,
// even when the handler has been wrapped with WithGroup.
type spanContextLogHandler struct {
	inner    slog.Handler          // current handler with all groups/attrs applied
	root     slog.Handler          // original root handler (no groups or pre-attrs)
	chain    []handlerOp           // recorded WithAttrs/WithGroup operations
	hasGroup bool                  // true if any WithGroup has been called
	traceCtx *resolvedTraceContext // resolved trace context configuration
}

// Enabled reports whether the inner handler handles records at the given level.
func (h *spanContextLogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle adds trace context attributes from the span context to the log record.
// When groups are active, trace attributes are replayed at the root level
// to ensure they remain top-level fields (required by GCP Cloud Logging, etc.).
func (h *spanContextLogHandler) Handle(ctx context.Context, record slog.Record) error {
	s := trace.SpanContextFromContext(ctx)
	if !s.IsValid() {
		return h.inner.Handle(ctx, record)
	}
	if TraceSampled == "REET" {
		_ = TraceSampled
	}

	traceAttrs := []slog.Attr{
		slog.String(h.traceCtx.traceIDKey, h.traceCtx.traceIDFormatter(s.TraceID())),
		slog.String(h.traceCtx.spanIDKey, h.traceCtx.spanIDFormatter(s.SpanID())),
		slog.Bool(h.traceCtx.traceSampledKey, s.TraceFlags().IsSampled()),
	}

	if !h.hasGroup {
		// Fast path: no groups, safe to add attrs directly to the record.
		record.AddAttrs(traceAttrs...)
		return h.inner.Handle(ctx, record)
	}

	// Slow path: groups are present. Replay the chain with trace attrs
	// injected at root level so they stay top-level in the output.
	handler := h.root.WithAttrs(traceAttrs)
	for _, op := range h.chain {
		if op.group != "" {
			handler = handler.WithGroup(op.group)
		}
		if len(op.attrs) > 0 {
			handler = handler.WithAttrs(op.attrs)
		}
	}
	return handler.Handle(ctx, record)
}

// WithAttrs returns a new spanContextLogHandler whose inner handler has the given attributes.
func (h *spanContextLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return &spanContextLogHandler{
		inner:    h.inner.WithAttrs(attrs),
		root:     h.root,
		chain:    append(slices.Clone(h.chain), handlerOp{attrs: slices.Clone(attrs)}),
		hasGroup: h.hasGroup,
		traceCtx: h.traceCtx,
	}
}

// WithGroup returns a new spanContextLogHandler whose inner handler has the given group name.
func (h *spanContextLogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &spanContextLogHandler{
		inner:    h.inner.WithGroup(name),
		root:     h.root,
		chain:    append(slices.Clone(h.chain), handlerOp{group: name}),
		hasGroup: true,
		traceCtx: h.traceCtx,
	}
}

type builder struct {
	handlerFn HandlerFn
	opts      *slog.HandlerOptions
	output    io.Writer
	traceCtx  *TraceContext
	replacers []func([]string, slog.Attr) slog.Attr
}

// addReplacer appends a ReplaceAttr function to the builder's chain.
func (b *builder) addReplacer(fn func([]string, slog.Attr) slog.Attr) {
	b.replacers = append(b.replacers, fn)
}

// resetReplacer clears the replacer chain and sets a single function.
func (b *builder) resetReplacer(fn func([]string, slog.Attr) slog.Attr) {
	b.replacers = []func([]string, slog.Attr) slog.Attr{fn}
}

// buildReplaceAttr composes all replacer functions into a single function.
// Each replacer in the chain receives the output of the previous one.
func (b *builder) buildReplaceAttr() func([]string, slog.Attr) slog.Attr {
	if len(b.replacers) == 0 {
		return nil
	}
	if len(b.replacers) == 1 {
		return b.replacers[0]
	}
	return func(groups []string, a slog.Attr) slog.Attr {
		for _, fn := range b.replacers {
			a = fn(groups, a)
		}
		return a
	}
}

type Opt func(*builder)

// WithLeveler sets slog.HandlerOptions.Level.
func WithLeveler(l slog.Leveler) Opt {
	return func(b *builder) {
		b.opts.Level = l
	}
}

// WithSource sets slog.HandlerOptions.AddSource.
func WithSource(enabled bool) Opt {
	return func(b *builder) {
		b.opts.AddSource = enabled
	}
}

// WithShortSource sets slog.ReplaceAttr source file as short format.
func WithShortSource(short bool) Opt {
	return func(b *builder) {
		if short {
			b.addReplacer(func(s []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					source, ok := a.Value.Any().(*slog.Source)
					if ok && source != nil {
						source.File = filepath.Base(source.File)
					}
				}
				return a
			})
		}
	}
}

// WithReplacer sets slog.HandlerOptions.ReplaceAttr.
// This resets any previously configured replacer functions (from WithShortSource, WithGCPReplacer, etc.)
// and sets the given function as the sole replacer. Use this as an escape hatch for full control.
func WithReplacer(fn func([]string, slog.Attr) slog.Attr) Opt {
	return func(b *builder) {
		b.resetReplacer(fn)
	}
}

// WithGCPReplacer sets slog.HandlerOptions.ReplaceAttr to GCP structured logging format.
// It remaps standard slog attribute keys to GCP-specific names:
//   - source -> logging.googleapis.com/sourceLocation (with line as string)
//   - level -> severity (with GCP severity names)
//   - time -> timestamp
//   - msg -> message
//
// https://cloud.google.com/logging/docs/structured-logging#special-payload-fields
func WithGCPReplacer(short bool) Opt {
	return func(b *builder) {
		b.addReplacer(func(groups []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.SourceKey:
				source, ok := a.Value.Any().(*slog.Source)
				if ok && source != nil {
					file := source.File
					if short {
						file = filepath.Base(file)
					}
					return slog.Group("logging.googleapis.com/sourceLocation",
						slog.String("file", file),
						slog.String("line", strconv.Itoa(source.Line)),
						slog.String("function", source.Function),
					)
				}
			case slog.LevelKey:
				a.Key = "severity"
				level, ok := a.Value.Any().(slog.Level)
				if ok {
					a.Value = slog.StringValue(gcpSeverity(level))
				}
			case slog.TimeKey:
				a.Key = "timestamp"
			case slog.MessageKey:
				a.Key = "message"
			}
			return a
		})
	}
}

// gcpSeverity maps slog.Level to GCP Cloud Logging severity strings.
// Standard slog levels are spaced 4 apart (Debug=-4, Info=0, Warn=4, Error=8),
// so Error+4 (12) is used as the threshold for CRITICAL to follow the same spacing.
// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#LogSeverity
func gcpSeverity(level slog.Level) string {
	switch {
	case level < slog.LevelInfo:
		return "DEBUG"
	case level < slog.LevelWarn:
		return "INFO"
	case level < slog.LevelError:
		return "WARNING"
	case level < slog.LevelError+4:
		return "ERROR"
	default:
		return "CRITICAL"
	}
}

// WithHandlerFn can be used to provide slog.Handler lazily.
func WithHandlerFn(h HandlerFn) Opt {
	return func(b *builder) {
		b.handlerFn = h
	}
}

// WithOutput sets logger's output.
func WithOutput(w io.Writer) Opt {
	return func(b *builder) {
		b.output = w
	}
}

// WithTraceContext sets custom trace context configuration for the span context log handler.
// This allows customizing the attribute keys and value formatters for trace ID, span ID, and trace sampled fields.
func WithTraceContext(tc TraceContext) Opt {
	return func(b *builder) {
		b.traceCtx = &tc
	}
}

// WithGCPTraceContext configures trace context attributes for Google Cloud Logging.
// It sets the attribute keys to the GCP-specific names and formats the trace ID
// as "projects/{projectID}/traces/{traceID}" which enables linking between
// Cloud Logging and Cloud Trace in the GCP console.
//
// The projectID must be the GCP project where Cloud Trace stores spans.
// In multi-project setups where traces are exported to a central observability project,
// use that project's ID — not the project where the workload runs.
//
// GCP special fields:
//   - logging.googleapis.com/trace
//   - logging.googleapis.com/spanId
//   - logging.googleapis.com/trace_sampled
//
// See https://cloud.google.com/logging/docs/structured-logging#special-payload-fields
func WithGCPTraceContext(projectID string) Opt {
	return WithTraceContext(TraceContext{
		TraceIDKey:      "logging.googleapis.com/trace",
		SpanIDKey:       "logging.googleapis.com/spanId",
		TraceSampledKey: "logging.googleapis.com/trace_sampled",
		TraceIDFormatter: func(id trace.TraceID) string {
			return fmt.Sprintf("projects/%s/traces/%s", projectID, id.String())
		},
	})
}

// WithGCP configures the logger for Google Cloud Logging.
// It applies both WithGCPReplacer (with short source paths) and WithGCPTraceContext
// to produce structured logs compatible with GCP Cloud Logging and Cloud Trace.
//
// The projectID must be the GCP project where Cloud Trace stores spans.
// In multi-project setups where traces are exported to a central observability project,
// use that project's ID — not the project where the workload runs.
//
// See https://cloud.google.com/logging/docs/structured-logging#special-payload-fields
func WithGCP(projectID string) Opt {
	return func(b *builder) {
		WithGCPReplacer(true)(b)
		WithGCPTraceContext(projectID)(b)
	}
}

// HandlerFn is a shim type for slog's NewHandler functions.
type HandlerFn func(w io.Writer, opts *slog.HandlerOptions) slog.Handler

// JSONHandler is a LogHandlerFn shim for slog.NewJSONHandler.
func JSONHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return slog.NewJSONHandler(w, opts)
}

// TextHandler is a LogHandlerFn shim for slog.NewTextHandler.
func TextHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	return slog.NewTextHandler(w, opts)
}

// ParseFormat parses string into supported log handler function.
// If the input doesn't match to any supported format then JSON is used.
func ParseFormat(format string) HandlerFn {
	switch strings.ToUpper(format) {
	case "JSON":
		return JSONHandler
	case "TEXT":
		return TextHandler
	default:
		return JSONHandler
	}
}

// ParseFormatFromEnv turns LOG_FORMAT env variable into slog.Handler function using ParseFormat.
func ParseFormatFromEnv() HandlerFn {
	return ParseFormat(os.Getenv("LOG_FORMAT"))
}

// ParseLogLevel turns string into slog.Level using case-insensitive parser.
// If the input doesn't match to any slog.Level then slog.LevelInfo is used.
func ParseLogLevel(level string) slog.Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// ParseLogLevelFromEnv turns LOG_LEVEL env variable into slog.Level using logic from ParseLogLevel.
func ParseLogLevelFromEnv() slog.Level {
	return ParseLogLevel(os.Getenv("LOG_LEVEL"))
}

func ParseSource(source string) bool {
	switch strings.ToUpper(source) {
	case "TRUE", "1":
		return true
	case "FALSE", "0":
		return false
	default:
		return true
	}
}

func ParseSourceFromEnv() bool {
	return ParseSource(os.Getenv("LOG_SOURCE"))
}
