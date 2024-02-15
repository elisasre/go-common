// Package sentry provides sentry functionality as a module.
package sentry

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/getsentry/sentry-go"
)

type Sentry struct {
	conf     sentry.ClientOptions
	initWrap func(error) error
	stopped  chan struct{}
	opts     []Opt
}

// New creates sentry module with given options.
func New(opts ...Opt) *Sentry {
	return &Sentry{opts: opts}
}

func (s *Sentry) Init() error {
	s.stopped = make(chan struct{})
	s.initWrap = func(err error) error { return err }

	for _, opt := range s.opts {
		if err := opt(s); err != nil {
			return fmt.Errorf("sentry.Sentry Option error: %w", err)
		}
	}

	return s.initWrap(sentry.Init(s.conf))
}

func (s *Sentry) Run() error {
	defer sentry.Flush(5 * time.Second)
	defer sentry.Recover()
	<-s.stopped
	return nil
}

func (s *Sentry) Stop() error {
	close(s.stopped)
	return nil
}

func (s *Sentry) Name() string {
	return "sentry.Sentry"
}

type Opt func(*Sentry) error

// WithIgnoreInitErr causes module to only log init errors instead of preventing the service to start.
func WithIgnoreInitErr() Opt {
	return func(s *Sentry) error {
		s.initWrap = func(err error) error {
			slog.Info("ignoring sentry init error",
				slog.String("error", err.Error()))
			return nil
		}
		return nil
	}
}

func WithClientOptions(opt sentry.ClientOptions) Opt {
	return func(s *Sentry) error {
		s.conf = opt
		return nil
	}
}

func WithDSN(dsn string) Opt {
	return func(s *Sentry) error {
		s.conf.Dsn = dsn
		return nil
	}
}

func WithAttachStacktrace(b bool) Opt {
	return func(s *Sentry) error {
		s.conf.AttachStacktrace = b
		return nil
	}
}

func WithSampleRate(rate float64) Opt {
	return func(s *Sentry) error {
		s.conf.SampleRate = rate
		return nil
	}
}

func WithTracesSampleRate(rate float64) Opt {
	return func(s *Sentry) error {
		s.conf.TracesSampleRate = rate
		return nil
	}
}

func WithEnableTracing(b bool) Opt {
	return func(s *Sentry) error {
		s.conf.EnableTracing = b
		return nil
	}
}

func WithTracesSampler(ts sentry.TracesSampler) Opt {
	return func(s *Sentry) error {
		s.conf.TracesSampler = ts
		return nil
	}
}
