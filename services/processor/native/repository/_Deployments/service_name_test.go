package deployments_systemcontract

import (
	. "github.com/orbs-network/orbs-contract-sdk/go/testing/unit"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOrbsDeployContract_isServiceNameValid(t *testing.T) {
	contractAlreadyDeployedName := "someName"
	InServiceScope(nil, nil, func(m Mockery) {
		_addServiceName(contractAlreadyDeployedName)
		tests := []struct {
			name         string
			contractName string
			shouldPanic  bool
		}{
			{"ok Name", "randomName", false},
			{"empty contract name", "", true},
			{"using system contract name", "_Elections", true},
			{"space at end", "randomName ", true},
			{"space in middle", "random Name", true},
			{"space in begin", " randomName", true},
			{"other non alpha", "random%Name", true},
			{"exact name again", contractAlreadyDeployedName, true},
			{"same name again but different lower/upper", "somename", true},
		}
		for _, tt := range tests {
			if tt.shouldPanic {
				require.Panics(t, func() { _validateServiceName(tt.contractName) }, "should panic when %s", tt.name)
			} else {
				require.NotPanics(t, func() { _validateServiceName(tt.contractName) }, "should not panic when %s", tt.name)
			}
		}
	})
}
