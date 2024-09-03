// Package golden provides standard way to write tests with golden files.
package golden

import (
	"fmt"
	"os"
	"strings"

	"github.com/elisasre/go-common/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var overrideTestData = common.StringToBool(os.Getenv("OVERRIDE_TEST_DATA"))

type T interface {
	Errorf(format string, args ...interface{})
	FailNow()
	Name() string
}

// Equal asserts that the golden file content is equal to the data.
func Equal(t T, data []byte) bool {
	return assert.Equal(t, File(t, data), data)
}

// EqualString asserts that the golden file content is equal to the data in string format.
func EqualString(t T, data []byte) bool {
	return assert.Equal(t, FileString(t, data), string(data))
}

// FileString returns the output of golden.File as a string.
func FileString(t T, data []byte) string { return string(File(t, data)) }

// File returns the golden file content for the test.
// If OVERRIDE_TEST_DATA env is set to true, the golden file will be created with the content of the data.
// OVERRIDE_TEST_DATA is read only once at the start of the test and it's value is not updated.
// Depending of the test structure the golden file and it's directories arew created in
// ./testdata/{testFuncName}/{subTestName}.golden or ./testdata/{testFuncName}/{testFuncName}.golden.
func File(t T, data []byte) []byte { return file(t, data, overrideTestData) }

func file(t T, data []byte, override bool) []byte {
	split := strings.SplitN(t.Name(), "/", 2)
	mainTestName := t.Name()
	testName := t.Name()
	if len(split) == 2 {
		mainTestName = split[0]
		testName = strings.ReplaceAll(split[1], "/", "_")
	}

	folderName := fmt.Sprintf("./testdata/%s", mainTestName)
	fileName := strings.ReplaceAll(fmt.Sprintf("%s/%s.golden", folderName, testName), " ", "_")
	if override {
		require.NoError(t, os.MkdirAll(folderName, 0o755))
		require.NoError(t, os.WriteFile(fileName, data, 0o600))
	}

	b, err := os.ReadFile(fileName)
	require.NoError(t, err)
	return b
}
