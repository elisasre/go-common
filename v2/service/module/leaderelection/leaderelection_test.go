package leaderelection_test

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/elisasre/go-common/v2/service/module/leaderelection"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func TestLeaderElection(t *testing.T) {
	count := &testCount{}
	leader := leaderelection.New(
		leaderelection.WithLeaderName("leader-name"),
		leaderelection.WithClientset(fake.NewSimpleClientset()),
		leaderelection.WithNamespace("foo"),
		leaderelection.WithPodName("bar"),
		leaderelection.WithFn(func(ctx context.Context) {
			slog.Info("leader loop executing")
			count.inc()
		}),
	)

	require.NoError(t, leader.Init())
	wg := &multierror.Group{}
	wg.Go(leader.Run)
	time.Sleep(500 * time.Millisecond)
	require.NoError(t, leader.Stop())
	require.NoError(t, wg.Wait().ErrorOrNil())
	require.Equal(t, 1, count.get())
}

type testCount struct {
	sync.Mutex
	count int
}

func (t *testCount) inc() {
	t.Lock()
	defer t.Unlock()
	t.count++
}

func (t *testCount) get() int {
	t.Lock()
	defer t.Unlock()
	return t.count
}
