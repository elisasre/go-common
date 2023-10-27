// Package log provides sane default loggers using slog.
package log_test

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/elisasre/go-common/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestRefreshLogLevel(t *testing.T) {
	l := &slog.LevelVar{}
	tick := time.NewTicker(time.Millisecond)
	done := make(chan struct{})
	go func() {
		defer close(done)
		log.RefreshLogLevel(l, tick)
	}()

	t.Setenv("LOG_LEVEL", "INFO")
	time.Sleep(time.Millisecond * 10)
	require.Equal(t, "INFO", l.Level().String())

	t.Setenv("LOG_LEVEL", "DEBUG")
	time.Sleep(time.Millisecond * 10)
	require.Equal(t, "DEBUG", l.Level().String())

	tick.Stop()
	time.Sleep(time.Millisecond * 10)

	t.Setenv("LOG_LEVEL", "INFO")
	time.Sleep(time.Millisecond * 10)
	require.Equal(t, "DEBUG", l.Level().String())
}

func TestNewDefaultLogger(t *testing.T) {
	logger := log.NewDefaultEnvLogger()
	require.Equal(t, logger, slog.Default())

	debugEnabled := logger.Handler().Enabled(context.Background(), slog.LevelDebug)
	require.False(t, debugEnabled)

	t.Setenv("LOG_LEVEL", "debug")
	time.Sleep(time.Second * 6)

	debugEnabled = logger.Handler().Enabled(context.Background(), slog.LevelDebug)
	require.True(t, debugEnabled)
}
