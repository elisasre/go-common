package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunAndExit(t *testing.T) {
	var called bool
	exitFn = func(code int) {
		called = true
		require.Equal(t, code, 1, "wrong exit code")
	}

	RunAndExit(Modules{&TestMod{}})
	require.True(t, called)
}

type TestMod struct{}

func (m *TestMod) Name() string { return "TestMod" }
func (m *TestMod) Init() error  { return errors.New("init err") }
func (m *TestMod) Run() error   { return nil }
func (m *TestMod) Stop() error  { return nil }
