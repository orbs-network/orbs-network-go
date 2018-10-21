package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBenchmarkContract_AddMethod(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness()

		t.Log("Runs BenchmarkContract.add to add two numbers")

		call := processCallInput().WithMethod("BenchmarkContract", "add").WithArgs(uint64(12), uint64(27)).Build()

		output, err := h.service.ProcessCall(ctx, call)
		require.NoError(t, err, "call should succeed")
		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
		require.Equal(t, builders.MethodArgumentsArray(uint64(12+27)), output.OutputArgumentArray, "call return args should be equal")
	})
}

func TestBenchmarkContract_SetGetMethods(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness()
		const value = uint64(41)

		t.Log("Runs BenchmarkContract.set to save a value in state")

		call := processCallInput().WithMethod("BenchmarkContract", "set").WithArgs(value).WithWriteAccess().Build()
		h.expectSdkCallMadeWithStateWrite(nil, nil)

		output, err := h.service.ProcessCall(ctx, call)
		require.NoError(t, err, "call should succeed")
		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
		require.Equal(t, builders.MethodArgumentsArray(), output.OutputArgumentArray, "call return args should be equal")
		h.verifySdkCallMade(t)

		t.Log("Runs BenchmarkContract.get to read that value back from state")

		call = processCallInput().WithMethod("BenchmarkContract", "get").Build()
		h.expectSdkCallMadeWithStateRead(nil, uint64ToBytes(value))

		output, err = h.service.ProcessCall(ctx, call)
		require.NoError(t, err, "call should succeed")
		require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
		require.Equal(t, builders.MethodArgumentsArray(value), output.OutputArgumentArray, "call return args should be equal")
		h.verifySdkCallMade(t)
	})
}
