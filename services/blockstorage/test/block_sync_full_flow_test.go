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
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestSyncPetitioner_CompleteSyncFlow(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		harness := newBlockStorageHarness(parent).
			withSyncNoCommitTimeout(200 * time.Millisecond).
			withSyncCollectResponsesTimeout(50 * time.Millisecond).
			withSyncCollectChunksTimeout(50 * time.Millisecond).
			withBlockSyncDescendingActivationDate(time.Now().AddDate(0, 1, 0).Format(time.RFC3339)) // ensures activation date in the future => ascending order

			testSyncPetitionerCompleteSyncFlow(ctx, t, harness)	})
}

func TestSyncPetitioner_CompleteSyncFlowDescending(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		harness := newBlockStorageHarness(parent).
			withSyncNoCommitTimeout(200 * time.Millisecond).
			withSyncCollectResponsesTimeout(50 * time.Millisecond).
			withSyncCollectChunksTimeout(50 * time.Millisecond).
			withBatchSize(3).
			withBlockSyncDescendingActivationDate(time.Now().AddDate(0, -1, 0).Format(time.RFC3339)) // ensures activation date in the past => descending order

		testSyncPetitionerCompleteSyncFlow(ctx, t, harness)
	})
}


func testSyncPetitionerCompleteSyncFlow(ctx context.Context, t *testing.T, harness *harness) {
	const NUM_BLOCKS = 4
	blockChain := generateInMemoryBlockChain(NUM_BLOCKS)

	resultsForVerification := newSyncFlowSummary()

	harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Call(func(ctx context.Context, input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
		respondToBroadcastAvailabilityRequest(ctx, harness, input, NUM_BLOCKS, 7, 8)
		return nil, nil
	})

	harness.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Call(func(ctx context.Context, input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
		resultsForVerification.logBlockSyncRequest(input, NUM_BLOCKS)
		requireBlockSyncRequestConformsToBlockAvailabilityResponse(t, input, NUM_BLOCKS, 7, 8)
		respondToBlockSyncRequest(ctx, harness, input, blockChain, harness.config.syncBatchSize)

		return nil, nil
	})

	harness.management.When("GetCurrentReference", mock.Any, mock.Any).Return(&services.GetCurrentReferenceOutput{CurrentReference: primitives.TimestampSeconds(time.Now().Unix())}, nil)

	harness.consensus.When("HandleBlockConsensus", mock.Any, mock.Any).Call(func(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
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

}

func generateInMemoryBlockChain(numBlocks int) []*protocol.BlockPairContainer {
	var blocks []*protocol.BlockPairContainer
	var prevBlock *protocol.BlockPairContainer
	for i := 1; i <= numBlocks; i++ {
		blockTime := time.Unix(1550394190000000000+int64(i), 0) // deterministic block creation in the past based on block height
		blockPair := builders.BlockPair().WithHeight(primitives.BlockHeight(i)).WithBlockCreated(blockTime).WithPrevBlock(prevBlock).Build()
		prevBlock = blockPair
		blocks = append(blocks, blockPair)
	}
	return blocks
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

func (s *syncFlowResults) logBlockSyncRequest(input *gossiptopics.BlockSyncRequestInput, availableBlocks primitives.BlockHeight) {
	s.Lock()
	defer s.Unlock()
	fromBlock := input.Message.SignedChunkRange.FirstBlockHeight()
	toBlock := input.Message.SignedChunkRange.LastBlockHeight()
	if fromBlock == 0 {
		fromBlock = 1
		toBlock = availableBlocks
	}
	for i := fromBlock; i <= toBlock; i++ {
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


func requireBlockSyncRequestConformsToBlockAvailabilityResponse(t *testing.T, input *gossiptopics.BlockSyncRequestInput, availableBlocks primitives.BlockHeight, sources ...int) {
	sourceAddresses := make([]primitives.NodeAddress, 0, len(sources))
	for _, sourceIndex := range sources {
		sourceAddresses = append(sourceAddresses, keys.EcdsaSecp256K1KeyPairForTests(sourceIndex).NodeAddress())
	}
	require.Contains(t, sourceAddresses, input.RecipientNodeAddress, "request is not consistent with my BlockAvailabilityResponse, the nodes accessed must be in %v", sources)

	firstRequestedBlock := input.Message.SignedChunkRange.FirstBlockHeight()
	lastRequestedBlock := input.Message.SignedChunkRange.LastBlockHeight()
	blocksOrder := input.Message.SignedChunkRange.BlocksOrder()

	if blocksOrder == gossipmessages.SYNC_BLOCKS_ORDER_DESCENDING {
		require.Conditionf(t, func() (success bool) {
			return (lastRequestedBlock >= 1 && lastRequestedBlock <= availableBlocks) && (firstRequestedBlock == 0 || (firstRequestedBlock >= lastRequestedBlock && firstRequestedBlock <= availableBlocks))
		}, "request is not consistent with my BlockAvailabilityResponse: first (%d) and last (%d) requested block must be smaller than total (%d); either first requested block is unknown(0) or first must be larger than last", firstRequestedBlock, lastRequestedBlock, availableBlocks )

	} else {
		require.Conditionf(t, func() (success bool) {
			return firstRequestedBlock >= 1 && firstRequestedBlock <= availableBlocks
		}, "request is not consistent with my BlockAvailabilityResponse, first requested block must be between 1 and total (%d) but was %d", availableBlocks, firstRequestedBlock)

		require.Conditionf(t, func() (success bool) {
			return lastRequestedBlock >= firstRequestedBlock && lastRequestedBlock <= availableBlocks
		}, "request is not consistent with my BlockAvailabilityResponse, last requested block must be between first (%d) and total (%d) but was %d", firstRequestedBlock, availableBlocks, lastRequestedBlock)

	}
}
