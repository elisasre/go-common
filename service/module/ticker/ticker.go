// Package ticker provides ticker functionality as a module.
package ticker

import (
	"fmt"
	"time"
)

var (
	ErrMissingWithInterval = fmt.Errorf("ticker.Ticker missing WithInterval option")
	ErrMissingWithFunc     = fmt.Errorf("ticker.Ticker missing WithFunc option")
)

type Ticker struct {
	t       *time.Ticker
	stopped chan struct{}
	fn      func() error
	opts    []Opt
}

// New creates ticker with given options.
// WithInterval and WithFunc options are mandatory.
func New(opts ...Opt) *Ticker {
	return &Ticker{opts: opts}
}

func (t *Ticker) Init() error {
	t.stopped = make(chan struct{})
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
			if err := t.fn(); err != nil {
				return err
			}
		case <-t.stopped:
			return nil
		}
	}
}

func (t *Ticker) Stop() error {
	defer close(t.stopped)
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
		t.fn = fn
		return nil
	}
}
