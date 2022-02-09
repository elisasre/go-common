package common

import "strings"

// Unique returns unique array items
func Unique(values []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, value := range values {
		if _, ok := keys[value]; !ok {
			keys[value] = true
			list = append(list, value)
		}
	}
	return list
}

// EqualStringArrays compares equality of two string arrays
func EqualStringArrays(a, b []string) bool {
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

// ContainsInteger returns true if integer is found from array
func ContainsInteger(array []int, value int) bool {
	for _, currentValue := range array {
		if currentValue == value {
			return true
		}
	}
	return false
}

// ContainsString returns true if string is found from array
func ContainsString(array []string, word string) bool {
	return containsF(array, word, func(item, word string) bool {
		return item == word
	})
}

// ContainsIgnoreCase returns true if word is found from array. Case of word and words in array is ignored
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
