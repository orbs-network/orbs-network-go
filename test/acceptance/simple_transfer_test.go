package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkToken"
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLeaderCommitsTransactionsAndSkipsInvalidOnes(t *testing.T) {
	harness.Network(t).
		AllowingErrors(
			"consensus round tick failed", // (aborting shared state update due to inconsistency) //TODO investigate and explain, or fix and remove expected error
		).Start(func(parent context.Context, network harness.InProcessTestNetwork) {
		ctx, _ := context.WithTimeout(parent, 1 * time.Second)

		contract := network.GetBenchmarkTokenContract()
		contract.DeployBenchmarkToken(ctx, 5)

		t.Log("testing", network.Description()) // leader is nodeIndex 0, validator is nodeIndex 1

		tx1 := <-contract.SendTransfer(ctx, 0, 17, 5, 6)
		<-contract.SendInvalidTransfer(ctx, 0, 5, 6)
		tx2 := <-contract.SendTransfer(ctx, 0, 22, 5, 6)

		t.Log("waiting for leader blocks")

		network.WaitForTransactionInState(ctx, 0, tx1.TransactionReceipt().Txhash())
		network.WaitForTransactionInState(ctx, 0, tx2.TransactionReceipt().Txhash())
		require.EqualValues(t, benchmarktoken.TOTAL_SUPPLY-39, <-contract.CallGetBalance(ctx, 0, 5), "getBalance result on leader")
		require.EqualValues(t, 39, <-contract.CallGetBalance(ctx, 0, 6), "getBalance result on leader")

		t.Log("waiting for non leader blocks")

		network.WaitForTransactionInState(ctx, 1, tx1.TransactionReceipt().Txhash())
		network.WaitForTransactionInState(ctx, 1, tx2.TransactionReceipt().Txhash())
		require.EqualValues(t, benchmarktoken.TOTAL_SUPPLY-39, <-contract.CallGetBalance(ctx, 1, 5), "getBalance result on non leader")
		require.EqualValues(t, 39, <-contract.CallGetBalance(ctx, 1, 6), "getBalance result on non leader")

	})
}

func TestNonLeaderPropagatesTransactionsToLeader(t *testing.T) {
	harness.Network(t).
		AllowingErrors(
			"consensus round tick failed", // (aborting shared state update due to inconsistency) //TODO investigate and explain, or fix and remove expected error
		).Start(func(parent context.Context, network harness.InProcessTestNetwork) {
		ctx, _ := context.WithTimeout(parent, 1 * time.Second)

		contract := network.GetBenchmarkTokenContract()
		contract.DeployBenchmarkToken(ctx, 5)

		t.Log("testing", network.Description()) // leader is nodeIndex 0, validator is nodeIndex 1

		pausedTxForwards := network.GossipTransport().Pause(adapter.TransactionRelayMessage(gossipmessages.TRANSACTION_RELAY_FORWARDED_TRANSACTIONS))
		txHash := contract.SendTransferInBackground(ctx, 1, 17, 5, 6)

		if err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(ctx, 2); err != nil {
			t.Errorf("failed waiting for block on node 0: %s", err)
		}
		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(ctx, 2); err != nil {
			t.Errorf("failed waiting for block on node 1: %s", err)
		}

		require.EqualValues(t, 0, <-contract.CallGetBalance(ctx, 0, 6), "initial getBalance result on leader")
		require.EqualValues(t, 0, <-contract.CallGetBalance(ctx, 1, 6), "initial getBalance result on non leader")

		pausedTxForwards.Release(ctx)
		network.WaitForTransactionInState(ctx, 0, txHash)
		require.EqualValues(t, 17, <-contract.CallGetBalance(ctx, 0, 6), "eventual getBalance result on leader")
		network.WaitForTransactionInState(ctx, 1, txHash)
		require.EqualValues(t, 17, <-contract.CallGetBalance(ctx, 1, 6), "eventual getBalance result on non leader")

	})
}
