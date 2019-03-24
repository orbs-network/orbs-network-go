// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCallSystemContract_Success(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)

		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Input arguments are propagated correctly")
			require.EqualValues(t, builders.ArgumentsArray(uint32(17), "hello", []byte{0x01, 0x02}), inputArgs, "call system contract should propagate matching input args")

			t.Log("Read state key from contract (to make sure height is correct)")
			res, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_STATE, "read", []byte{0x01})
			require.NoError(t, err, "handleSdkCall should not fail")
			require.Equal(t, []byte{0xaa, 0xbb}, res[0].BytesValue(), "handleSdkCall result should be equal")

			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(uint32(19), "goodbye", []byte{0x03, 0x04}), nil
		})
		h.expectStateStorageRead(12, "Contract1", []byte{0x01}, []byte{0xaa, 0xbb})

		result, outputArgs, err := h.callSystemContract(ctx, 12, "Contract1", "method1", uint32(17), "hello", []byte{0x01, 0x02})
		require.NoError(t, err, "call system contract should not fail")
		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, result, "call system contract should return successful result")
		require.EqualValues(t, builders.ArgumentsArray(uint32(19), "goodbye", []byte{0x03, 0x04}), outputArgs, "call system contract should return matching output args")

		h.verifyNativeContractMethodCalled(t)
		h.verifyStateStorageRead(t)
	})
}
