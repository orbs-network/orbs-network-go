package test

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type transactionStatuser interface {
	TransactionStatus() protocol.TransactionStatus
	TransactionReceipt() *protocol.TransactionReceipt
}

func RequireSuccess(t *testing.T, tx transactionStatuser, msg string, args ...interface{}) {
	message := fmt.Sprintf(msg, args...)
	RequireStatus(t, protocol.TRANSACTION_STATUS_COMMITTED, tx, message)
	require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, tx.TransactionReceipt().ExecutionResult(), message)

}

func RequireStatus(t *testing.T, status protocol.TransactionStatus, tx transactionStatuser, msg string) {
	require.EqualValues(t, status, tx.TransactionStatus(), msg)
}
