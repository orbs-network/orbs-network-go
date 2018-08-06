package test

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCallWithNoArgsAndNoReturn(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithMethod("BenchmarkContract", "_init").Build()

	output, err := h.service.ProcessCall(call)
	assert.NoError(t, err, "call should succeed")
	assert.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
	assert.Equal(t, argumentBuilder(), output.OutputArguments, "call return args should be empty")
}

func TestCallWithAllArgTypes(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(uint32(11), uint64(12), "hello", []byte{0x01, 0x02, 0x03}).Build()

	output, err := h.service.ProcessCall(call)
	assert.NoError(t, err, "call should succeed")
	assert.Equal(t, protocol.EXECUTION_RESULT_SUCCESS, output.CallResult, "call result should be success")
	assert.Equal(t, argumentBuilder(uint32(12), uint64(13), "hello1", []byte{0x01, 0x02, 0x03, 0x01}), output.OutputArguments, "call return args should be equal")
}

func TestCallWithIncorrectArgTypeFails(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(uint64(12), uint32(11), []byte{0x01, 0x02, 0x03}, "hello").Build()

	output, err := h.service.ProcessCall(call)
	assert.Error(t, err, "call should fail")
	assert.Equal(t, protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, output.CallResult, "call result should be unexpected error")
}

func TestCallWithIncorrectArgNumFails(t *testing.T) {
	h := newHarness()
	tooLittleCall := processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(uint32(11), uint64(12), "hello").Build()

	output, err := h.service.ProcessCall(tooLittleCall)
	assert.Error(t, err, "call should fail")
	assert.Equal(t, protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, output.CallResult, "call result should be unexpected error")

	tooMuchCall := processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(uint32(11), uint64(12), "hello", []byte{0x01, 0x02, 0x03}, uint32(11)).Build()

	output, err = h.service.ProcessCall(tooMuchCall)
	assert.Error(t, err, "call should fail")
	assert.Equal(t, protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, output.CallResult, "call result should be unexpected error")
}

func TestCallWithUnknownArgTypeFails(t *testing.T) {
	h := newHarness()
	call1 := processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(uint32(11), uint64(12), "hello", []int{0x01, 0x02, 0x03}).Build()

	output, err := h.service.ProcessCall(call1)
	assert.Error(t, err, "call should fail")
	assert.Equal(t, protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, output.CallResult, "call result should be unexpected error")

	call2 := processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(float32(11), uint64(12), "hello", []byte{0x01, 0x02, 0x03}).Build()

	output, err = h.service.ProcessCall(call2)
	assert.Error(t, err, "call should fail")
	assert.Equal(t, protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, output.CallResult, "call result should be unexpected error")
}

func TestCallThatThrowsError(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithMethod("BenchmarkContract", "throw").Build()

	output, err := h.service.ProcessCall(call)
	assert.Error(t, err, "call should fail")
	assert.Equal(t, protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT, output.CallResult, "call result should be smart contract error")
}

func TestCallThatPanics(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithMethod("BenchmarkContract", "panic").Build()

	output, err := h.service.ProcessCall(call)
	assert.Error(t, err, "call should fail")
	assert.Equal(t, protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, output.CallResult, "call result should be unexpected error")
}

func TestCallWithInvalidMethodMissingErrorFails(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithMethod("BenchmarkContract", "invalidNoError").Build()

	output, err := h.service.ProcessCall(call)
	assert.Error(t, err, "call should fail")
	assert.Equal(t, protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, output.CallResult, "call result should be unexpected error")
}

func TestCallWithInvalidMethodMissingContextFails(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithMethod("BenchmarkContract", "invalidNoContext").Build()

	output, err := h.service.ProcessCall(call)
	assert.Error(t, err, "call should fail")
	assert.Equal(t, protocol.EXECUTION_RESULT_ERROR_UNEXPECTED, output.CallResult, "call result should be unexpected error")
}
