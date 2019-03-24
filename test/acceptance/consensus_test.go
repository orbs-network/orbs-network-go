// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/internodesync"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TODO: add similar test for lean helix

func TestBenchmarkConsensus_LeaderGetsVotesBeforeNextBlock(t *testing.T) {
	newHarness().
		WithLogFilters(log.ExcludeField(internodesync.LogTag), log.ExcludeEntryPoint("BlockSync")).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS). // override default consensus algo
		WithMaxTxPerBlock(1).
		Start(t, func(t testing.TB, parent context.Context, network *NetworkHarness) {
			ctx, cancel := context.WithTimeout(parent, 1*time.Second)
			defer cancel()

			contract := network.DeployBenchmarkTokenContract(ctx, 5)

			committedTamper := network.TransportTamperer().Fail(testkit.BenchmarkConsensusMessage(consensus.BENCHMARK_CONSENSUS_COMMITTED))
			blockSyncTamper := network.TransportTamperer().Fail(testkit.BlockSyncMessage(gossipmessages.BLOCK_SYNC_AVAILABILITY_REQUEST)) // block sync discovery message so it does not add the blocks in a 'back door'
			committedLatch := network.TransportTamperer().LatchOn(testkit.BenchmarkConsensusMessage(consensus.BENCHMARK_CONSENSUS_COMMITTED))

			contract.TransferInBackground(ctx, 0, 0, 5, 6) // send a transaction so that network advances to block 1. the tamper prevents COMMITTED messages from reaching leader, so it doesn't move to block 2
			committedLatch.Wait()                          // wait for validator to try acknowledge that it reached block 1 (and fail)
			committedLatch.Wait()                          // wait for another consensus round (to make sure transaction(0) does not arrive after transaction(17) due to scheduling flakiness

			txHash := contract.TransferInBackground(ctx, 0, 17, 5, 6) // this should be included in block 2 which will not be closed until leader knows network is at block 2

			committedLatch.Wait()

			committedLatch.Remove()
			committedTamper.StopTampering(ctx) // this will allow COMMITTED messages to reach leader so that it can progress

			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 17, contract.GetBalance(ctx, 0, 6), "eventual getBalance result on leader")

			network.WaitForTransactionInNodeState(ctx, txHash, 1)
			require.EqualValues(t, 17, contract.GetBalance(ctx, 1, 6), "eventual getBalance result on non leader")

			blockSyncTamper.StopTampering(ctx)
		})
}
