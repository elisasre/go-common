package common

import "strings"

// Unique returns unique array items.
func Unique[T comparable](values []T) []T {
	keys := make(map[T]bool)
	list := []T{}
	for _, value := range values {
		if _, ok := keys[value]; !ok {
			keys[value] = true
			list = append(list, value)
		}
	}
	return list
}

// EqualArrays compares equality of two arrays. Both input variables must be same type.
func EqualArrays[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// EqualStringArrays compares equality of two string arrays.
func EqualStringArrays(a, b []string) bool {
	return EqualArrays(a, b)
}

// Contains returns true if value is found in array. Both input variables must be same type.
func Contains[T comparable](array []T, value T) bool {
	for _, currentValue := range array {
		if currentValue == value {
			return true
		}
	}
	return false
}

// ContainsInteger returns true if integer is found from array.
func ContainsInteger(array []int, value int) bool {
	return Contains(array, value)
}

// ContainsString returns true if string is found from array.
func ContainsString(array []string, word string) bool {
	return Contains(array, word)
}

// ContainsIgnoreCase returns true if word is found from array. Case of word and words in array is ignored.
func ContainsIgnoreCase(array []string, word string) bool {
	return containsF(array, word, strings.EqualFold)
}

func containsF(array []string, word string, f func(item, word string) bool) bool {
	for _, item := range array {
		if f(item, word) {
			return true
		}
	}
	return false
}

// AnyStartsWith ...
func AnyStartsWith(array []string, word string) bool {
	for _, item := range array {
		if strings.HasPrefix(item, word) {
			return true
		}
	}
	return false
}

// GetResultDiff returns array of strings that were desired but missing from results.
func GetResultDiff[T comparable](results []T, desiredResults []T) []T {
	missingResults := []T{}
	for _, desiredResult := range desiredResults {
		found := false
		for _, result := range results {
			if desiredResult == result {
				found = true
			}
		}
		if !found {
			missingResults = append(missingResults, desiredResult)
		}
	}
	return missingResults
}
