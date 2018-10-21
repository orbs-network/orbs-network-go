package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test.WithContext(func(ctx context.Context) {
				h := newHarness()

				output, err := h.service.GetContractInfo(ctx, tt.input)
				if tt.expectedError {
					require.Error(t, err, "GetContractInfo should fail")
				} else {
					require.NoError(t, err, "GetContractInfo should not fail")
					require.Equal(t, tt.expectedPermissions, output.PermissionScope, "contract permissions should match")
				}
			})
		})
	}
}
