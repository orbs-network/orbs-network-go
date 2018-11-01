package test

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"strings"
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

type Fataler interface {
	Fatal(args ...interface{})
}

type ErrorTracker interface {
	HasErrors() bool
	GetUnexpectedErrors() []string
}

func RequireNoUnexpectedErrors(f Fataler, errorTracker ErrorTracker) {
	if errorTracker.HasErrors() {
		f.Fatal("Encountered unexpected errors:\n\t", strings.Join(errorTracker.GetUnexpectedErrors(), "\n\t"))
	}
}