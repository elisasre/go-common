package common_test

import (
	"testing"

	"github.com/elisasre/go-common/v2"
	"github.com/stretchr/testify/assert"
)

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
		result := common.EnsureDot(tc.input)
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
		result := common.RemoveDot(tc.input)
		if result != tc.want {
			t.Errorf(
				"Expected %v got %v", tc.input, tc.want)
		}
	}
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
		assert.Equal(t, testcase.pass, common.StringToBool(testcase.teststring))
	}
}

func TestReverseCopy(t *testing.T) {
	arr := []int{1, 2, 3, 4, 5}
	reversed := common.ReverseCopy(arr)
	expected := []int{5, 4, 3, 2, 1}
	assert.Equal(t, expected, reversed)
}
