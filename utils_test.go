package common

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func TestLoadAndListenConfig(t *testing.T) {
	type Config struct {
		Index int `yaml:"index"`
	}
	filePath := "testdata/test.yaml"
	data, err := yaml.Marshal(&Config{})
	assert.NoError(t, err)
	err = os.WriteFile(filePath, data, 0o600)
	assert.NoError(t, err)

	realConf := &Config{}
	// should fail because file does not exists
	err = LoadAndListenConfig("invalid.yaml", realConf, nil)
	assert.ErrorContains(t, err, "no such file or directory")

	// should fail because file is not valid yaml
	err = LoadAndListenConfig("testdata/invalid.yaml", realConf, nil)
	assert.ErrorContains(t, err, "invalid syntax")

	err = LoadAndListenConfig(filePath, realConf, nil)
	assert.NoError(t, err)
	assert.Equal(t, realConf.Index, 0)

	data, err = yaml.Marshal(&Config{
		Index: 1,
	})
	assert.NoError(t, err)
	err = os.WriteFile(filePath, data, 0o600)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, realConf.Index, 1)
}

func TestLoadAndListenConfigOnUpdate(t *testing.T) {
	type Config struct {
		Index int `yaml:"index"`
	}
	filePath := "testdata/test2.yaml"
	data, err := yaml.Marshal(&Config{})
	assert.NoError(t, err)
	err = os.WriteFile(filePath, data, 0o600)
	assert.NoError(t, err)

	realConf := &Config{}
	var updateCalls int
	err = LoadAndListenConfig(filePath, realConf, func() {
		updateCalls += 1
	})
	assert.NoError(t, err)
	assert.Equal(t, realConf.Index, 0)
	assert.Equal(t, updateCalls, 0)

	data, err = yaml.Marshal(&Config{
		Index: 1,
	})
	assert.NoError(t, err)
	err = os.WriteFile(filePath, data, 0o600)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, realConf.Index, 1)
	assert.Equal(t, updateCalls, 1)
}
