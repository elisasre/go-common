// Package watcher provides file update notification functionality as a module.
package watcher_test

import (
	"errors"
	"os"
	"testing"

	"github.com/elisasre/go-common/v2/service/module/watcher"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/require"
)

var testFilePattern = "watcher-test-*"

func TestListener(t *testing.T) {
	tmpFile, err := os.CreateTemp("", testFilePattern)
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	called := make(chan struct{}, 100)
	watcherMod := watcher.New(
		watcher.WithFilename(tmpFile.Name()),
		watcher.WithFunc(func() error {
			called <- struct{}{}
			return nil
		}),
	)

	require.NoError(t, watcherMod.Init())
	wg := &multierror.Group{}
	wg.Go(watcherMod.Run)
	_, err = tmpFile.Write([]byte("updated"))
	require.NoError(t, err)
	<-called
	require.NoError(t, watcherMod.Stop())
	require.NoError(t, wg.Wait().ErrorOrNil())
	require.Equal(t, "watcher.Watcher", watcherMod.Name())
}

func TestListenerRunError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", testFilePattern)
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	errRun := errors.New("run error")
	watcherMod := watcher.New(
		watcher.WithFilename(tmpFile.Name()),
		watcher.WithFunc(func() error { return errRun }),
	)

	require.NoError(t, watcherMod.Init())
	wg := &multierror.Group{}
	wg.Go(watcherMod.Run)
	_, err = tmpFile.Write([]byte("updated"))
	require.NoError(t, err)
	require.ErrorIs(t, wg.Wait().ErrorOrNil(), errRun)
	require.NoError(t, watcherMod.Stop())
}

func TestListenerInitErrors(t *testing.T) {
	errOpt := errors.New("opt error")

	tests := []struct {
		name        string
		watcher     *watcher.Watcher
		expectedErr error
	}{
		{
			name:        "ErrOpt",
			watcher:     watcher.New(func(t *watcher.Watcher) error { return errOpt }),
			expectedErr: errOpt,
		},
		{
			name:        "ErrMissingWithFunc",
			watcher:     watcher.New(watcher.WithFilename("something")),
			expectedErr: watcher.ErrMissingWithFunc,
		},
		{
			name:        "ErrMissingWithFilename",
			watcher:     watcher.New(watcher.WithFunc(func() error { return nil })),
			expectedErr: watcher.ErrMissingWithFilename,
		},
		{
			name:        "ErrEmptyFilename",
			watcher:     watcher.New(watcher.WithFilename(""), watcher.WithFunc(func() error { return nil })),
			expectedErr: watcher.ErrMissingWithFilename,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.watcher.Init()
			require.ErrorIs(t, err, tc.expectedErr)
		})
	}
}