package common

import "strings"

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
