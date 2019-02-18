package acceptance

import (
	"context"
	"github.com/orbs-network/lean-helix-go/services/quorum"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	. "github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// Control group - if this fails, there are bugs unrelated to message tampering
func TestGazillionTxHappyFlow(t *testing.T) {
	getStressTestHarness().
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			h := newStressTestHarness(t, ctx, network)

			h.sendTransfersAndAssertTotalBalanceInMinimalQuorumSize(ctx, 100)
			h.sendSingleTransactionAndAssertBalanceInFullNetwork(ctx)

		})
}

func TestGazillionTxWhileDuplicatingMessages(t *testing.T) {
	getStressTestHarness().
		AllowingErrors(
			"error adding forwarded transaction to pending pool", // because we duplicate, among other messages, the transaction propagation message
		).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			h := newStressTestHarness(t, ctx, network)

			tamperer := network.TransportTamperer().Duplicate(WithPercentChance(h.rnd, 15))
			h.sendTransfersAndAssertTotalBalanceInMinimalQuorumSize(ctx, 100)

			tamperer.StopTampering(ctx)
			h.sendSingleTransactionAndAssertBalanceInFullNetwork(ctx)

		})
}

// TODO (v1) Must drop message from up to "f" fixed nodes (for 4 nodes f=1)
func TestGazillionTxWhileDroppingMessages(t *testing.T) {
	getStressTestHarness().
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			h := newStressTestHarness(t, ctx, network)

			tamperer := network.TransportTamperer().Fail(HasHeader(AConsensusMessage).And(WithPercentChance(h.rnd, 12)))
			h.sendTransfersAndAssertTotalBalanceInMinimalQuorumSize(ctx, 100)

			tamperer.StopTampering(ctx)
			h.sendSingleTransactionAndAssertBalanceInFullNetwork(ctx)
		})
}

// See BLOCK_SYNC_COLLECT_CHUNKS_TIMEOUT - cannot delay messages consistently more than that, or block sync will never work - it throws "timed out when waiting for chunks"
func TestGazillionTxWhileDelayingMessages(t *testing.T) {
	getStressTestHarness().
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			h := newStressTestHarness(t, ctx, network)
			tamperer := network.TransportTamperer().Delay(func() time.Duration {
				return (time.Duration(h.rnd.Intn(50))) * time.Millisecond
			}, WithPercentChance(h.rnd, 30))
			h.sendTransfersAndAssertTotalBalanceInMinimalQuorumSize(ctx, 100)

			tamperer.StopTampering(ctx)
			h.sendSingleTransactionAndAssertBalanceInFullNetwork(ctx)
		})
}

// TODO (v1) Must corrupt message from up to "f" fixed nodes (for 4 nodes f=1)
func TestGazillionTxWhileCorruptingMessages(t *testing.T) {
	t.Skip("This should work - fix and remove Skip")
	getStressTestHarness().
		AllowingErrors(
			"transport header is corrupt", // because we corrupt messages
		).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			h := newStressTestHarness(t, ctx, network)
			tamper := network.TransportTamperer().Corrupt(Not(HasHeader(ATransactionRelayMessage)).And(WithPercentChance(h.rnd, 15)), h.rnd)

			h.sendTransfersAndAssertTotalBalanceInMinimalQuorumSize(ctx, 90)
			tamper.StopTampering(ctx)

			h.sendTransfersAndAssertTotalBalanceInMinimalQuorumSize(ctx, 10)

			h.sendSingleTransactionAndAssertBalanceInFullNetwork(ctx)
		})
}

func WithPercentChance(ctrlRand *rand.ControlledRand, pct int) MessagePredicate {
	return func(data *adapter.TransportData) bool {
		if pct >= 100 {
			return true
		} else if pct <= 0 {
			return false
		} else {
			return ctrlRand.Intn(101) <= pct
		}
	}
}

func TestWithNPctChance_AlwaysTrue(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)
	require.True(t, WithPercentChance(ctrlRand, 100)(nil), "100% chance should always return true")
}

func TestWithNPctChance_AlwaysFalse(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)
	require.False(t, WithPercentChance(ctrlRand, 0)(nil), "0% chance should always return false")
}

func TestWithNPctChance_ManualCheck(t *testing.T) {
	ctrlRand := rand.NewControlledRand(t)
	tries := 1000
	pct := ctrlRand.Intn(100)
	hits := 0
	for i := 0; i < tries; i++ {
		if WithPercentChance(ctrlRand, pct)(nil) {
			hits++
		}
	}
	t.Logf("Manual test for WithPercentChance: Tries=%d Chance=%d%% Hits=%d\n", tries, pct, hits)
}

type stressTestHarness struct {
	fromAddress int
	toAddress   int
	network     *NetworkHarness
	tb          testing.TB
	contract    callcontract.BenchmarkTokenClient
	rnd         *rand.ControlledRand
}

func newStressTestHarness(tb testing.TB, ctx context.Context, network *NetworkHarness) *stressTestHarness {

	h := &stressTestHarness{
		fromAddress: 5,
		toAddress:   6,
		network:     network,
		tb:          tb,
		rnd:         rand.NewControlledRand(tb),
	}

	h.contract = network.DeployBenchmarkTokenContract(ctx, h.fromAddress)

	return h
}

func (h *stressTestHarness) sendTransfersAndAssertTotalBalanceInMinimalQuorumSize(ctx context.Context, numTransactions int) {
	quorumSize := quorum.CalcQuorumSize(h.network.Size()) // these tests do not require all network to sync, but rather, relying on the fact that 2f+1 nodes are the minimum required for the state to be considered "committed"

	var txHashes []primitives.Sha256
	for i := 0; i < numTransactions; i++ {

		txHash := h.contract.TransferInBackground(ctx, h.rnd.Intn(h.network.Size()), 17, h.fromAddress, h.toAddress)
		txHashes = append(txHashes, txHash)
	}

	committedNodeIndices := h.collectCommittedNodeIndices(ctx, txHashes, quorumSize)

	lastTxBlockHeights := h.collectBlockHeightsOfLastTx(ctx, txHashes, committedNodeIndices)
	h.requireLastTxIsInSameBlockHeightInAllCommittedNodes(lastTxBlockHeights)

	h.requireBlockContainingLastTxHasSameHashInAllCommittedNodes(lastTxBlockHeights, committedNodeIndices)

}

func (h *stressTestHarness) requireLastTxIsInSameBlockHeightInAllCommittedNodes(lastTxBlockHeights []primitives.BlockHeight) {
	for i := 1; i < len(lastTxBlockHeights); i++ {
		require.Equal(h.tb, lastTxBlockHeights[i], lastTxBlockHeights[i-1], "last transaction was not in the same block in all committed nodes")
	}
}

func (h *stressTestHarness) requireBlockContainingLastTxHasSameHashInAllCommittedNodes(lastTxBlockHeights []primitives.BlockHeight, committedNodeIndices []int) {
	for i := 1; i < len(lastTxBlockHeights); i++ {
		blocks1, err1 := h.network.Nodes[committedNodeIndices[i-1]].ExtractBlocks()
		blocks2, err2 := h.network.Nodes[committedNodeIndices[i]].ExtractBlocks()
		require.NoError(h.tb, err1)
		require.NoError(h.tb, err2)

		hash1 := blocks1[lastTxBlockHeights[0]-1].TransactionsBlock.Header.PrevBlockHashPtr()
		hash2 := blocks2[lastTxBlockHeights[0]-1].TransactionsBlock.Header.PrevBlockHashPtr()

		test.RequireCmpEqual(h.tb, hash1, hash2, "last interesting block hash did not equal among all committed nodes")
	}
}

func (h *stressTestHarness) collectBlockHeightsOfLastTx(ctx context.Context, txHashes []primitives.Sha256, committedNodeIndices []int) (lastTxBlockHeights []primitives.BlockHeight) {
	lastTxHash := txHashes[len(txHashes)-1]
	for _, nodeIndex := range committedNodeIndices {
		bh, err := h.network.tamperingBlockPersistences[nodeIndex].WaitForTransaction(ctx, lastTxHash)
		require.NoError(h.tb, err, "A node that has already committed txhash %s failed to return its block height", lastTxHash)
		lastTxBlockHeights = append(lastTxBlockHeights, bh)
	}
	return
}

func (h *stressTestHarness) collectCommittedNodeIndices(parent context.Context, txHashes []primitives.Sha256, requiredQuorumSize int) (committedNodeIndices []int) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	ch := make(chan int, h.network.Size())
	for i := 0; i < h.network.Size(); i++ {
		nodeIndex := i
		supervised.GoOnce(h.network.Logger, func() {
			var err error
			for _, txHash := range txHashes {
				blockHeight, err := h.network.tamperingBlockPersistences[nodeIndex].WaitForTransaction(ctx, txHash)
				if err != nil {
					break
				}
				err = h.network.stateBlockHeightTrackers[nodeIndex].WaitForBlock(ctx, blockHeight)
				if err != nil {
					break
				}
			}
			if err == nil {
				ch <- nodeIndex
			}
		})
	}

	for i := 0; i < requiredQuorumSize; i++ {
		select {
		case nodeIndex := <-ch:
			committedNodeIndices = append(committedNodeIndices, nodeIndex)
		case <-parent.Done():
			h.tb.Fatalf("Failed to reach quorum size %d during stress test", requiredQuorumSize)
		}
	}
	return
}

func (h *stressTestHarness) sendSingleTransactionAndAssertBalanceInFullNetwork(ctx context.Context) {
	txHash := h.contract.TransferInBackground(ctx, h.rnd.Intn(h.network.Size()), 42, h.fromAddress, h.toAddress)
	h.network.WaitForTransactionInState(ctx, txHash)
}
