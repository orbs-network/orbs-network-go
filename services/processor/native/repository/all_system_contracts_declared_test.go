package repository

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkContract"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkToken"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Deployments"
	"github.com/stretchr/testify/require"
	"testing"
)

// Important this is a safety test that makes sure new system contracts are added to the deploy contract as system contract to avoid attempt at re-deploy.
func TestOrbsDeployContract_AreAllSystemContracts(t *testing.T) {
	r := NewPrebuilt()
	for contractName, _ := range r.preBuiltContracts {
		if contractName == benchmarkcontract.CONTRACT_NAME || contractName == benchmarktoken.CONTRACT_NAME {
			continue // Important these two contracts are in the pre-build for acceptance and e2e tests but are not actually system
		}
		require.True(t, deployments_systemcontract.IsImplicitlyDeployed(contractName), "deploy.go func _isImplicitlyDeployed is missing system contract with name %s", contractName)
	}
}
