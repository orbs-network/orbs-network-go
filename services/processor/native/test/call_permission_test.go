package test

import (
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessCallPermissions(t *testing.T) {
	tests := []struct {
		name             string
		input            *services.ProcessCallInput
		expectedError    bool
		expectedSdkWrite bool
	}{
		{
			name:          "UnknownMethodFails",
			input:         processCallInput().WithUnknownMethod().Build(),
			expectedError: true,
		},
		{
			name:          "ExternalMethodFromAnotherServiceSucceeds",
			input:         processCallInput().WithExternalMethod().WithDifferentCallingService().Build(),
			expectedError: false,
		},
		{
			name:          "InternalMethodFromSameServiceSucceeds",
			input:         processCallInput().WithInternalMethod().WithSameCallingService().Build(),
			expectedError: false,
		},
		{
			name:          "InternalMethodFromAnotherServiceFails",
			input:         processCallInput().WithInternalMethod().WithDifferentCallingService().Build(),
			expectedError: true,
		},
		{
			name:          "InternalMethodFromAnotherServiceUnderSystemPermissionsSucceeds",
			input:         processCallInput().WithInternalMethod().WithDifferentCallingService().WithSystemPermissions().Build(),
			expectedError: false,
		},
		{
			name:             "WriteMethodWithWriteAccessSucceeds",
			input:            processCallInput().WithExternalWriteMethod().WithWriteAccess().Build(),
			expectedError:    false,
			expectedSdkWrite: true,
		},
		{
			name:          "WriteMethodWithoutWriteAccessFails",
			input:         processCallInput().WithExternalWriteMethod().Build(),
			expectedError: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newHarness()
			if test.expectedSdkWrite {
				h.expectSdkCallMadeWithStateWrite(nil, nil)
			}

			_, err := h.service.ProcessCall(test.input)
			if test.expectedError {
				require.Error(t, err, "call should fail")
			} else {
				require.NoError(t, err, "call should succeed")
			}
		})
	}
}
