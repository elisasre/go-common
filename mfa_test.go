package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotify(t *testing.T) {
	assert.Equal(t, prefix0("123"), "000123", "fail")
	assert.Equal(t, prefix0("12223"), "012223", "fail")
	assert.Equal(t, prefix0("123123"), "123123", "fail")
	assert.Equal(t, prefix0("1"), "000001", "fail")
}
