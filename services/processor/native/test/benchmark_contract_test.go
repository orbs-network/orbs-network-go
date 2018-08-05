package test

import (
	"encoding/binary"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBenchmarkContractAddMethod(t *testing.T) {
	h := newHarness()

	t.Log("Runs BenchmarkContract.add to add two numbers")

	call := processCallInput().WithMethod("BenchmarkContract", "add").WithArgs(uint64(12), uint64(27)).Build()

	output, err := h.service.ProcessCall(call)
	assert.NoError(t, err, "call should succeed")
	assert.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
	assert.Equal(t, argumentBuilder(uint64(12+27)), output.OutputArguments, "call return args should be equal")
}

func TestBenchmarkContractSetGetMethods(t *testing.T) {
	h := newHarness()
	const valueAsUint64 = uint64(41)
	valueAsBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(valueAsBytes, valueAsUint64)

	t.Log("Runs BenchmarkContract.set to save a value in state")

	call := processCallInput().WithMethod("BenchmarkContract", "set").WithArgs(valueAsUint64).WithWriteAccess().Build()
	h.expectSdkCallMadeWithStateWrite()

	output, err := h.service.ProcessCall(call)
	assert.NoError(t, err, "call should succeed")
	assert.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
	assert.Equal(t, argumentBuilder(), output.OutputArguments, "call return args should be equal")
	h.verifySdkCallMade(t)

	t.Log("Runs BenchmarkContract.get to read that value back from state")

	call = processCallInput().WithMethod("BenchmarkContract", "get").Build()
	h.expectSdkCallMadeWithStateRead(valueAsBytes)

	output, err = h.service.ProcessCall(call)
	assert.NoError(t, err, "call should succeed")
	assert.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
	assert.Equal(t, argumentBuilder(valueAsUint64), output.OutputArguments, "call return args should be equal")
	h.verifySdkCallMade(t)
}
