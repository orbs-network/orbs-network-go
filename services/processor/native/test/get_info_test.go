package test

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetContractInfo(t *testing.T) {
	tests := []struct {
		name                string
		input               *services.GetContractInfoInput
		expectedError       bool
		expectedPermissions protocol.ExecutionPermissionScope
	}{
		{
			name:          "UnknownContract",
			input:         getContractInfoInput().WithUnknownContract().Build(),
			expectedError: true,
		},
		{
			name:                "SystemService",
			input:               getContractInfoInput().WithSystemService().Build(),
			expectedError:       false,
			expectedPermissions: protocol.PERMISSION_SCOPE_SYSTEM,
		},
		{
			name:                "RegularService",
			input:               getContractInfoInput().WithRegularService().Build(),
			expectedError:       false,
			expectedPermissions: protocol.PERMISSION_SCOPE_SERVICE,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newHarness()

			output, err := h.service.GetContractInfo(test.input)
			if test.expectedError {
				require.Error(t, err, "GetContractInfo should fail")
			} else {
				require.NoError(t, err, "GetContractInfo should not fail")
				require.Equal(t, test.expectedPermissions, output.PermissionScope, "contract permissions should match")
			}
		})
	}
}
