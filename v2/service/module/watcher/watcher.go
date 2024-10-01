// Package watcher provides file update notification functionality as a module.
package watcher

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
)

var (
	ErrMissingWithTarget = fmt.Errorf("watcher.Watcher missing or empty WithTarget option")
	ErrMissingWithFunc   = fmt.Errorf("watcher.Watcher missing WithFunc option")
)

type Watcher struct {
	w      *fsnotify.Watcher
	target string
	fn     func() error
	opts   []Opt
}

// New creates watcher with given options.
// WithTarget and WithFunc options are mandatory.
func New(opts ...Opt) *Watcher {
	return &Watcher{opts: opts}
}

func (w *Watcher) Init() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("watcher.Watcher error: %w", err)
	}
	w.w = watcher

	for _, opt := range w.opts {
		if err := opt(w); err != nil {
			return fmt.Errorf("watcher.Watcher Option error: %w", err)
		}
	}

	switch {
	case w.target == "":
		return ErrMissingWithTarget
	case w.fn == nil:
		return ErrMissingWithFunc
	}

	return w.w.Add(w.target)
}

func (w *Watcher) Run() error {
	for {
		select {
		case event, ok := <-w.w.Events:
			if !ok {
				return nil
			}
			if event.Op.Has(fsnotify.Write) || event.Op.Has(fsnotify.Create) {
				err := w.fn()
				if err != nil {
					return err
				}
			}
		case err, ok := <-w.w.Errors:
			if !ok {
				return nil
			}
			return fmt.Errorf("watcher.Watcher error: %w", err)
		}
	}
}

func (w *Watcher) Stop() error {
	return w.w.Close()
}

func (w *Watcher) Name() string {
	return "watcher.Watcher"
}

type Opt func(*Watcher) error

func WithTarget(fileOrDirName string) Opt {
	return func(w *Watcher) error {
		w.target = fileOrDirName
		return nil
	}
}

func WithFunc(fn func() error) Opt {
	return func(w *Watcher) error {
		w.fn = fn
		return nil
	}
}
