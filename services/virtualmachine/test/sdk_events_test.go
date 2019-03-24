// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSdkEvents_EmitEvent_InTransactionReceipts(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Emit of Event1")
			_, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_EVENTS, "emitEvent", "Event1", builders.ArgumentsArray("hello").Raw())
			require.NoError(t, err, "handleSdkCall should succeed")

			t.Log("Emit of Event2")
			_, err = h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_EVENTS, "emitEvent", "Event2", builders.ArgumentsArray(uint64(17)).Raw())
			require.NoError(t, err, "handleSdkCall should succeed")

			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})
		h.expectNativeContractMethodCalled("Contract1", "method2", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})

		_, _, _, outputEvents := h.processTransactionSet(ctx, []*contractAndMethod{
			{"Contract1", "method1"},
			{"Contract1", "method2"},
		})

		expectedEventsArray1 := (&protocol.EventsArrayBuilder{
			Events: []*protocol.EventBuilder{
				{ContractName: "Contract1", EventName: "Event1", OutputArgumentArray: builders.PackedArgumentArrayEncode("hello")},
				{ContractName: "Contract1", EventName: "Event2", OutputArgumentArray: builders.PackedArgumentArrayEncode(uint64(17))},
			},
		}).Build().RawEventsArray()
		expectedEventsArray2 := (&protocol.EventsArrayBuilder{}).Build().RawEventsArray()

		require.Equal(t, expectedEventsArray1, outputEvents[0], "processTransactionSet returned output events should match")
		require.Equal(t, expectedEventsArray2, outputEvents[1], "processTransactionSet returned output events should match")

		h.verifySystemContractCalled(t)
		h.verifyNativeContractMethodCalled(t)
	})
}

func TestSdkEvents_EmitEvent_InProcessQuery(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		h.expectSystemContractCalled(deployments_systemcontract.CONTRACT_NAME, deployments_systemcontract.METHOD_GET_INFO, nil, uint32(protocol.PROCESSOR_TYPE_NATIVE)) // assume all contracts are deployed

		h.expectStateStorageBlockHeightRequested(12)
		h.expectNativeContractMethodCalled("Contract1", "method1", func(executionContextId primitives.ExecutionContextId, inputArgs *protocol.ArgumentArray) (protocol.ExecutionResult, *protocol.ArgumentArray, error) {
			t.Log("Emit of Event1")
			_, err := h.handleSdkCall(ctx, executionContextId, native.SDK_OPERATION_NAME_EVENTS, "emitEvent", "Event1", builders.ArgumentsArray("hello").Raw())
			require.NoError(t, err, "handleSdkCall should succeed")
			return protocol.EXECUTION_RESULT_SUCCESS, builders.ArgumentsArray(), nil
		})

		result, _, _, outputEvents, err := h.processQuery(ctx, "Contract1", "method1")
		require.NoError(t, err, "process query should not fail")
		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, result, "process query should return successful result")

		expectedEventsArray := (&protocol.EventsArrayBuilder{
			Events: []*protocol.EventBuilder{
				{ContractName: "Contract1", EventName: "Event1", OutputArgumentArray: builders.PackedArgumentArrayEncode("hello")},
			},
		}).Build().RawEventsArray()

		require.Equal(t, expectedEventsArray, outputEvents)

		h.verifySystemContractCalled(t)
		h.verifyStateStorageBlockHeightRequested(t)
		h.verifyNativeContractMethodCalled(t)
	})
}
