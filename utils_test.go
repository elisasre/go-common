package common

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestMinUint(t *testing.T) {
	tests := []struct {
		inputA, inputB uint
		want           uint
	}{
		{inputA: 1, inputB: 2, want: 1},
		{inputA: 2, inputB: 1, want: 1},
		{inputA: 0, inputB: 1, want: 0},
		{inputA: 1, inputB: 0, want: 0},
	}
	for _, tc := range tests {
		result := MinUint(tc.inputA, tc.inputB)
		if result != tc.want {
			t.Errorf(
				"Expected %v < %v to be %v got %v", tc.inputA, tc.inputB, tc.want, result)
		}
	}
}

func TestEnsureDot(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "foo", want: "foo."},
		{input: "foo.", want: "foo."},
		{input: "", want: "."},
	}
	for _, tc := range tests {
		result := EnsureDot(tc.input)
		if result != tc.want {
			t.Errorf(
				"Expected %v got %v", tc.input, tc.want)
		}
	}
}

func TestRemoveDot(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "foo.", want: "foo"},
		{input: "foo..", want: "foo."},
		{input: ".", want: ""},
		{input: "..", want: "."},
	}
	for _, tc := range tests {
		result := RemoveDot(tc.input)
		if result != tc.want {
			t.Errorf(
				"Expected %v got %v", tc.input, tc.want)
		}
	}
}

type Config struct {
	Value int `yaml:"value"`
}

func TestLoadAndListenConfig_NonExistingFile(t *testing.T) {
	_, err := LoadAndListenConfig("invalid.yaml", &Config{}, nil)
	assert.ErrorContains(t, err, "no such file or directory")
}

func TestLoadAndListenConfig_InvalidSyntax(t *testing.T) {
	_, err := LoadAndListenConfig("testdata/invalid.yaml", &Config{}, nil)
	assert.ErrorContains(t, err, "invalid syntax")
}

func TestLoadAndListenConfigOnUpdate(t *testing.T) {
	firstConf := Config{Value: 1}
	filePath := "testdata/listen_config.yaml"
	f, err := os.Create(filePath)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, os.Remove(filePath)) })

	writeConf(t, firstConf, f)

	app := &TestApp{}
	notifyFn, waitForUpdate := updateCallbacks()
	c, err := LoadAndListenConfig(filePath, Config{}, func(newConf Config) {
		app.SetConf(newConf, notifyFn)
	})

	require.NoError(t, err)
	assert.Equal(t, firstConf, c)
	assert.Equal(t, 0, app.GetUpdateCalls())

	secondConf := Config{Value: 10}
	writeConf(t, secondConf, f)

	waitForUpdate(t)
	assert.Equal(t, secondConf, app.GetConf())
	assert.Equal(t, 1, app.GetUpdateCalls())

	thirdConf := Config{Value: 20}
	writeConf(t, thirdConf, f)

	waitForUpdate(t)
	assert.Equal(t, thirdConf, app.GetConf())
	assert.Equal(t, 2, app.GetUpdateCalls())
}

func writeConf(t *testing.T, c Config, f *os.File) {
	t.Helper()
	data, err := yaml.Marshal(&c)
	require.NoError(t, err)
	_, err = f.WriteAt(data, 0)
	require.NoError(t, err)
}

type TestApp struct {
	conf        Config
	updateCalls int
}

func (u *TestApp) SetConf(c Config, notifyFn func(Config)) {
	u.updateCalls++
	fmt.Println(c)
	u.conf = c
	notifyFn(c)
}

func (u *TestApp) GetUpdateCalls() int {
	return u.updateCalls
}

func (u *TestApp) GetConf() Config {
	return u.conf
}

func updateCallbacks() (func(Config), func(testing.TB)) {
	// Give some buffer for channel in case
	// viper decides to send multiple events.
	ch := make(chan struct{}, 10)
	notifier := func(Config) {
		ch <- struct{}{}
	}

	waitForUpdate := func(t testing.TB) {
		const onUpdateTimeout = time.Second * 2

		select {
		case <-ch:
		case <-time.After(onUpdateTimeout):
			t.Fatalf("OnUpdate not triggered with in %s", onUpdateTimeout)
		}
	}

	return notifier, waitForUpdate
}

func TestRecoverWithContext(t *testing.T) {
	tests := []struct {
		name  string
		cause string
		fn    func()
	}{
		{
			name:  "string panic",
			cause: "panic: test panic",
			fn:    func() { panic("test panic") },
		},
		{
			name:  "error panic",
			cause: "panic: error panic",
			fn:    func() { panic(fmt.Errorf("error panic")) },
		},
		{
			name:  "runtime error",
			cause: "panic: runtime error: index out of range",
			fn:    func() { _ = []int{}[1] },
		},
	}

	t.Cleanup(func() {
		output = os.Stdout
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			output = out

			func() {
				ctx := context.Background()
				span := sentry.SpanFromContext(ctx)
				defer RecoverWithContext(ctx, span)
				tt.fn()
			}()

			buf := out.String()
			hasPrefix := strings.HasPrefix(buf, tt.cause)
			require.True(t, hasPrefix, "expected %q to start with %q", buf, tt.cause)
		})
	}
}
