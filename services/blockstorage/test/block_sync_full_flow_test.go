// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestSyncPetitioner_CompleteSyncFlow(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).
			withSyncNoCommitTimeout(200 * time.Millisecond).
			withSyncCollectResponsesTimeout(50 * time.Millisecond).
			withSyncCollectChunksTimeout(50 * time.Millisecond)

		const NUM_BLOCKS = 4

		resultsForVerification := newSyncFlowSummary()

		harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Call(func(ctx context.Context, input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
			respondToBroadcastAvailabilityRequest(t, ctx, harness, input, NUM_BLOCKS, 7, 8)
			return nil, nil
		})

		harness.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Call(func(ctx context.Context, input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
			resultsForVerification.logBlockSyncRequest(input)
			requireBlockSyncRequestConformsToBlockAvailabilityResponse(t, input, NUM_BLOCKS, 7, 8)
			respondToBlockSyncRequest(ctx, harness, input, NUM_BLOCKS)
			return nil, nil
		})

		harness.consensus.Reset().When("HandleBlockConsensus", mock.Any, mock.Any).Call(func(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
			resultsForVerification.logHandleBlockConsensusCalls(t, input, NUM_BLOCKS)
			requireValidHandleBlockConsensusMode(t, input.Mode)
			return nil, nil
		})

		harness.start(ctx)

		passed := test.Eventually(2*time.Second, func() bool { // wait for sync flow to complete successfully:
			resultsForVerification.Lock()
			defer resultsForVerification.Unlock()
			if !resultsForVerification.didUpdateConsensusAboutHeightZero {
				return false
			}
			for i := primitives.BlockHeight(1); i < NUM_BLOCKS; i++ {
				if !resultsForVerification.blocksSentBySource[i] || !resultsForVerification.blocksReceivedByConsensus[i] {
					return false
				}
			}
			return true
		})
		require.Truef(t, passed, "timed out waiting for passing conditions: %+v", resultsForVerification)
	})
}

func requireValidHandleBlockConsensusMode(t *testing.T, mode handlers.HandleBlockConsensusMode) {
	require.Contains(t, []handlers.HandleBlockConsensusMode{ // require mode is one of two expected
		handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY,
		handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE,
	}, mode, "consensus updates must be update or update+verify")
}

type syncFlowResults struct {
	sync.Mutex
	blocksSentBySource                map[primitives.BlockHeight]bool
	blocksReceivedByConsensus         map[primitives.BlockHeight]bool
	didUpdateConsensusAboutHeightZero bool
}

func newSyncFlowSummary() *syncFlowResults {
	return &syncFlowResults{
		blocksSentBySource:        make(map[primitives.BlockHeight]bool),
		blocksReceivedByConsensus: make(map[primitives.BlockHeight]bool),
	}
}

func (s *syncFlowResults) logBlockSyncRequest(input *gossiptopics.BlockSyncRequestInput) {
	s.Lock()
	defer s.Unlock()
	for i := input.Message.SignedChunkRange.FirstBlockHeight(); i <= input.Message.SignedChunkRange.LastBlockHeight(); i++ {
		s.blocksSentBySource[i] = true
	}
}

func (s *syncFlowResults) logHandleBlockConsensusCalls(t *testing.T, input *handlers.HandleBlockConsensusInput, availableBlocks primitives.BlockHeight) {
	s.Lock()
	defer s.Unlock()
	switch input.Mode {
	case handlers.HANDLE_BLOCK_CONSENSUS_MODE_UPDATE_ONLY:
		if input.BlockPair == nil {
			s.didUpdateConsensusAboutHeightZero = true
		}
	case handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE:
		require.Condition(t, func() (success bool) {
			return input.BlockPair.TransactionsBlock.Header.BlockHeight() >= 1 && input.BlockPair.TransactionsBlock.Header.BlockHeight() <= availableBlocks
		}, "validated block must be between 1 and total")
		s.blocksReceivedByConsensus[input.BlockPair.TransactionsBlock.Header.BlockHeight()] = true
	}
}

func respondToBlockSyncRequest(ctx context.Context, harness *harness, input *gossiptopics.BlockSyncRequestInput, availableBlocks int) {
	response := builders.BlockSyncResponseInput().
		WithFirstBlockHeight(input.Message.SignedChunkRange.FirstBlockHeight()).
		WithLastBlockHeight(input.Message.SignedChunkRange.LastBlockHeight()).
		WithLastCommittedBlockHeight(primitives.BlockHeight(availableBlocks)).
		WithSenderNodeAddress(input.RecipientNodeAddress).Build()
	go harness.blockStorage.HandleBlockSyncResponse(ctx, response)
}

func requireBlockSyncRequestConformsToBlockAvailabilityResponse(t *testing.T, input *gossiptopics.BlockSyncRequestInput, availableBlocks primitives.BlockHeight, sources ...int) {
	sourceAddresses := make([]primitives.NodeAddress, 0, len(sources))
	for _, sourceIndex := range sources {
		sourceAddresses = append(sourceAddresses, keys.EcdsaSecp256K1KeyPairForTests(sourceIndex).NodeAddress())
	}
	require.Contains(t, sourceAddresses, input.RecipientNodeAddress, "request is not consistent with my BlockAvailabilityResponse, the nodes accessed must be in %v", sources)

	firstRequestedBlock := input.Message.SignedChunkRange.FirstBlockHeight()
	require.Conditionf(t, func() (success bool) {
		return firstRequestedBlock >= 1 && firstRequestedBlock <= availableBlocks
	}, "request is not consistent with my BlockAvailabilityResponse, first requested block must be between 1 and total (%d) but was %d", availableBlocks, firstRequestedBlock)

	lastRequestedBlock := input.Message.SignedChunkRange.LastBlockHeight()
	require.Conditionf(t, func() (success bool) {
		return lastRequestedBlock >= firstRequestedBlock && lastRequestedBlock <= availableBlocks
	}, "request is not consistent with my BlockAvailabilityResponse, last requested block must be between first (%d) and total (%d) but was %d", firstRequestedBlock, availableBlocks, lastRequestedBlock)
}

func respondToBroadcastAvailabilityRequest(t *testing.T, ctx context.Context, harness *harness, requestInput *gossiptopics.BlockAvailabilityRequestInput, availableBlocks primitives.BlockHeight, sources ...int) {
	if harness.blockStorage == nil {
		return // protect against edge condition where harness did not finish initializing and sync has started
	}

	firstBlockHeight := requestInput.Message.SignedBatchRange.FirstBlockHeight()
	if firstBlockHeight > availableBlocks {
		return
	}

	for _, sourceAddressIndex := range sources {
		response := builders.BlockAvailabilityResponseInput().
			WithLastCommittedBlockHeight(primitives.BlockHeight(availableBlocks)).
			WithFirstBlockHeight(firstBlockHeight).
			WithLastBlockHeight(primitives.BlockHeight(availableBlocks)).
			WithSenderNodeAddress(keys.EcdsaSecp256K1KeyPairForTests(sourceAddressIndex).NodeAddress()).Build()
		go harness.blockStorage.HandleBlockAvailabilityResponse(ctx, response)
	}

}
