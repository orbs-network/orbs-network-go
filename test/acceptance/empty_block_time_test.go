// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package acceptance

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test/acceptance/callcontract"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/protocol/consensus"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestIncomingTransactionTriggersExactlyOneBlock(t *testing.T) {
	NewHarness().
		WithSetup(func(ctx context.Context, network *Network) {
			// set current reference time to now for node sync verifications
			newRefTime := GenerateNewManagementReferenceTime(0)
			err := network.committeeProvider.AddCommittee(newRefTime, testKeys.NodeAddressesForTests()[1:5])
			require.NoError(t, err)
		}).
		WithEmptyBlockTime(1*time.Second).
		Start(t, func(tb testing.TB, ctx context.Context, network *Network) {
			contract := callcontract.NewContractClient(network)
			time.Sleep(100 * time.Millisecond)
			heightBeforeTx, _ := network.BlockPersistence(0).GetLastBlockHeight()

			_, txHash := contract.Transfer(ctx, 0, 43, 5, 6)
			network.WaitForTransactionInState(ctx, txHash)

			heightAfterTx, _ := network.BlockPersistence(0).GetLastBlockHeight()

			require.Equal(t, uint64(heightBeforeTx)+1, uint64(heightAfterTx), "incoming transaction triggered closure of more than one block")
		})
}

func TestIncomingTransactionTriggersImmediateBlockClosure(t *testing.T) {
	NewHarness().
		WithSetup(func(ctx context.Context, network *Network) {
			// set current reference time to now for node sync verifications
			newRefTime := GenerateNewManagementReferenceTime(0)
			err := network.committeeProvider.AddCommittee(newRefTime, testKeys.NodeAddressesForTests()[1:5])
			require.NoError(t, err)
		}).
		WithEmptyBlockTime(1*time.Hour).
		WithConsensusAlgos(consensus.CONSENSUS_ALGO_TYPE_LEAN_HELIX).
		WithLogFilters(log.ExcludeEntryPoint("BlockSync")).
		Start(t, func(tb testing.TB, ctx context.Context, network *Network) {
			contract := callcontract.NewContractClient(network)

			network.WaitForBlock(ctx, 1) // wait for network to start closing blocks

			_, txHash := contract.Transfer(ctx, 0, 43, 5, 6)

			network.WaitForTransactionInState(ctx, txHash)

		})
}
