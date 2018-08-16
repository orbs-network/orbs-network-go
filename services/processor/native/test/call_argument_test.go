package test

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessCallArguments(t *testing.T) {
	tests := []struct {
		name           string
		input          *services.ProcessCallInput
		expectedError  bool
		expectedResult protocol.ExecutionResult
		expectedOutput []*protocol.MethodArgument
	}{
		{
			name:           "WithNoArgsAndNoReturn",
			input:          processCallInput().WithMethod("BenchmarkContract", "nop").Build(),
			expectedError:  false,
			expectedResult: protocol.EXECUTION_RESULT_SUCCESS,
			expectedOutput: builders.MethodArguments(),
		},
		{
			name:           "WithAllArgTypes",
			input:          processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(uint32(11), uint64(12), "hello", []byte{0x01, 0x02, 0x03}).Build(),
			expectedError:  false,
			expectedResult: protocol.EXECUTION_RESULT_SUCCESS,
			expectedOutput: builders.MethodArguments(uint32(12), uint64(13), "hello1", []byte{0x01, 0x02, 0x03, 0x01}),
		},
		{
			name:           "WithIncorrectArgTypeFails",
			input:          processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(uint64(12), uint32(11), []byte{0x01, 0x02, 0x03}, "hello").Build(),
			expectedError:  true,
			expectedResult: protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		},
		{
			name:           "WithTooLittleArgsFails",
			input:          processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(uint32(11), uint64(12), "hello").Build(),
			expectedError:  true,
			expectedResult: protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		},
		{
			name:           "WithTooManyArgsFails",
			input:          processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(uint32(11), uint64(12), "hello", []byte{0x01, 0x02, 0x03}, uint32(11)).Build(),
			expectedError:  true,
			expectedResult: protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		},
		{
			name:           "WithUnknownArgSliceTypeFails",
			input:          processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(uint32(11), uint64(12), "hello", []int{0x01, 0x02, 0x03}).Build(),
			expectedError:  true,
			expectedResult: protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		},
		{
			name:           "WithUnknownArgTypeFails",
			input:          processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(float32(11), uint64(12), "hello", []byte{0x01, 0x02, 0x03}).Build(),
			expectedError:  true,
			expectedResult: protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		},
		{
			name:           "ThatThrowsError",
			input:          processCallInput().WithMethod("BenchmarkContract", "throw").Build(),
			expectedError:  true,
			expectedResult: protocol.EXECUTION_RESULT_ERROR_SMART_CONTRACT,
		},
		{
			name:           "ThatPanics",
			input:          processCallInput().WithMethod("BenchmarkContract", "panic").Build(),
			expectedError:  true,
			expectedResult: protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		},
		{
			name:           "WithInvalidMethodMissingErrorFails",
			input:          processCallInput().WithMethod("BenchmarkContract", "invalidNoError").Build(),
			expectedError:  true,
			expectedResult: protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		},
		{
			name:           "WithInvalidMethodMissingContextFails",
			input:          processCallInput().WithMethod("BenchmarkContract", "invalidNoContext").Build(),
			expectedError:  true,
			expectedResult: protocol.EXECUTION_RESULT_ERROR_UNEXPECTED,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newHarness()

			output, err := h.service.ProcessCall(test.input)
			if test.expectedError {
				require.Error(t, err, "call should fail")
			} else {
				require.NoError(t, err, "call should succeed")
				require.Equal(t, test.expectedOutput, output.OutputArguments, "call return args should be equal")
			}
			require.Equal(t, test.expectedResult, output.CallResult, "call result should be equal")
		})
	}
}
