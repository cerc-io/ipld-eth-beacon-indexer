package testhelpers

import (
	"reflect"
	"testing"
)

// ExpectEqual asserts the provided interfaces are deep equal
func ExpectEqual(t *testing.T, got interface{}, want interface{}) {
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Expected: %v\nActual: %v", want, got)
	}
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
