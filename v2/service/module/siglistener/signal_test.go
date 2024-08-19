package siglistener_test

import (
	"os"
	"syscall"
	"testing"

	"github.com/elisasre/go-common/v2/service/module/siglistener"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/require"
)

func TestListener(t *testing.T) {
	l := siglistener.New(syscall.SIGINT)
	require.NoError(t, l.Init())
	require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGINT))
	require.NoError(t, l.Run())
	require.NoError(t, l.Stop())
	require.NotEmpty(t, l.Name())
}

func TestStop(t *testing.T) {
	l := siglistener.New(os.Interrupt)
	require.NoError(t, l.Init())

	wg := &multierror.Group{}
	wg.Go(l.Run)
	require.NoError(t, l.Stop())
	require.NoError(t, wg.Wait().ErrorOrNil())
}
