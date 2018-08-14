package test

import (
	"encoding/binary"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBenchmarkTokenGetBalancePostInit(t *testing.T) {
	h := newHarness()

	t.Log("Runs BenchmarkToken.getBalance")

	call := processCallInput().WithMethod("BenchmarkToken", "getBalance").Build()
	h.expectSdkCallMadeWithStateRead([]byte{})

	output, err := h.service.ProcessCall(call)
	require.NoError(t, err, "call should succeed")
	require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
	require.Equal(t, builders.MethodArguments(uint64(0)), output.OutputArguments, "call return args should be equal")
	h.verifySdkCallMade(t)
}

func TestBenchmarkTokenTransferThenGetBalance(t *testing.T) {
	h := newHarness()
	const valueAsUint64 = uint64(11)
	valueAsBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(valueAsBytes, valueAsUint64)

	t.Log("Runs BenchmarkToken.transfer")

	call := processCallInput().WithMethod("BenchmarkToken", "transfer").WithArgs(valueAsUint64).WithWriteAccess().Build()
	h.expectSdkCallMadeWithStateRead([]byte{})
	h.expectSdkCallMadeWithStateWrite()

	output, err := h.service.ProcessCall(call)
	require.NoError(t, err, "call should succeed")
	require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
	require.Equal(t, builders.MethodArguments(), output.OutputArguments, "call return args should be equal")
	h.verifySdkCallMade(t)

	t.Log("Runs BenchmarkToken.getBalance")

	call = processCallInput().WithMethod("BenchmarkToken", "getBalance").Build()
	h.expectSdkCallMadeWithStateRead(valueAsBytes)

	output, err = h.service.ProcessCall(call)
	require.NoError(t, err, "call should succeed")
	require.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
	require.Equal(t, builders.MethodArguments(valueAsUint64), output.OutputArguments, "call return args should be equal")
	h.verifySdkCallMade(t)
}

func TestBenchmarkTokenTransferLargeAmountFails(t *testing.T) {
	h := newHarness()

	t.Log("Runs BenchmarkToken.transfer large amount")

	call := processCallInput().WithMethod("BenchmarkToken", "transfer").WithArgs(uint64(9999)).WithWriteAccess().Build()

	output, err := h.service.ProcessCall(call)
	require.Error(t, err, "call should fail")
	require.Equal(t, protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, output.CallResult, "call result should be smart contract error")
}
