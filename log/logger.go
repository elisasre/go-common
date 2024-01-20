// Package log provides sane default loggers using slog.
package log

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

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

	logger := slog.New(b.handlerFn(b.output, b.opts))
	slog.SetDefault(logger)

	return logger
}

type builder struct {
	handlerFn HandlerFn
	opts      *slog.HandlerOptions
	output    io.Writer
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
			b.opts.ReplaceAttr = func(s []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					source, ok := a.Value.Any().(*slog.Source)
					if ok && source != nil {
						source.File = filepath.Base(source.File)
					}
				}
				return a
			}
		}
	}
}

// WithReplacer sets slog.HandlerOptions.ReplaceAttr.
func WithReplacer(fn func([]string, slog.Attr) slog.Attr) Opt {
	return func(b *builder) {
		b.opts.ReplaceAttr = fn
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

// ParseFormatFromEnv turns LOG_FORMAT env variable into slog.Handler function using ParseLogFormat.
func ParseFormatFromEnv() HandlerFn {
	return ParseFormat(os.Getenv("LOG_FORMAT"))
}

// ParseFormat turns string into slog.Level using case-insensitive parser.
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
	return ParseSource((os.Getenv("LOG_SOURCE")))
}
