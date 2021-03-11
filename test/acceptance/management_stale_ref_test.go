package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const committeeGracePeriod = 2 * time.Second

func TestStaleManagementRef(t *testing.T) {
	NewHarness().
		WithConfigOverride(config.NodeConfigKeyValue{Key: config.COMMITTEE_GRACE_PERIOD, Value: config.NodeConfigValue{DurationValue: committeeGracePeriod}}).
		WithNumNodes(6).
		WithManagementPollingInterval(20*time.Millisecond).
		WithLogFilters(log.DiscardAll()).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			response, txHash := token.Transfer(ctx, 0, 17, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.Equal(t, response.TransactionStatus(), protocol.TRANSACTION_STATUS_COMMITTED)
			require.EqualValues(t, 17, token.GetBalance(ctx, 0, 6))
			txs, err := network.BlockPersistence(0).GetTransactionsBlock(response.RequestResult().BlockHeight())
			require.NoError(t, err)
			require.EqualValues(t, 2, txs.Header.NumSignedTransactions(), "should have 2 tx : transfer + trigger")

			t.Log("set RefTime To Now")
			now := time.Now()
			refTime := primitives.TimestampSeconds(now.Unix() + 1)
			err = network.committeeProvider.AddSubscription(refTime, true, 10, 100)
			require.NoError(t, err)

			changedBlock, err2 := network.WaitForManagementChange(ctx, 0, refTime)
			require.NoError(t, err2)

			// Wait for time to pass committee grace
			blockTimeDiff := committeeGracePeriod * 2
			blockTime := primitives.TimestampNano(now.UnixNano() + int64(blockTimeDiff))
			limit := blockTimeDiff * 2
			blockHeight := waitForBlockTime(t, ctx, network, changedBlock, blockTime, limit)
			t.Log("reached target blockTime passed committee grace period ", uint64(blockHeight))

			response, _ = token.Transfer(ctx, 0, 17, 5, 6)
			require.Equal(t, response.TransactionStatus(), protocol.TRANSACTION_STATUS_REJECTED_GLOBAL_PRE_ORDER) // rejected because committee is no longer active
			require.EqualValues(t, 17, token.GetBalance(ctx, 0, 6))
			txs, err3 := network.BlockPersistence(0).GetTransactionsBlock(response.RequestResult().BlockHeight())
			require.NoError(t, err3)
			require.EqualValues(t, 1, txs.Header.NumSignedTransactions(), "should have 1 tx : trigger")

			t.Log("test done, shutting down")
		})
}

func waitForBlockTime(t testing.TB, ctx context.Context, network *Network,
	startBlock primitives.BlockHeight, blockTime primitives.TimestampNano,
	limit time.Duration) primitives.BlockHeight {
	waitingBlock := startBlock + 1
	targetBlockTimedOut := time.After(limit)
	for {
		select {
		case <-targetBlockTimedOut:
			t.FailNow()
		case <-ctx.Done():
			return 0
		default:
			network.WaitForBlock(ctx, waitingBlock)
			bp, _ := network.BlockPersistence(0).GetLastBlock()
			if bp.TransactionsBlock.Header.Timestamp() >= blockTime {
				return bp.TransactionsBlock.Header.BlockHeight()
			}
			waitingBlock = bp.TransactionsBlock.Header.BlockHeight() + 1
		}
	}
	return 0
}
