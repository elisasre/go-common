package sentry_test

import (
	"errors"
	"testing"

	"github.com/elisasre/go-common/service/module/sentry"
	sentrygo "github.com/getsentry/sentry-go"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/require"
)

func TestSentry(t *testing.T) {
	s := sentry.New(
		sentry.WithClientOptions(sentrygo.ClientOptions{}),
		sentry.WithAttachStacktrace(true),
		sentry.WithEnableTracing(true),
		sentry.WithSampleRate(0),
		sentry.WithTracesSampleRate(0),
		sentry.WithTracesSampler(sentrygo.TracesSampler(func(ctx sentrygo.SamplingContext) float64 { return 0 })),
	)
	require.NoError(t, s.Init())
	wg := &multierror.Group{}
	wg.Go(s.Run)
	require.NoError(t, s.Stop())
	require.NoError(t, wg.Wait().ErrorOrNil())
	require.Equal(t, "sentry.Sentry", s.Name())
}

func TestSentryInvalidDSN(t *testing.T) {
	s := sentry.New(sentry.WithDSN("asd"))
	require.Error(t, s.Init())
}

func TestSentryInvalidDSN_WithIgnoreInitErr(t *testing.T) {
	s := sentry.New(
		sentry.WithDSN("asd"),
		sentry.WithIgnoreInitErr(),
	)

	require.NoError(t, s.Init())
	wg := &multierror.Group{}
	wg.Go(s.Run)
	require.NoError(t, s.Stop())
	require.NoError(t, wg.Wait().ErrorOrNil())
	require.Equal(t, "sentry.Sentry", s.Name())
}

func TestSentry_OptErr(t *testing.T) {
	errOpt := errors.New("opt err")
	s := sentry.New(func(s *sentry.Sentry) error {
		return errOpt
	})

	err := s.Init()
	require.ErrorIs(t, err, errOpt)
}
