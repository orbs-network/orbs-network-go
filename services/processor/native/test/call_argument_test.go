package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessCall_Arguments(t *testing.T) {
	tests := []struct {
		name           string
		input          *services.ProcessCallInput
		expectedError  bool
		expectedResult protocol.ExecutionResult
		expectedOutput *protocol.MethodArgumentArray
	}{
		{
			name:           "WithAllArgTypes",
			input:          processCallInput().WithMethod("BenchmarkContract", "argTypes").WithArgs(uint32(11), uint64(12), "hello", []byte{0x01, 0x02, 0x03}).Build(),
			expectedError:  false,
			expectedResult: protocol.EXECUTION_RESULT_SUCCESS,
			expectedOutput: builders.MethodArgumentsArray(uint32(12), uint64(13), "hello1", []byte{0x01, 0x02, 0x03, 0x01}),
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test.WithContext(func(ctx context.Context) {
				h := newHarness()

				output, err := h.service.ProcessCall(ctx, tt.input)
				if tt.expectedError {
					require.Error(t, err, "call should fail")
					require.NotEmpty(t, output.OutputArgumentArray.ArgumentsIterator().NextArguments().StringValue(), "call return args should contain an error string")
				} else {
					require.NoError(t, err, "call should succeed")
					require.Equal(t, tt.expectedOutput, output.OutputArgumentArray, "call return args should be equal")
				}
				require.Equal(t, tt.expectedResult, output.CallResult, "call result should be equal")
			})
		})
	}
}
