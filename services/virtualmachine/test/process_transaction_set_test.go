// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessTransactionSet_Success(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Transaction 1: successful")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectNativeContractMethodCalled("Contract1", "method2", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Transaction 2: successful")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(uint32(17), "hello", []byte{0x01, 0x02}), nil
		})

		results, outputArgs, _, _ := h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
			{"Contract1", "method2"},
		})
		require.Equal(t, results, []protocol.ExecutionResult{
			protocol.EXECUTION_RESULT_SUCCESS,
			protocol.EXECUTION_RESULT_SUCCESS,
		}, "processTransactionSet returned receipts should match")
		require.Equal(t, outputArgs, [][]byte{
			{},
			builders.PackedArgumentArrayEncode(uint32(17), "hello", []byte{0x01, 0x02}),
		}, "processTransactionSet returned output args should match")

		h.verifySystemContractCalled(t)
		h.verifyNativeContractMethodCalled(t)
	})
}

func TestProcessTransactionSet_WithErrors(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Transaction 1: failed (contract error)")
			return protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, builders.ArgumentsArray(), errors.New("contract error")
		})
		h.expectNativeContractMethodCalled("Contract1", "method2", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Transaction 2: failed (unexpected error)")
			return protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, builders.ArgumentsArray(), errors.New("unexpected error")
		})

		results, outputArgs, _, _ := h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
			{"Contract1", "method2"},
		})
		require.Equal(t, results, []protocol.ExecutionResult{
			protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT,
			protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		}, "processTransactionSet returned receipts should match")
		require.Equal(t, outputArgs, [][]byte{
			{},
			{},
		}, "processTransactionSet returned output args should match")

		h.verifySystemContractCalled(t)
		h.verifyNativeContractMethodCalled(t)
	})
}
