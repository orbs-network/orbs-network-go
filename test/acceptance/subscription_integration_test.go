package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	"github.com/orbs-network/orbs-network-go/test/contracts/subscription"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSubscriptionCheck_ValidSubscription(t *testing.T) {
	newHarness().
		WithVirtualChainId(42).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			sim := network.ethereumConnection
			address, _, err := sim.DeployEthereumContract(sim.GetAuth(), subscription.SubscriptionABI, subscription.SubscriptionBin)
			stringAddress := address.String()

			sim.Commit()

			require.NoError(t, err, "failed deploying subscription contract")

			contract := callcontract.NewContractClient(network)
			res, _ := contract.RefreshSubscription(ctx, 0, stringAddress)
			test.RequireSuccess(t, res, "failed refreshing subscription")

			res, _ = contract.Transfer(ctx, 0, 50, 5, 6)
			test.RequireSuccess(t, res, "failed sending transaction to a virtual chain with a valid subscription")

		})
}
