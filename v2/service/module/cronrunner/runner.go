// Package cronrunner provides module wrapper for github.com/robfig/cron/v3.
package cronrunner

import (
	"fmt"

	"github.com/robfig/cron/v3"
)

var (
	ErrNilCronAfterInit = fmt.Errorf("cron.Cron was nil, WithCron is required option")
	ErrAddFuncToNilCron = fmt.Errorf("WithCron has to be applied before WithFunc")
)

// Runner is a wrapper around cron.Cron implementing service.Module interface.
type Runner struct {
	cron *cron.Cron
	opts []Opt
}

type Opt func(r *Runner) error

// New creates Runner with given options.
// Options are applied in same order as they were provided.
// WithCron is required option.
func New(opts ...Opt) *Runner {
	return &Runner{
		opts: opts,
	}
}

// Init initializes Runner with given options.
func (r *Runner) Init() error {
	for _, opt := range r.opts {
		if err := opt(r); err != nil {
			return fmt.Errorf("cron.Runner Option error: %w", err)
		}
	}
	if r.cron == nil {
		return ErrNilCronAfterInit
	}
	return nil
}

// Run starts cron job runner.
func (r *Runner) Run() error {
	r.cron.Run()
	return nil
}

// Stop stops cron job runner.
func (r *Runner) Stop() error {
	ctx := r.cron.Stop()
	<-ctx.Done()
	return nil
}

func (r *Runner) Name() string {
	return "cron.Runner"
}

// WithCron sets cron.Cron instance.
func WithCron(c *cron.Cron) Opt {
	return func(r *Runner) error {
		r.cron = c
		return nil
	}
}

// WithFunc adds given functions to cron runner with given spec.
func WithFunc(spec string, fn func()) Opt {
	return func(r *Runner) error {
		if r.cron == nil {
			return ErrAddFuncToNilCron
		}

		_, err := r.cron.AddFunc(spec, fn)
		return err
	}
}
