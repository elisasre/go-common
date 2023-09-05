package common

import (
	"os"
	"sync"
	"testing"
	"time"

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
	Index int `yaml:"index"`
}

func TestLoadAndListenConfig_NonExistingFile(t *testing.T) {
	err := LoadAndListenConfig("invalid.yaml", &Config{}, nil)
	assert.ErrorContains(t, err, "no such file or directory")
}

func TestLoadAndListenConfig_InvalidSyntax(t *testing.T) {
	err := LoadAndListenConfig("testdata/invalid.yaml", &Config{}, nil)
	assert.ErrorContains(t, err, "invalid syntax")
}

func (u *UpdateValues) Set(updateCalls int, oldConf interface{}, notifyFn func(interface{})) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.UpdateCalls += updateCalls
	u.OldValue = oldConf.(Config).Index
	notifyFn(oldConf)
}

func (u *UpdateValues) GetUpdateCalls() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.UpdateCalls
}

func (u *UpdateValues) GetOldValue() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.OldValue
}

type UpdateValues struct {
	mu          sync.Mutex
	UpdateCalls int
	OldValue    int
}

func TestLoadAndListenConfigOnUpdate(t *testing.T) {
	filePath := "testdata/test2.yaml"
	data, err := yaml.Marshal(&Config{})
	require.NoError(t, err)
	err = os.WriteFile(filePath, data, 0o600)
	require.NoError(t, err)

	realConf := &Config{}
	values := &UpdateValues{}
	notifyFn, waitForUpdate := updateCallbacks()
	err = LoadAndListenConfig(filePath, realConf, func(oldConf interface{}) {
		values.Set(1, oldConf, notifyFn)
	})
	require.NoError(t, err)
	assert.Equal(t, 0, realConf.Index)
	assert.Equal(t, 0, values.GetOldValue())
	assert.Equal(t, 0, values.GetUpdateCalls())

	data, err = yaml.Marshal(&Config{
		Index: 1,
	})
	require.NoError(t, err)
	err = os.WriteFile(filePath, data, 0o600)
	require.NoError(t, err)

	waitForUpdate(t)
	assert.Equal(t, 1, realConf.Index)
	assert.Equal(t, 0, values.GetOldValue())
	assert.Equal(t, 1, values.GetUpdateCalls())

	data, err = yaml.Marshal(&Config{
		Index: 2,
	})
	require.NoError(t, err)
	err = os.WriteFile(filePath, data, 0o600)
	require.NoError(t, err)

	waitForUpdate(t)
	assert.Equal(t, 2, realConf.Index)
	assert.Equal(t, 1, values.GetOldValue())
	assert.Equal(t, 2, values.GetUpdateCalls())
}

func updateCallbacks() (func(interface{}), func(testing.TB)) {
	// Give some buffer for channel in case
	// viper decides to send multiple events.
	ch := make(chan struct{}, 10)
	notifier := func(interface{}) {
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
