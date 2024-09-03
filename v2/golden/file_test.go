package golden

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type test struct {
	name         string
	mt           mockT
	data         string
	expectedData string
	expectedPath string
}

func TestFileString(t *testing.T) {
	t.Cleanup(func() { assert.NoError(t, os.RemoveAll("./testdata"), "failed to remove testdata") })
	tests := []test{
		{
			name:         "plain test func",
			mt:           mockT{name: "TestPlainFunc"},
			data:         "some data",
			expectedData: "some data",
			expectedPath: "./testdata/TestPlainFunc/TestPlainFunc.golden",
		},
		{
			name:         "sub test",
			mt:           mockT{name: "TestFunc/subtest"},
			data:         "other data",
			expectedData: "other data",
			expectedPath: "./testdata/TestFunc/subtest.golden",
		},
		{
			name:         "second sub test",
			mt:           mockT{name: "TestFunc/subtest_other"},
			data:         "yet another data",
			expectedData: "yet another data",
			expectedPath: "./testdata/TestFunc/subtest_other.golden",
		},
		{
			name:         "nested sub test",
			mt:           mockT{name: "TestFunc/subtest/nested"},
			data:         "nested data",
			expectedData: "nested data",
			expectedPath: "./testdata/TestFunc/subtest_nested.golden",
		},
		{
			name:         "parent of sub test",
			mt:           mockT{name: "TestFunc"},
			data:         "parent data",
			expectedData: "parent data",
			expectedPath: "./testdata/TestFunc/TestFunc.golden",
		},
	}

	t.Run("create", func(t *testing.T) {
		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				mt := &tt.mt
				got := string(file(mt, []byte(tt.data), true))
				assertResult(t, tt, mt, got)
			})
		}
	})

	t.Run("read only", func(t *testing.T) {
		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				mt := &tt.mt
				got := FileString(mt, []byte(tt.data))
				assertResult(t, tt, mt, got)
			})
		}
	})

	const suffix = " override"

	t.Run("override", func(t *testing.T) {
		for _, tt := range tests {
			tt := tt
			tt.data += suffix
			tt.expectedData += suffix
			t.Run(tt.name, func(t *testing.T) {
				mt := &tt.mt
				got := string(file(mt, []byte(tt.data), true))
				assertResult(t, tt, mt, got)
			})
		}
	})

	t.Run("read only after override", func(t *testing.T) {
		for _, tt := range tests {
			tt := tt
			tt.data += suffix
			tt.expectedData += suffix
			t.Run(tt.name, func(t *testing.T) {
				mt := &tt.mt
				got := FileString(mt, []byte(tt.data))
				assertResult(t, tt, mt, got)
			})
		}
	})
}

func TestFolderDoesNotExist(t *testing.T) {
	mt := mockT{name: "TestDirFail"}
	got := File(&mt, []byte("data"))
	assert.Empty(t, got)
	assert.True(t, mt.failed)
	assert.Contains(t, mt.msg, "open ./testdata/TestDirFail/TestDirFail.golden: no such file or directory")
	assert.NoDirExists(t, "./testdata/TestDirFail")
}

func TestEqual(t *testing.T) {
	t.Cleanup(func() { assert.NoError(t, os.RemoveAll("./testdata"), "failed to remove testdata") })

	data := []byte("some data")
	assert.NoError(t, os.MkdirAll("./testdata/TestSomeBytes", 0o755))
	assert.NoError(t, os.WriteFile("./testdata/TestSomeBytes/TestSomeBytes.golden", data, 0o600))

	mt := mockT{name: "TestSomeBytes"}
	got := Equal(&mt, data)
	assert.True(t, got)
	assert.Empty(t, mt.msg)
	assert.False(t, mt.failed)
}

func TestEqualString(t *testing.T) {
	t.Cleanup(func() { assert.NoError(t, os.RemoveAll("./testdata"), "failed to remove testdata") })

	data := []byte("some string")
	assert.NoError(t, os.MkdirAll("./testdata/TestSomeString", 0o755))
	assert.NoError(t, os.WriteFile("./testdata/TestSomeString/TestSomeString.golden", data, 0o600))

	mt := mockT{name: "TestSomeString"}
	got := EqualString(&mt, data)
	assert.True(t, got)
	assert.Empty(t, mt.msg)
	assert.False(t, mt.failed)
}

func TestEqualS_No_Match(t *testing.T) {
	t.Cleanup(func() { assert.NoError(t, os.RemoveAll("./testdata"), "failed to remove testdata") })

	data := []byte("some string")
	assert.NoError(t, os.MkdirAll("./testdata/TestSomeString", 0o755))
	assert.NoError(t, os.WriteFile("./testdata/TestSomeString/TestSomeString.golden", data, 0o600))

	mt := mockT{name: "TestSomeString"}
	got := EqualString(&mt, []byte("other string"))
	assert.False(t, got)
	assert.Contains(t, mt.msg, "Not equal:")
	assert.False(t, mt.failed) // In case of assert failure, FailNow is not called
}

func assertResult(t *testing.T, tt test, mt *mockT, got string) {
	t.Helper()
	assert.Equal(t, tt.expectedData, got)
	assert.Empty(t, mt.msg)
	assert.False(t, mt.failed)
	assert.FileExists(t, tt.expectedPath)
	b, err := os.ReadFile(tt.expectedPath)
	require.NoError(t, err)
	assert.Equal(t, tt.expectedData, string(b))
}

type mockT struct {
	name   string
	failed bool
	msg    string
}

func (m *mockT) Name() string                         { return m.name }
func (m *mockT) Errorf(f string, args ...interface{}) { m.msg = fmt.Sprintf(f, args...) }
func (m *mockT) FailNow()                             { m.failed = true }
