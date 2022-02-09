package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBase64decode(t *testing.T) {
	encoded := "U1VDQ0VTUw=="
	decoded, err := Base64decode(encoded)
	assert.Nil(t, err)
	assert.Equal(t, "SUCCESS", decoded)

	failing := "^"
	_, err = Base64decode(failing)
	assert.NotNil(t, err)
}
