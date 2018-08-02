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

func TestCallAllArgTypesNoReturn(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(uint32(11), uint64(12), "hello", []byte{0x01, 0x02, 0x03}).Build()

	output, err := h.service.ProcessCall(call)
	assert.NoError(t, err)
	assert.Equal(t, output.CallResult, protocol.EXECUTION_RESULT_SUCCESS)

	res := []*protocol.MethodArgument{}
	assert.Equal(t, output.OutputArguments, res)
}

func TestCallIncorrectArgTypeFails(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(uint64(12), uint32(11), []byte{0x01, 0x02, 0x03}, "hello").Build()

	_, err := h.service.ProcessCall(call)
	assert.Error(t, err)
}

func TestCallIncorrectArgNumFails(t *testing.T) {
	h := newHarness()
	tooLittleCall := processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(uint32(11), uint64(12), "hello").Build()

	_, err := h.service.ProcessCall(tooLittleCall)
	assert.Error(t, err)

	tooMuchCall := processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(uint32(11), uint64(12), "hello", []byte{0x01, 0x02, 0x03}, uint32(11)).Build()

	_, err = h.service.ProcessCall(tooMuchCall)
	assert.Error(t, err)
}
