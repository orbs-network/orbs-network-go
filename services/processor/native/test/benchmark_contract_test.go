package test

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBenchmarkContractAddMethod(t *testing.T) {
	h := newHarness()
	call := processCallInput().WithMethod("BenchmarkContract", "add").WithArgs(uint64(12), uint64(27)).Build()

	output, err := h.service.ProcessCall(call)
	assert.NoError(t, err, "call should succeed")
	assert.Equal(t, output.CallResult, protocol.EXECUTION_RESULT_SUCCESS, "call result should be success")

	expected := argumentBuilder(uint64(39))
	assert.Equal(t, expected, output.OutputArguments, "call return args should be equal")
}
