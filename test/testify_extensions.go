// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
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
}

func RequireNoUnexpectedErrors(f Fataler, errorTracker ErrorTracker) {
	if errorTracker.HasErrors() {
		f.Fatal("Test failed; encountered unexpected errors")
	}
}

type transactionStatuser interface {
	TransactionStatus() protocol.TransactionStatus
	TransactionReceipt() *protocol.TransactionReceipt
}

func RequireSuccess(t testing.TB, tx transactionStatuser, msg string, args ...interface{}) {
	message := fmt.Sprintf(msg, args...)
	require.EqualValues(t, protocol.TRANSACTION_STATUS_COMMITTED.String(), tx.TransactionStatus().String(), msg)
	require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS.String(), tx.TransactionReceipt().ExecutionResult().String(), message)
}

func RequireDoesNotContainNil(t *testing.T, obj interface{}) bool {
	if obj == nil {
		return true
	}
	return valueContainsNil(reflect.ValueOf(obj).Elem())
}

func valueContainsNil(v reflect.Value) bool {
	k := v.Kind()
	switch k {
	case reflect.Ptr:
		return v.IsNil() || valueContainsNil(reflect.Indirect(v))
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Interface, reflect.Slice:
		return v.IsNil()
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).CanSet() { // this is the only "elegant" way you can find if a field is exported when using reflection
				if valueContainsNil(v.Field(i)) {
					return true
				}
			}
		}
	}
	return false
}
