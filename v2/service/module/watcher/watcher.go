// Package watcher provides file update notification functionality as a module.
package watcher

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
)

var (
	ErrMissingWithFilename = fmt.Errorf("watcher.Watcher missing WithFilename option or empty filename")
	ErrMissingWithFunc     = fmt.Errorf("watcher.Watcher missing WithFunc option")
)

type Watcher struct {
	w        *fsnotify.Watcher
	filename string
	fn       func() error
	opts     []Opt
}

// New creates watcher with given options.
// WithFilename and WithFunc options are mandatory.
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
	case w.filename == "":
		return ErrMissingWithFilename
	case w.fn == nil:
		return ErrMissingWithFunc
	}

	return w.w.Add(w.filename)
}

func (w *Watcher) Run() error {
	for {
		select {
		case event, ok := <-w.w.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
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

func WithFilename(filename string) Opt {
	return func(w *Watcher) error {
		w.filename = filename
		return nil
	}
}

func WithFunc(fn func() error) Opt {
	return func(w *Watcher) error {
		w.fn = fn
		return nil
	}
}
