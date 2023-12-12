package integrationtest

import (
	"context"

	"github.com/testcontainers/testcontainers-go"
	tc "github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Composer interface {
	ServiceContainer(ctx context.Context, svcName string) (*testcontainers.DockerContainer, error)
	Services() []string
	Down(ctx context.Context, opts ...tc.StackDownOption) error
	Up(ctx context.Context, opts ...tc.StackUpOption) (err error)
	WaitForService(s string, strategy wait.Strategy) tc.ComposeStack
	WithEnv(m map[string]string) tc.ComposeStack
	WithOsEnv() tc.ComposeStack
}

// ComposeOpt is option type for OptCompose.
type ComposeOpt func(*composeHandler)

// ComposeUpOptions set options for compose.Up().
func ComposeUpOptions(opts ...tc.StackUpOption) ComposeOpt {
	return func(c *composeHandler) { c.upOpts = append(c.upOpts, opts...) }
}

// ComposeDownOptions set options for compose.Down().
func ComposeDownOptions(opts ...tc.StackDownOption) ComposeOpt {
	return func(c *composeHandler) { c.downOpts = append(c.downOpts, opts...) }
}

// ComposeWaitForService makes compose up wait for specific service with given strategy.
func ComposeWaitForService(service string, strategy wait.Strategy) ComposeOpt {
	return func(c *composeHandler) { c.c.WaitForService(service, strategy) }
}

// ComposeEnv set environment variables for compose.
func ComposeEnv(env map[string]string) ComposeOpt {
	return func(c *composeHandler) { c.c.WithEnv(env) }
}

// ComposeOsEnv passes environment from OS to compose.
func ComposeOsEnv() ComposeOpt {
	return func(c *composeHandler) { c.c.WithOsEnv() }
}

// ComposeComposer allows getting access to compose instance which is usable after Init() is called.
func ComposeComposer(compose *Compose) ComposeOpt {
	return func(c *composeHandler) { compose.Composer = c.c }
}

type Compose struct {
	Composer
}

type composeHandler struct {
	c        Composer
	upOpts   []tc.StackUpOption
	downOpts []tc.StackDownOption
}

func (c *composeHandler) Run() error {
	return c.c.Up(context.Background(), c.upOpts...)
}

func (c *composeHandler) Stop() error {
	return c.c.Down(context.Background(), c.downOpts...)
}
