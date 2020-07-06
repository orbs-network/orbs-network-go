// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"bytes"
	"context"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCommitReputation_TransactionToElected(t *testing.T) {
	nodeTampered := testKeys.NodeAddressesForTests()[4]
	const maxruns = 100

	NewHarness().
		WithNumNodes(5).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			onGoingTamperer := network.TransportTamperer().Fail(func(data *adapter.TransportData) bool {
				return bytes.Equal(data.SenderNodeAddress, nodeTampered)
			})

			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 0)

			i := 0
			for ; i < maxruns; i++ {
				_, txHash := token.Transfer(ctx, 0, 10, 5, 6)
				network.WaitForTransactionInNodeState(ctx, txHash, 0)
				committee, misses := getCommitteeMisses(t, contract.GetAllCommitteeMisses(ctx, 0))
				m := findMissesOf(nodeTampered, committee, misses)
				if m == 3 {
					break
				}

			}
			require.NotEqual(t, maxruns, i, "failed to get 3 misses after %d passes", maxruns)

			onGoingTamperer.StopTampering(ctx)
			i = 0
			for ; i < maxruns; i++ {
				_, txHash := token.Transfer(ctx, 0, 10, 5, 6)
				network.WaitForTransactionInNodeState(ctx, txHash, 0)
				committee, misses := getCommitteeMisses(t, contract.GetAllCommitteeMisses(ctx, 0))
				m := findMissesOf(nodeTampered, committee, misses)
				if m == 0 {
					break
				}

			}
			require.NotEqual(t, maxruns, i, "failed to clear misses after %d passes", maxruns)
	})
}

func TestCommitReputation_LeavingCommitteeClearsMisses(t *testing.T) {
	nodeTamperedIndex := 4
	nodeTampered := testKeys.NodeAddressesForTests()[nodeTamperedIndex]

	NewHarness().
		WithNumNodes(5).
		WithManagementPollingInterval(20*time.Millisecond).
		WithLogFilters(log.DiscardAll()).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		Start(t, func(t testing.TB, ctx context.Context, network *Network) {
			network.TransportTamperer().Fail(func(data *adapter.TransportData) bool {
				return bytes.Equal(data.SenderNodeAddress, nodeTampered)
			})

			contract := callcontract.NewContractClient(network)
			token := network.DeployBenchmarkTokenContract(ctx, 0)

			for {
				_, txHash := token.Transfer(ctx, 0, 10, 3, 4)
				network.WaitForTransactionInNodeState(ctx, txHash, 0)
				argsArray, err := protocol.PackedOutputArgumentsToNatives(contract.GetMisses(ctx, 0, nodeTamperedIndex).QueryResult().RawOutputArgumentArrayWithHeader())
				require.NoError(t, err)
				miss := argsArray[0].(uint32)
				if miss == 1 {
					break
				}
			}

			// remove the "bad" validator from committee
			newRefTime := GenerateNewManagementReferenceTime(0)
			blockOfChange := setElectCommitteeAtAndWait(t, ctx, network, 0, newRefTime, 0, 1, 2, 3)
			network.WaitForBlock(ctx, blockOfChange+1) // need to be able to run query on block closed AFTER change

			// node still tampered, but now it is out of the committee for sure.
			argsArray, err := protocol.PackedOutputArgumentsToNatives(contract.GetMisses(ctx, 0, nodeTamperedIndex).QueryResult().RawOutputArgumentArrayWithHeader())
			require.NoError(t, err)
			miss := argsArray[0].(uint32)
			require.EqualValues(t, 0, miss, "should be cleared")
	})
}

func findMissesOf(nodeAddress primitives.NodeAddress, committee [][20]byte, misses []uint32) int {
	for i := 0;i < len(committee); i++ {
		if bytes.Equal(nodeAddress, committee[i][:]) {
			return int(misses[i])
		}
	}
	return -1
}

func getCommitteeMisses(t testing.TB, queryResponse *client.RunQueryResponse) (committee [][20]byte, misses []uint32) {
	argsArray, err := protocol.PackedOutputArgumentsToNatives(queryResponse.QueryResult().RawOutputArgumentArrayWithHeader())
	require.NoError(t, err)
	block := queryResponse.RequestResult().BlockHeight()
	committee = argsArray[0].([][20]byte)
	misses = argsArray[1].([]uint32)

	t.Logf("Committee at block %d", block)
	for i := 0;i < len(committee); i++ {
		t.Logf("#%d: Node %x Misses %d", i, committee[i], misses[i])
	}
	return
}
