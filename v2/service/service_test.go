package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/elisasre/go-common/v2/service"
	"github.com/stretchr/testify/require"
)

var (
	errInit = errors.New("init error")
	errRun  = errors.New("run error")
	errStop = errors.New("stop error")
)

func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		mods        service.Modules
		expectedErr error
	}{
		{
			name:        "NoError",
			mods:        nil,
			expectedErr: nil,
		},
		{
			name:        "NoError_MultipleModules",
			mods:        []service.Module{SuccessMod(), SuccessMod(), SuccessMod()},
			expectedErr: nil,
		},
		{
			name:        "InitError",
			mods:        []service.Module{InitErrMod()},
			expectedErr: errInit,
		},
		{
			name:        "RunError",
			mods:        []service.Module{RunErrMod()},
			expectedErr: errRun,
		},
		{
			name:        "StopError",
			mods:        []service.Module{StopErrMod()},
			expectedErr: errStop,
		},
		{
			name:        "InitPanic",
			mods:        []service.Module{InitPanicMod()},
			expectedErr: service.ErrPanic,
		},
		{
			name:        "RunPanic",
			mods:        []service.Module{RunPanicMod()},
			expectedErr: service.ErrPanic,
		},
		{
			name:        "StopPanic",
			mods:        []service.Module{StopPanicMod()},
			expectedErr: service.ErrPanic,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			stopMod, stop := StopMod()
			tc.mods = append(tc.mods, stopMod)

			go func() {
				time.Sleep(time.Second)
				stop()
			}()

			err := service.Run(tc.mods)
			require.ErrorIs(t, err, tc.expectedErr)
		})
	}
}

func TestRunAndExit(t *testing.T) {
	stopMod, stop := StopMod()
	go func() {
		time.Sleep(time.Second)
		stop()
	}()

	service.RunAndExit(service.Modules{stopMod})
}

// StopMod can be used to trigger stop sequence for service.Run in tests.
func StopMod() (service.Module, func()) {
	ctx, stop := context.WithCancel(context.Background())
	return &TestMod{
		init: func() error {
			return nil
		},
		run: func() error {
			<-ctx.Done()
			return nil
		},
		stop: func() error {
			stop()
			return nil
		},
	}, stop
}

func InitErrMod() service.Module {
	return &TestMod{
		init: func() error { return errInit },
		run:  func() error { return nil },
		stop: func() error { return nil },
	}
}

func InitPanicMod() service.Module {
	return &TestMod{
		run:  func() error { return nil },
		stop: func() error { return nil },
	}
}

func RunErrMod() service.Module {
	return &TestMod{
		init: func() error { return nil },
		run:  func() error { return errRun },
		stop: func() error { return nil },
	}
}

func RunPanicMod() service.Module {
	return &TestMod{
		init: func() error { return nil },
		stop: func() error { return nil },
	}
}

func StopErrMod() service.Module {
	return &TestMod{
		init: func() error { return nil },
		run:  func() error { return nil },
		stop: func() error { return errStop },
	}
}

func StopPanicMod() service.Module {
	return &TestMod{
		init: func() error { return nil },
		run:  func() error { return nil },
	}
}

func SuccessMod() service.Module {
	mod, _ := StopMod()
	return mod
}

type TestMod struct{ init, run, stop func() error }

func (m *TestMod) Name() string { return "TestMod" }
func (m *TestMod) Init() error  { return m.init() }
func (m *TestMod) Run() error   { return m.run() }
func (m *TestMod) Stop() error  { return m.stop() }
