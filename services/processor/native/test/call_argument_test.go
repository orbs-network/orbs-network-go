package test

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCallNoArgsNoReturn(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithMethod("BenchmarkContract", "_init").WithArgs().Build()

	output, err := h.service.ProcessCall(call)
	assert.NoError(t, err)
	assert.Equal(t, output.CallResult, protocol.EXECUTION_RESULT_SUCCESS)

	res := []*protocol.MethodArgument{}
	assert.Equal(t, output.OutputArguments, res)
}

func TestCallUint64ArgNoReturn(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithMethod("BenchmarkContract", "set").WithArgs(uint64(12)).WithWriteAccess().Build()

	output, err := h.service.ProcessCall(call)
	assert.NoError(t, err)
	assert.Equal(t, output.CallResult, protocol.EXECUTION_RESULT_SUCCESS)

	res := []*protocol.MethodArgument{}
	assert.Equal(t, output.OutputArguments, res)
}

func TestCallIncorrectStringArgForUint64Fails(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithMethod("BenchmarkContract", "set").WithArgs("hello").WithWriteAccess().Build()

	_, err := h.service.ProcessCall(call)
	assert.Error(t, err)
}

func TestCallIncorrectArgNumFails(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithMethod("BenchmarkContract", "set").WithArgs(uint64(12), uint64(13)).WithWriteAccess().Build()

	_, err := h.service.ProcessCall(call)
	assert.Error(t, err)
}
