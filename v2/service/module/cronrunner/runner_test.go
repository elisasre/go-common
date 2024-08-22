package cronrunner_test

import (
	"testing"

	"github.com/elisasre/go-common/v2/service/module/cronrunner"
	"github.com/hashicorp/go-multierror"
	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	const everySecond = "@every 1s"
	called := make(chan struct{})
	c := cron.New(cron.WithSeconds())

	fn := func() {
		entries := c.Entries()
		for _, entry := range entries {
			c.Remove(entry.ID)
		}
		close(called)
	}

	runner := cronrunner.New(
		cronrunner.WithCron(c),
		cronrunner.WithFunc(everySecond, fn),
	)

	require.NoError(t, runner.Init())
	wg := &multierror.Group{}
	wg.Go(runner.Run)

	<-called

	require.NoError(t, runner.Stop())
	require.NoError(t, wg.Wait().ErrorOrNil())
	require.Equal(t, "cron.Runner", runner.Name())
}

func TestNew_ErrNilCronAfterInit(t *testing.T) {
	runner := cronrunner.New()
	require.ErrorIs(t, runner.Init(), cronrunner.ErrNilCronAfterInit)
}

func TestNew_ErrAddFuncToNilCron(t *testing.T) {
	runner := cronrunner.New(cronrunner.WithFunc("@every 1s", func() {}))
	require.ErrorIs(t, runner.Init(), cronrunner.ErrAddFuncToNilCron)
}
