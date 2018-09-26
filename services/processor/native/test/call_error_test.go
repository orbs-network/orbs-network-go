package test

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessCall_Errors(t *testing.T) {
	tests := []struct {
		name           string
		input          *services.ProcessCallInput
		expectedError  bool
		expectedResult protocol.ExecutionResult
		expectedOutput *protocol.MethodArgumentArray
	}{
		{
			name:           "ThatThrowsError",
			input:          processCallInput().WithMethod("BenchmarkContract", "throw").Build(),
			expectedError:  true,
			expectedResult: protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT,
			expectedOutput: builders.MethodArgumentsArray("example error returned by contract"),
		},
		{
			name:           "ThatPanics",
			input:          processCallInput().WithMethod("BenchmarkContract", "panic").Build(),
			expectedError:  true,
			expectedResult: protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT,
			expectedOutput: builders.MethodArgumentsArray("example panic thrown by contract"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newHarness()

			output, err := h.service.ProcessCall(test.input)
			if test.expectedError {
				require.Error(t, err, "call should fail")
				require.Equal(t, test.expectedOutput, output.OutputArgumentArray, "call return args should be equal")
			} else {
				require.NoError(t, err, "call should succeed")
				require.Equal(t, test.expectedOutput, output.OutputArgumentArray, "call return args should be equal")
			}
			require.Equal(t, test.expectedResult, output.CallResult, "call result should be equal")
		})
	}
}
