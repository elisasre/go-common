// Package service provides simple service framework on top of Module interface.
// Ready made modules can be found under: github.com/elisasre/go-common/v2/service/module.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/hashicorp/go-multierror"
)

var exitFn = os.Exit

// Service is a container for modules.
type Service interface {
	Modules() []Module
}

// Modules is a convenience type for modules to avoid implementing Service when not necessary.
type Modules []Module

func (m Modules) Modules() []Module { return m }

type Module interface {
	Name() string
	Init() error
	Run() error
	Stop() error
}

// Run runs svc using following control flow:
//
//  1. Exec Init() for each module in order.
//     If error is occurred Run returns immediately.
//  2. Exec Run() for each module in own goroutine.
//  3. Wait for any Run() function to return.
//     When that happens move to Stop sequence.
//  4. Exec Stop() for modules in reverse order.
//  5. Wait for all Run() and Stop() calls to return.
//  6. Return all errors or nil
//
// Possible panics inside modules are captured to allow graceful shutdown of other modules.
// Captured panics are converted into errors and ErrPanic is returned.
func Run(svc Service) error {
	r := &runner{
		modules: svc.Modules(),
	}

	slog.Info("starting service")
	if err := r.run(); err != nil {
		slog.Error("service exited with error", slog.Any("error", err))
		return err
	}

	slog.Info("service stopped successfully")
	return nil
}

func RunAndExit(svc Service) {
	if err := Run(svc); err != nil {
		exitFn(1)
	}
}

type runner struct {
	modules []Module
}

func (r *runner) run() error {
	slog.Info("initializing modules")
	for _, mod := range r.modules {
		slog.Info("module initializing", slog.String("name", mod.Name()))
		err := catchPanic(mod.Init)
		if err != nil {
			return fmt.Errorf("failed to initialize module %s: %w", mod.Name(), err)
		}
		slog.Info("module initialized", slog.String("name", mod.Name()))
	}
	slog.Info("all modules initialized successfully")

	slog.Info("starting modules")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &multierror.Group{}
	for _, mod := range r.modules {
		wg.Go(func() error {
			defer func() {
				slog.Info("module run exited", slog.String("name", mod.Name()))
				cancel()
			}()

			slog.Info("module started", slog.String("name", mod.Name()))
			err := catchPanic(mod.Run)
			if err != nil {
				return fmt.Errorf("failed to run module %s: %w", mod.Name(), err)
			}

			return nil
		})
	}

	<-ctx.Done()

	slog.Info("stopping modules")
	for i := len(r.modules) - 1; i >= 0; i-- {
		mod := r.modules[i]
		closed := make(chan struct{})
		wg.Go(func() error {
			defer func() {
				close(closed)
				slog.Info("module stopped", slog.String("name", mod.Name()))
			}()

			slog.Info("module stopping", slog.String("name", mod.Name()))
			return catchPanic(mod.Stop)
		})
		<-closed
	}

	return wg.Wait().ErrorOrNil()
}

var ErrPanic = errors.New("recovered from panic")

func catchPanic(fn func() error) (err error) {
	defer func() {
		if rErr := recover(); rErr != nil {
			// Print stack trace to log without logger to preserver proper multiline formatting.
			fmt.Println(string(debug.Stack()))
			err = fmt.Errorf("%w: %s", ErrPanic, rErr)
		}
	}()
	return fn()
}
