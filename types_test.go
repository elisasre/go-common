package common

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	testliteral := "literal"
	var i interface{} = String(testliteral)
	assert.Equal(t, reflect.Ptr, reflect.ValueOf(i).Kind())
	assert.Equal(t, &testliteral, i)
}

func TestStringValue(t *testing.T) {
	testliteral := "literal"
	teststrptr := &testliteral
	var sv interface{} = StringValue(teststrptr)
	assert.Equal(t, reflect.String, reflect.ValueOf(sv).Kind())
	assert.Equal(t, testliteral, sv)
	assert.Equal(t, "", StringValue((*string)(nil)))
}

func TestInt64(t *testing.T) {
	var testint int64 = 7
	testintptr := Int64(testint)
	assert.Equal(t, reflect.Ptr, reflect.ValueOf(testintptr).Kind())
	assert.Equal(t, &testint, testintptr)
}

func TestUintValue(t *testing.T) {
	var testUintValue uint = 7
	testUintPtr := &testUintValue
	var uiv interface{} = UintValue(testUintPtr)
	assert.Equal(t, reflect.Uint, reflect.ValueOf(uiv).Kind())
	assert.Equal(t, testUintValue, uiv)
	assert.Equal(t, uint(0), UintValue((*uint)(nil)))
}

func TestStringToBool(t *testing.T) {
	testcases := []struct {
		teststring string
		pass       bool
	}{
		{
			teststring: "true",
			pass:       true,
		},
		{
			teststring: "True",
			pass:       true,
		},
		{
			teststring: "t",
			pass:       true,
		},
		{
			teststring: "yEs",
			pass:       true,
		},
		{
			teststring: "Y",
			pass:       true,
		},
		{
			teststring: "on",
			pass:       true,
		},
		{
			teststring: "tr",
			pass:       false,
		},
		{
			teststring: "nil",
			pass:       false,
		},
		{
			teststring: "false",
			pass:       false,
		},
		{
			teststring: "FOOBAR",
			pass:       false,
		},
	}

	for _, testcase := range testcases {
		assert.Equal(t, testcase.pass, StringToBool(testcase.teststring))
	}
}

func TestStringEmpty(t *testing.T) {
	assert.True(t, StringEmpty(""))
	assert.False(t, StringEmpty("NONEMPTY"))
}
