package testhelpers

import (
	"fmt"
	"reflect"
)

// ExpectEqual asserts the provided interfaces are deep equal
func IsEqual(got interface{}, want interface{}) (bool, error) {
	if !reflect.DeepEqual(got, want) {
		return false, fmt.Errorf("Expected: %v\nActual: %v", want, got)
	}
	return true, nil
}

// ListContainsString used to check if a list of strings contains a particular string
func ListContainsString(sss []string, s string) bool {
	for _, str := range sss {
		if s == str {
			return true
		}
	}
	return false
}
