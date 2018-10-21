package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProcessCall_Permissions(t *testing.T) {
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test.WithContext(func(ctx context.Context) {
				h := newHarness()
				if tt.expectedSdkWrite {
					h.expectSdkCallMadeWithStateWrite(nil, nil)
				}

				_, err := h.service.ProcessCall(ctx, tt.input)
				if tt.expectedError {
					require.Error(t, err, "call should fail")
				} else {
					require.NoError(t, err, "call should succeed")
				}
			})
		})
	}
}
