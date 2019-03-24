// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/BenchmarkToken"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLeanHelix_CommitTransaction(t *testing.T) {
	newHarness().
		WithNumNodes(4).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *NetworkHarness) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			_, txHash := token.Transfer(ctx, 0, 17, 5, 6)

			network.WaitForTransactionInNodeState(ctx, txHash, 0)

			t.Log("finished waiting for tx")

			require.EqualValues(t, benchmarktoken.TOTAL_SUPPLY-17, token.GetBalance(ctx, 0, 5), "getBalance result for the sender on gateway node")
			require.EqualValues(t, 17, token.GetBalance(ctx, 0, 6), "getBalance result for the receiver on gateway node")

			t.Log("checking signers on the block proof")

			response := contract.API.GetTransactionReceiptProof(ctx, txHash, 0)
			signers, err := digest.GetBlockSignersFromReceiptProof(response.PackedProof())
			require.NoError(t, err)
			signerIndexes := testKeys.NodeAddressesForTestsToIndexes(signers)
			require.Subset(t, []int{0, 1, 2, 3}, signerIndexes, "block proof signers should be subset of first 4 test nodes")
			require.True(t, len(signerIndexes) >= 3, "block proof signers should include at least 3 nodes")

			t.Log("test done")
		})
}

func TestLeaderCommitsTransactionsAndSkipsInvalidOnes(t *testing.T) {
	newHarness().
		Start(t, func(t testing.TB, parent context.Context, network *NetworkHarness) {
			ctx, cancel := context.WithTimeout(parent, 2*time.Second)
			defer cancel()

			contract := network.DeployBenchmarkTokenContract(ctx, 5)

			// In benchmark consensus, leader is nodeIndex 0, validator is nodeIndex 1
			// In Lean Helix, leader and validators are random

			_, txHash1 := contract.Transfer(ctx, 0, 17, 5, 6)
			contract.InvalidTransfer(ctx, 0, 5, 6)
			_, txHash2 := contract.Transfer(ctx, 0, 22, 5, 6)

			t.Log("waiting for node 0")

			network.WaitForTransactionInNodeState(ctx, txHash1, 0)
			network.WaitForTransactionInNodeState(ctx, txHash2, 0)
			require.EqualValues(t, benchmarktoken.TOTAL_SUPPLY-39, contract.GetBalance(ctx, 0, 5), "getBalance result on leader")
			require.EqualValues(t, 39, contract.GetBalance(ctx, 0, 6), "getBalance result on leader")

			t.Log("waiting for node 1")

			network.WaitForTransactionInNodeState(ctx, txHash1, 1)
			network.WaitForTransactionInNodeState(ctx, txHash2, 1)
			require.EqualValues(t, benchmarktoken.TOTAL_SUPPLY-39, contract.GetBalance(ctx, 1, 5), "getBalance result on non leader")
			require.EqualValues(t, 39, contract.GetBalance(ctx, 1, 6), "getBalance result on non leader")
		})
}

func TestNonLeaderPropagatesTransactionsToLeader(t *testing.T) {
	newHarness().
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_BENCHMARK_CONSENSUS).
		Start(t, func(t testing.TB, parent context.Context, network *NetworkHarness) {
			ctx, cancel := context.WithTimeout(parent, 1*time.Second)
			defer cancel()

			contract := network.DeployBenchmarkTokenContract(ctx, 5)

			// leader is nodeIndex 0, validator is nodeIndex 1

			pausedTxForwards := network.TransportTamperer().Pause(testkit.TransactionRelayMessage(gossipmessages.TRANSACTION_RELAY_FORWARDED_TRANSACTIONS))
			txHash := contract.TransferInBackground(ctx, 1, 17, 5, 6)

			if err := network.BlockPersistence(0).GetBlockTracker().WaitForBlock(ctx, 2); err != nil {
				t.Errorf("failed waiting for block on node 0: %s", err)
			}
			if err := network.BlockPersistence(1).GetBlockTracker().WaitForBlock(ctx, 2); err != nil {
				t.Errorf("failed waiting for block on node 1: %s", err)
			}

			pausedTxForwards.StopTampering(ctx)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 17, contract.GetBalance(ctx, 0, 6), "eventual getBalance result on leader")
			network.WaitForTransactionInNodeState(ctx, txHash, 1)
			require.EqualValues(t, 17, contract.GetBalance(ctx, 1, 6), "eventual getBalance result on non leader")
		})
}

func TestLeaderCommitsTwoTransactionsInOneBlock(t *testing.T) {
	newHarness().Start(t, func(t testing.TB, parent context.Context, network *NetworkHarness) {
		ctx, cancel := context.WithTimeout(parent, 1*time.Second)
		defer cancel()

		contract := network.DeployBenchmarkTokenContract(ctx, 5)

		// leader is nodeIndex 0, validator is nodeIndex 1

		txHash1 := contract.TransferInBackground(ctx, 0, 17, 5, 6)
		txHash2 := contract.TransferInBackground(ctx, 0, 22, 5, 6)

		t.Log("waiting for node 0")

		network.WaitForTransactionInNodeState(ctx, txHash1, 0)
		network.WaitForTransactionInNodeState(ctx, txHash2, 0)
		require.EqualValues(t, benchmarktoken.TOTAL_SUPPLY-39, contract.GetBalance(ctx, 0, 5), "getBalance result on leader")
		require.EqualValues(t, 39, contract.GetBalance(ctx, 0, 6), "getBalance result on leader")

		t.Log("waiting for node 1")

		network.WaitForTransactionInNodeState(ctx, txHash1, 1)
		network.WaitForTransactionInNodeState(ctx, txHash2, 1)
		require.EqualValues(t, benchmarktoken.TOTAL_SUPPLY-39, contract.GetBalance(ctx, 1, 5), "getBalance result on non leader")
		require.EqualValues(t, 39, contract.GetBalance(ctx, 1, 6), "getBalance result on non leader")
	})
}
