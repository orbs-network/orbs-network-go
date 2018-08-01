package test

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCallNoArgsNoReturn(t *testing.T) {
	h := newHarness()
	args := []*protocol.MethodArgument{}
	call := processCallInput().WithMethod("BenchmarkContract", "_init").WithArgs(args).Build()

	output, err := h.service.ProcessCall(call)
	assert.NoError(t, err)
	assert.Equal(t, output.CallResult, protocol.EXECUTION_RESULT_SUCCESS)

	res := []*protocol.MethodArgument{}
	assert.Equal(t, output.OutputArguments, res)
}

func TestCallUint64ArgNoReturn(t *testing.T) {
	h := newHarness()
	args := []*protocol.MethodArgument{
		(&protocol.MethodArgumentBuilder{Name: "a", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: 12}).Build(),
	}
	call := processCallInput().WithMethod("BenchmarkContract", "set").WithArgs(args).WithWriteAccess().Build()

	output, err := h.service.ProcessCall(call)
	assert.NoError(t, err)
	assert.Equal(t, output.CallResult, protocol.EXECUTION_RESULT_SUCCESS)

	res := []*protocol.MethodArgument{}
	assert.Equal(t, output.OutputArguments, res)
}
