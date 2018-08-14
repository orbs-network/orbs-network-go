package test

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"testing"
)

func AssertCmpEqual(t *testing.T, expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool {
	if !cmp.Equal(expected, actual) {
		diff := cmp.Diff(expected, actual)
		return assert.Fail(t, fmt.Sprintf("Not equal: \n"+
			"expected: %s\n"+
			"actual  : %s%s", expected, actual, diff), msgAndArgs...)
	}
	return true
}

func RequireCmpEqual(t *testing.T, expected interface{}, actual interface{}, msgAndArgs ...interface{}) {
	if AssertCmpEqual(t, expected, actual, msgAndArgs...) {
		return
	}
	t.FailNow()
}
