package common

import "testing"

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
