// Package log provides sane default loggers using slog.
package log

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

const DefaultRefreshInterval = time.Second * 5

// NewDefaultEnvLogger creates new slog.Logger using sane default configuration and sets it as a default logger.
// Environment variables can be used to configure loggers format and level. Changing log level at runtime is also supported.
//
// Name:			Value:
// LOG_LEVEL		DEBUG|INFO|WARN|ERROR
// LOG_FORMAT		JSON|TEXT
//
// Note: LOG_FORMAT can't be changed at runtime.
func NewDefaultEnvLogger() *slog.Logger {
	lvl := &slog.LevelVar{}
	lvl.Set(ParseLogLevelFromEnv())
	go RefreshLogLevel(lvl, time.NewTicker(DefaultRefreshInterval))

	handlerFn := ParseFormatEnv()
	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     lvl,
	}

	logger := slog.New(handlerFn(os.Stdout, opts))
	slog.SetDefault(logger)

	return logger
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

// ParseFormatEnv turns LOG_FORMAT env variable into slog.Handler function using ParseLogFormat.
func ParseFormatEnv() HandlerFn {
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

// RefreshLogLevel updates l's value from env with given interval until ticker is stopped.
func RefreshLogLevel(l *slog.LevelVar, t *time.Ticker) {
	for range t.C {
		l.Set(ParseLogLevelFromEnv())
	}
}
