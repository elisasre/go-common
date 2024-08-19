// Package ticker provides ticker functionality as a module.
package ticker_test

import (
	"errors"
	"testing"
	"time"

	"github.com/elisasre/go-common/v2/service/module/ticker"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/require"
)

func TestListener(t *testing.T) {
	called := make(chan struct{})
	tickerMod := ticker.New(
		ticker.WithInterval(time.Millisecond*10),
		ticker.WithFunc(func() error {
			select {
			case called <- struct{}{}:
			default:
			}
			return nil
		}),
	)

	require.NoError(t, tickerMod.Init())
	wg := &multierror.Group{}
	wg.Go(tickerMod.Run)
	<-called
	require.NoError(t, tickerMod.Stop())
	require.NoError(t, wg.Wait().ErrorOrNil())
	require.Equal(t, "ticker.Ticker", tickerMod.Name())
}

func TestListenerRunError(t *testing.T) {
	errRun := errors.New("run error")
	tickerMod := ticker.New(
		ticker.WithInterval(time.Millisecond*10),
		ticker.WithFunc(func() error { return errRun }),
	)

	require.NoError(t, tickerMod.Init())
	require.ErrorIs(t, tickerMod.Run(), errRun)
	require.NoError(t, tickerMod.Stop())
}

func TestListenerInitErrors(t *testing.T) {
	errOpt := errors.New("opt error")

	tests := []struct {
		name        string
		ticker      *ticker.Ticker
		expectedErr error
	}{
		{
			name:        "ErrOpt",
			ticker:      ticker.New(func(t *ticker.Ticker) error { return errOpt }),
			expectedErr: errOpt,
		},
		{
			name:        "ErrMissingWithFunc",
			ticker:      ticker.New(ticker.WithInterval(time.Second)),
			expectedErr: ticker.ErrMissingWithFunc,
		},
		{
			name:        "ErrMissingWithInterval",
			ticker:      ticker.New(ticker.WithFunc(func() error { return nil })),
			expectedErr: ticker.ErrMissingWithInterval,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.ticker.Init()
			require.ErrorIs(t, err, tc.expectedErr)
		})
	}
}
