package leaderelection

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

type Leader struct {
	clientSet  kubernetes.Interface
	podName    string
	leaderName string
	namespace  string
	ctx        context.Context
	cancel     context.CancelFunc
	fn         func(context.Context)
	opts       []Opt
}

func New(opts ...Opt) *Leader {
	return &Leader{
		opts:       opts,
		leaderName: "leader-election",
	}
}

func (l *Leader) Init() error {
	l.ctx, l.cancel = context.WithCancel(context.Background())
	for _, opt := range l.opts {
		if err := opt(l); err != nil {
			return fmt.Errorf("leaderelection.Leader Option error: %w", err)
		}
	}

	if l.fn == nil {
		return fmt.Errorf("WithFn is required")
	}

	slog.Info("pod name",
		slog.String("name", l.podName))

	return nil
}

func (l *Leader) Run() error {
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      l.leaderName,
			Namespace: l.namespace,
		},
		Client: l.clientSet.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: l.podName,
		},
	}

	leaderelection.RunOrDie(l.ctx, leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: 20 * time.Second,
		RenewDeadline: 15 * time.Second,
		RetryPeriod:   5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				l.fn(ctx)
			},
			OnStoppedLeading: func() {
				slog.Error("leader lost",
					slog.String("pod_name", l.podName))
			},
			OnNewLeader: func(identity string) {
				slog.Info("new leader elected",
					slog.String("identity", identity))
			},
		},
	})

	return nil
}

func (l *Leader) Stop() error {
	l.cancel()
	return nil
}

func (l *Leader) Name() string {
	return "leaderelection.Leader"
}

type Opt func(*Leader) error

func WithClientset(clientSet kubernetes.Interface) Opt {
	return func(l *Leader) error {
		l.clientSet = clientSet
		return nil
	}
}

func WithNamespace(namespace string) Opt {
	return func(l *Leader) error {
		l.namespace = namespace
		return nil
	}
}

func WithPodName(podName string) Opt {
	return func(l *Leader) error {
		l.podName = podName
		return nil
	}
}

func WithLeaderName(leaderName string) Opt {
	return func(l *Leader) error {
		l.leaderName = leaderName
		return nil
	}
}

func WithFn(fn func(context.Context)) Opt {
	return func(l *Leader) error {
		l.fn = fn
		return nil
	}
}
