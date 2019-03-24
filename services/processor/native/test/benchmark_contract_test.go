// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBenchmarkContract_SimpleCalculation(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)

		t.Log("Runs BenchmarkContract.add to add two numbers")

		call := processCallInput().WithMethod("BenchmarkContract", "add").WithArgs(uint64(12), uint64(27)).Build()

		output, err := h.service.ProcessCall(ctx, call)
		require.NoError(t, err, "call should succeed")
		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
		require.Equal(t, builders.ArgumentsArray(uint64(12+27)), output.OutputArgumentArray, "call return args should be equal")
	})
}

func TestBenchmarkContract_StateReadWrite(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)
		const value = uint64(41)

		t.Log("Runs BenchmarkContract.set to save a value in state")

		call := processCallInput().WithMethod("BenchmarkContract", "set").WithArgs(value).Build()
		h.expectSdkCallMadeWithStateWrite(nil, nil)

		output, err := h.service.ProcessCall(ctx, call)
		require.NoError(t, err, "call should succeed")
		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
		require.Equal(t, builders.ArgumentsArray(), output.OutputArgumentArray, "call return args should be equal")
		h.verifySdkCallMade(t)

		t.Log("Runs BenchmarkContract.get to read that value back from state")

		call = processCallInput().WithMethod("BenchmarkContract", "get").Build()
		h.expectSdkCallMadeWithStateRead(nil, uint64ToBytes(value))

		output, err = h.service.ProcessCall(ctx, call)
		require.NoError(t, err, "call should succeed")
		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
		require.Equal(t, builders.ArgumentsArray(value), output.OutputArgumentArray, "call return args should be equal")
		h.verifySdkCallMade(t)
	})
}

func TestBenchmarkContract_EmitEvent(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t)

		t.Log("Runs BenchmarkContract.giveBirth to emit an event")

		call := processCallInput().WithMethod("BenchmarkContract", "giveBirth").WithArgs("John Snow").Build()
		h.expectSdkCallMadeWithEventsEmit("BabyBorn", builders.ArgumentsArray("John Snow", uint32(3)), nil)

		output, err := h.service.ProcessCall(ctx, call)
		require.NoError(t, err, "call should succeed")
		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
		h.verifySdkCallMade(t)
	})
}
