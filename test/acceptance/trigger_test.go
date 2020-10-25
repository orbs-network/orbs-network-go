package acceptance

import (
	"context"
	"github.com/orbs-network/crypto-lib-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTriggers_ABlockContainsATriggerTransaction(t *testing.T) {
	NewHarness().
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {

			blockHeight := primitives.BlockHeight(1)
			network.WaitForBlock(ctx, blockHeight)

			block, err := network.PublicApi(0).GetBlock(ctx, &services.GetBlockInput{
				ClientRequest: (&client.GetBlockRequestBuilder{
					ProtocolVersion: config.MAXIMAL_CLIENT_PROTOCOL_VERSION,
					VirtualChainId:  builders.DEFAULT_TEST_VIRTUAL_CHAIN_ID,
					BlockHeight:     blockHeight,
				}).Build(),
			})
			require.NoError(t, err, "failed getting block with height %d", blockHeight)

			tx := lastTxOf(block)
			require.NotNil(t, tx, "block was empty")
			require.EqualValues(t, "_Triggers", tx.Transaction().ContractName(), "last transaction was not a call to Triggers contract")
			require.EqualValues(t, "trigger", tx.Transaction().MethodName(), "last transaction was not a call to trigger method")

			receipt := lastReceiptOf(block)
			require.NotNil(t, tx, "block had no receipts")
			require.EqualValues(t, digest.CalcTxHash(tx.Transaction()), receipt.Txhash(), "last receipt does not match last transaction")
			require.EqualValues(t, protocol.EXECUTION_RESULT_SUCCESS.String(), receipt.ExecutionResult().String(), "trigger transaction failed")
		})
}

func lastTxOf(block *services.GetBlockOutput) (tx *protocol.SignedTransaction) {
	txs := block.ClientResponse.SignedTransactionsIterator()
	for txs.HasNext() {
		tx = txs.NextSignedTransactions()
	}
	return
}

func lastReceiptOf(block *services.GetBlockOutput) (tx *protocol.TransactionReceipt) {
	receipts := block.ClientResponse.TransactionReceiptsIterator()
	for receipts.HasNext() {
		tx = receipts.NextTransactionReceipts()
	}
	return
}
