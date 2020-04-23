// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCommitteeElections_OneElectionAndCheckReputationChanges(t *testing.T) {
	NewHarness().
		WithNumNodes(6).
		WithManagementPollingInterval(20*time.Millisecond).
		WithLogFilters(log.DiscardAll()).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			contract := callcontract.NewContractClient(network)
			network.WaitForBlock(ctx, 2)

			reputationCommitteeTerm0, _ := getCommitteeMisses(t, contract.GetAllCommitteeMisses(ctx, 0))
			require.Len(t, reputationCommitteeTerm0, 6, "number of addresses should equal original committee of 6")

			t.Log("elect 0,1,2,3")
			newRefTime := generateNewRefTime(0)
			blockOfChange := setElectCommitteeAtAndWait(t, ctx, network, 0, newRefTime, 0, 1, 2, 3)
			network.WaitForBlock(ctx, blockOfChange+1) // need to be able to run query on block closed AFTER change

			reputationCommitteeTerm1, _ := getCommitteeMisses(t, contract.GetAllCommitteeMisses(ctx, 0))
			require.Len(t, reputationCommitteeTerm1, 4, "number of addresses should equal new partail committee of 4")

			t.Log("test done, shutting down")
		})
}

func TestCommitteeElections_VerifyCommitteeSigns(t *testing.T) {
	NewHarness().
		WithNumNodes(6).
		WithManagementPollingInterval(20*time.Millisecond).
		WithLogFilters(log.DiscardAll()).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			_, txHash := token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 10, token.GetBalance(ctx, 0, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 0, []int{0, 1, 2, 3, 4, 5})

			t.Log("elect 0,1,2,3")
			newRefTime := generateNewRefTime(0)
			setElectCommitteeAtAndWait(t, ctx, network, 0, newRefTime, 0, 1, 2, 3)

			_, txHash = token.Transfer(ctx, 4, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 4)
			require.EqualValues(t, 20, token.GetBalance(ctx, 4, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 4, []int{0, 1, 2, 3})

			t.Log("test done, shutting down")
		})
}

func TestCommitteeElections_MultipleReElections(t *testing.T) {
	NewHarness().
		WithNumNodes(6).
		WithManagementPollingInterval(20*time.Millisecond).
		WithLogFilters(log.DiscardAll()).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			t.Log("elect 0,1,2,3")
			newRefTime := generateNewRefTime(0)
			setElectCommitteeAtAndWait(t, ctx, network, 0, newRefTime, 0, 1, 2, 3)

			t.Log("elect 1,2,3,4")
			newRefTime = generateNewRefTime(newRefTime)
			setElectCommitteeAtAndWait(t, ctx, network, 0, newRefTime, 1, 2, 3, 4)

			t.Log("elect 2,3,4,5")
			newRefTime = generateNewRefTime(newRefTime)
			setElectCommitteeAtAndWait(t, ctx, network, 1, newRefTime, 2, 3, 4, 5)

			_, txHash := token.Transfer(ctx, 3, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 3)
			require.EqualValues(t, 10, token.GetBalance(ctx, 3, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 3, []int{2, 3, 4, 5})

			t.Log("test done, shutting down")
		})
}

func TestCommitteeElections_AllNodesLoseElectionButReturn(t *testing.T) {
	NewHarness().
		WithNumNodes(8).
		WithManagementPollingInterval(20*time.Millisecond).
		WithLogFilters(log.DiscardAll()).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			t.Log("elect 0,1,2,3")
			newRefTime := generateNewRefTime(0)
			setElectCommitteeAtAndWait(t, ctx, network, 0, newRefTime, 0, 1, 2, 3)

			t.Log("elect 4,5,6,7 - entire first group loses")
			newRefTime = generateNewRefTime(newRefTime)
			setElectCommitteeAtAndWait(t, ctx, network, 0, newRefTime, 4, 5, 6, 7)

			_, txHash := token.Transfer(ctx, 4, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 10, token.GetBalance(ctx, 4, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 4, []int{4, 5, 6, 7})

			t.Log("elect 0,1,2,3 - first group returns")
			newRefTime = generateNewRefTime(newRefTime)
			setElectCommitteeAtAndWait(t, ctx, network, 4, newRefTime, 0, 1, 2, 3)

			_, txHash = token.Transfer(ctx, 3, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 3)
			require.EqualValues(t, 20, token.GetBalance(ctx, 3, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 3, []int{0, 1, 2, 3})

			t.Log("test done, shutting down")
		})
}

func TestCommitteeElections_GrowingNumberOfElected(t *testing.T) {
	NewHarness().
		WithNumNodes(7).
		WithManagementPollingInterval(50*time.Millisecond).
		WithLogFilters(log.DiscardAll()).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 5)

			t.Log("elect 0,1,2,3")
			newRefTime := generateNewRefTime(0)
			setElectCommitteeAtAndWait(t, ctx, network, 0, newRefTime, 0, 1, 2, 3)

			_, txHash := token.Transfer(ctx, 0, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 0)
			require.EqualValues(t, 10, token.GetBalance(ctx, 0, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 0, []int{0, 1, 2, 3})

			t.Log("elect 0,1,2,3,4,5,6")
			newRefTime = generateNewRefTime(newRefTime)
			setElectCommitteeAtAndWait(t, ctx, network, 0, newRefTime, 0, 1, 2, 3, 4, 5, 6)

			_, txHash = token.Transfer(ctx, 4, 10, 5, 6)
			network.WaitForTransactionInNodeState(ctx, txHash, 4)
			require.EqualValues(t, 20, token.GetBalance(ctx, 4, 6))
			verifyTxSignersAreFromGroup(t, ctx, contract.API, txHash, 4, []int{0, 1, 2, 3, 4, 5, 6})

			t.Log("test done, shutting down")
		})
}

func generateNewRefTime(oldRefTime primitives.TimestampSeconds) primitives.TimestampSeconds {
	now := primitives.TimestampSeconds(time.Now().Unix() + 1)
	if oldRefTime < now  {
		return now
	}
	return oldRefTime + 1
}

func verifyTxSignersAreFromGroup(t testing.TB, ctx context.Context, api callcontract.CallContractAPI, txHash primitives.Sha256, nodeIndex int, allowedIndexes []int) {
	response := api.GetTransactionReceiptProof(ctx, txHash, nodeIndex)
	signers, err := digest.GetBlockSignersFromReceiptProof(response.PackedProof())
	require.NoError(t, err, "failed getting signers from block proof")
	signerIndexes := testKeys.NodeAddressesForTestsToIndexes(signers)
	require.Subset(t, allowedIndexes, signerIndexes, "tx signers should be subset of allowed group")
}

func setElectCommitteeAtAndWait(t testing.TB, ctx context.Context, network *Network, currentCommitteeMemberId int, refTime primitives.TimestampSeconds, newCommitteeIds ...int) primitives.BlockHeight {
	var committee []primitives.NodeAddress
	for _, committeeIndex := range newCommitteeIds {
		committee = append(committee, testKeys.EcdsaSecp256K1KeyPairForTests(committeeIndex).NodeAddress())
	}

	currentBlockHeight, err := network.BlockPersistence(currentCommitteeMemberId).GetLastBlockHeight()
	require.NoError(t, err)

	err = network.committeeProvider.AddCommittee(refTime, committee)
	require.NoError(t, err)

	waitingBlock := currentBlockHeight + 1
	for waitingBlock < currentBlockHeight+50 {
		network.WaitForBlock(ctx, waitingBlock)
		bp, _ := network.BlockPersistence(currentCommitteeMemberId).GetLastBlock()
		if bp.TransactionsBlock.Header.ReferenceTime() >= refTime {
			return bp.TransactionsBlock.Header.BlockHeight()
		}
		waitingBlock = bp.TransactionsBlock.Header.BlockHeight()+1
	}
	require.False(t, true, "Error waited too much and failed")
	return 0
}
