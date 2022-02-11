package common

import "testing"

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
		{inputSlice: []string{"foo", "bar", "baz"}, inputDesiredResultSlice: []string{"foobar"}, want: []string{"foobar"}},
		{inputSlice: []string{"foo", "bar", "baz"}, inputDesiredResultSlice: []string{"foobar", "foo"}, want: []string{"foobar"}},
		{inputSlice: []string{"foo", "bar", "baz"}, inputDesiredResultSlice: []string{"Foo"}, want: []string{"Foo"}},
		{inputSlice: []string{"foo", "bar", "baz"}, inputDesiredResultSlice: []string{"foo", "bar", "baz"}, want: []string{}},
	}

	for _, tc := range tests {
		results := GetResultDiff(tc.inputSlice, tc.inputDesiredResultSlice)
		if !EqualStringArrays(results, tc.want) {
			t.Errorf("Expected not to find %v in %v", tc.inputDesiredResultSlice, tc.inputSlice)
		}
	}
}
