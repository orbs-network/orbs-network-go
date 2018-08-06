package test

import (
	"encoding/binary"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBenchmarkTokenGetBalancePostInit(t *testing.T) {
	h := newHarness()

	t.Log("Runs BenchmarkToken.getBalance")

	call := processCallInput().WithMethod("BenchmarkToken", "getBalance").Build()
	h.expectSdkCallMadeWithStateRead([]byte{})

	output, err := h.service.ProcessCall(call)
	assert.NoError(t, err, "call should succeed")
	assert.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
	assert.Equal(t, builders.MethodArguments(uint64(0)), output.OutputArguments, "call return args should be equal")
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
	assert.NoError(t, err, "call should succeed")
	assert.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
	assert.Equal(t, builders.MethodArguments(), output.OutputArguments, "call return args should be equal")
	h.verifySdkCallMade(t)

	t.Log("Runs BenchmarkToken.getBalance")

	call = processCallInput().WithMethod("BenchmarkToken", "getBalance").Build()
	h.expectSdkCallMadeWithStateRead(valueAsBytes)

	output, err = h.service.ProcessCall(call)
	assert.NoError(t, err, "call should succeed")
	assert.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
	assert.Equal(t, builders.MethodArguments(valueAsUint64), output.OutputArguments, "call return args should be equal")
	h.verifySdkCallMade(t)
}

func TestBenchmarkTokenTransferZeroAmountFails(t *testing.T) {
	h := newHarness()

	t.Log("Runs BenchmarkToken.transfer zero amount")

	call := processCallInput().WithMethod("BenchmarkToken", "transfer").WithArgs(uint64(0)).WithWriteAccess().Build()

	output, err := h.service.ProcessCall(call)
	assert.Error(t, err, "call should fail")
	assert.Equal(t, protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, output.CallResult, "call result should be smart contract error")
}
