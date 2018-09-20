package acceptance

import (
	"github.com/orbs-network/orbs-network-go/test/harness"
	"github.com/orbs-network/orbs-network-go/test/harness/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLeaderCommitsTransactionsAndSkipsInvalidOnes(t *testing.T) {
	harness.Network(t).Start(func(network harness.InProcessNetwork) {

		network.DeployBenchmarkToken()

		t.Log("testing", network.Description()) // leader is nodeIndex 0, validator is nodeIndex 1

		tx1 := <-network.SendTransfer(0, 17)
		<-network.SendInvalidTransfer(0)
		tx2 := <-network.SendTransfer(0, 22)

		t.Log("waiting for leader blocks")

		network.WaitForTransactionInState(0, tx1.TransactionReceipt().Txhash())
		network.WaitForTransactionInState(0, tx2.TransactionReceipt().Txhash())
		require.EqualValues(t, 39, <-network.CallGetBalance(0), "getBalance result on leader")

		t.Log("waiting for non leader blocks")

		network.WaitForTransactionInState(1, tx1.TransactionReceipt().Txhash())
		network.WaitForTransactionInState(1, tx2.TransactionReceipt().Txhash())
		require.EqualValues(t, 39, <-network.CallGetBalance(1), "getBalance result on non leader")

	})
}

func TestNonLeaderPropagatesTransactionsToLeader(t *testing.T) {
	harness.Network(t).Start(func(network harness.InProcessNetwork) {

		network.DeployBenchmarkToken()

		t.Log("testing", network.Description()) // leader is nodeIndex 0, validator is nodeIndex 1

		pausedTxForwards := network.GossipTransport().Pause(adapter.TransactionRelayMessage(gossipmessages.TRANSACTION_RELAY_FORWARDED_TRANSACTIONS))
		txHash := network.SendTransferInBackground(1, 17)

		if err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(2); err != nil {
			t.Errorf("failed waiting for block on node 0: %s", err)
		}
		if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(2); err != nil {
			t.Errorf("failed waiting for block on node 1: %s", err)
		}

		require.EqualValues(t, 0, <-network.CallGetBalance(0), "initial getBalance result on leader")
		require.EqualValues(t, 0, <-network.CallGetBalance(1), "initial getBalance result on non leader")

		pausedTxForwards.Release()
		network.WaitForTransactionInState(0, txHash)
		require.EqualValues(t, 17, <-network.CallGetBalance(0), "eventual getBalance result on leader")
		network.WaitForTransactionInState(1, txHash)
		require.EqualValues(t, 17, <-network.CallGetBalance(1), "eventual getBalance result on non leader")

	})
}
