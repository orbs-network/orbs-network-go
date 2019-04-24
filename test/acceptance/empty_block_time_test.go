package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/scribe/log"
	"testing"
	"time"
)

func TestIncomingTransactionTriggersImmediateBlockClosure(t *testing.T) {
	newHarness().
		WithEmptyBlockTime(1*time.Hour).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		WithLogFilters(log.ExcludeEntryPoint("BlockSync")).
		Start(t, func(tb testing.TB, ctx context.Context, network *NetworkHarness) {
			contract := callcontract.NewContractClient(network)
			time.Sleep(5 * time.Second)
			_, txHash := contract.Transfer(ctx, 0, 43, 5, 6)
			network.WaitForTransactionInState(ctx, txHash)

		})
}
