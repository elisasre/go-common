package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnyStartsWith(t *testing.T) {
	tests := []struct {
		inputSlice  []string
		inputString string
		want        bool
	}{
		{inputSlice: []string{"foo", "bar", "baz"}, inputString: "foo", want: true},
		{inputSlice: []string{"foobar", "bar", "baz"}, inputString: "foo", want: true},
		{inputSlice: []string{"Foo", "bar", "baz"}, inputString: "foo", want: false},
		{inputSlice: []string{"Foobar", "bar", "baz"}, inputString: "foo", want: false},
		{inputSlice: []string{"bar", "baz"}, inputString: "foo", want: false},
	}
	for _, tc := range tests {
		result := AnyStartsWith(tc.inputSlice, tc.inputString)
		if result != tc.want {
			t.Errorf(
				"%v in %v, expected %v got %v", tc.inputString, tc.inputSlice, tc.want, result)
		}
	}
}

func TestGetResultDiff(t *testing.T) {
	tests := []struct {
		inputSlice              []string
		inputDesiredResultSlice []string
		want                    []string
	}{
		{
			inputSlice:              []string{"foo", "bar", "baz"},
			inputDesiredResultSlice: []string{"foobar"}, want: []string{"foobar"},
		},
		{
			inputSlice:              []string{"foo", "bar", "baz"},
			inputDesiredResultSlice: []string{"foobar", "foo"}, want: []string{"foobar"},
		},
		{
			inputSlice:              []string{"foo", "bar", "baz"},
			inputDesiredResultSlice: []string{"Foo"}, want: []string{"Foo"},
		},
		{
			inputSlice:              []string{"foo", "bar", "baz"},
			inputDesiredResultSlice: []string{"foo", "bar", "baz"}, want: []string{},
		},
	}

	for _, tc := range tests {
		results := GetResultDiff(tc.inputSlice, tc.inputDesiredResultSlice)
		if !EqualStringArrays(results, tc.want) {
			t.Errorf("Expected not to find %v in %v", tc.inputDesiredResultSlice, tc.inputSlice)
		}
	}
}

func TestUnique(t *testing.T) {
	array := []string{"1", "2", "2", "2", "3"}
	uniqueArray := Unique(array)
	assert.Equal(t, uniqueArray, []string{"1", "2", "3"})
}

func TestEqualStringArrays(t *testing.T) {
	arr1 := []string{"1", "2", "3"}
	arr2 := []string{"1", "2", "3", "4"}
	arr3 := []string{"1", "2", "3", "4"}
	arr4 := []string{"1", "2", "3", "5"}

	assert.False(t, EqualStringArrays(arr1, arr2))
	assert.True(t, EqualStringArrays(arr2, arr3))
	assert.False(t, EqualStringArrays(arr3, arr4))

	tests := []struct {
		input1 []string
		input2 []string
		want   bool
	}{
		{input1: []string{"foo", "bar"}, input2: []string{"foo", "foo", "bar"}, want: false},
		{input1: []string{"foo", "bar"}, input2: []string{"foo", "BAR"}, want: false},
		{input1: []string{"foo", "bar"}, input2: []string{"foo", "bar"}, want: true},
	}
	for _, tc := range tests {
		assert.Equal(t, EqualStringArrays(tc.input1, tc.input2), tc.want)
	}
}

func TestContainsInteger(t *testing.T) {
	tests := []struct {
		inputSlice []int
		input      int
		want       bool
	}{
		{inputSlice: []int{2, 3}, input: 2, want: true},
		{inputSlice: []int{3, 4}, input: 2, want: false},
	}
	for _, tc := range tests {
		assert.Equal(t, ContainsInteger(tc.inputSlice, tc.input), tc.want)
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		inputSlice  []string
		inputString string
		want        bool
	}{
		{inputSlice: []string{"foo", "bar"}, inputString: "bar", want: true},
		{inputSlice: []string{"foo", "bar"}, inputString: "BAR", want: false},
		{inputSlice: []string{"foo", "foo"}, inputString: "bar", want: false},
		{inputSlice: []string{"foo", "foo"}, inputString: "", want: false},
	}
	for _, tc := range tests {
		assert.Equal(t, ContainsString(tc.inputSlice, tc.inputString), tc.want)
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		inputSlice  []string
		inputString string
		want        bool
	}{
		{inputSlice: []string{"foo", "bar"}, inputString: "bar", want: true},
		{inputSlice: []string{"foo", "bar"}, inputString: "BAR", want: true},
		{inputSlice: []string{"foo", "bar"}, inputString: "bAR", want: true},
		{inputSlice: []string{"foo", "foo"}, inputString: "bar", want: false},
	}
	for _, tc := range tests {
		assert.Equal(t, ContainsIgnoreCase(tc.inputSlice, tc.inputString), tc.want)
	}
}
