// Package ticker provides ticker functionality as a module.
package ticker

import (
	"context"
	"fmt"
	"time"
)

var (
	ErrMissingWithInterval = fmt.Errorf("ticker.Ticker missing WithInterval option")
	ErrMissingWithFunc     = fmt.Errorf("ticker.Ticker missing WithFunc option")
)

type Ticker struct {
	t      *time.Ticker
	cancel func()
	// required to avoid concurrency issues, only used privately
	ctx  context.Context //nolint: containedctx
	fn   func(ctx context.Context) error
	opts []Opt
}

// New creates ticker with given options.
// WithInterval and WithFunc options are mandatory.
func New(opts ...Opt) *Ticker {
	return &Ticker{opts: opts, cancel: func() {}}
}

func (t *Ticker) Init() error {
	t.ctx, t.cancel = context.WithCancel(context.Background())

	for _, opt := range t.opts {
		if err := opt(t); err != nil {
			return fmt.Errorf("ticker.Ticker Option error: %w", err)
		}
	}

	switch {
	case t.t == nil:
		return ErrMissingWithInterval
	case t.fn == nil:
		return ErrMissingWithFunc
	}

	return nil
}

func (t *Ticker) Run() error {
	for {
		select {
		case <-t.t.C:
			if err := t.fn(t.ctx); err != nil {
				return err
			}
		case <-t.ctx.Done():
			return nil
		}
	}
}

func (t *Ticker) Stop() error {
	t.cancel()
	t.t.Stop()
	return nil
}

func (t *Ticker) Name() string {
	return "ticker.Ticker"
}

type Opt func(*Ticker) error

func WithInterval(d time.Duration) Opt {
	return func(t *Ticker) error {
		t.t = time.NewTicker(d)
		return nil
	}
}

func WithFunc(fn func() error) Opt {
	return func(t *Ticker) error {
		t.fn = func(context.Context) error {
			return fn()
		}
		return nil
	}
}

func WithFuncContext(fn func(context.Context) error) Opt {
	return func(t *Ticker) error {
		t.fn = fn
		return nil
	}
}
