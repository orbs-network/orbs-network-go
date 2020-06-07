// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"errors"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/consensuscontext"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"testing"
	"time"
)


// Node Sync assumes consensus verifies block without blocking
// Start syncing - in parallel to consensus service progressing
// During HandleBlockConsensus and before block proof verification, commit blocks from consensus:
// Example: during block execution check (audit mode) receive "numOfStateRevisionsToRetain" commits from consensus
// Calling old state for committee fails - too far back (out of stateStorage cache reach)
// Recover from "Old State" query (consensusContext does not poll forever)
func TestSyncPetitioner_ConsensusVerify_NonBlocking(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		harness := newBlockStorageHarness(parent).
			withSyncBroadcast(1).
			withSyncNoCommitTimeout(10 * time.Millisecond).
			withSyncCollectResponsesTimeout(10 * time.Millisecond).
			withSyncCollectChunksTimeout(50 * time.Millisecond).
			withBlockSyncDescendingEnabled(false). // => ascending order
			allowingErrorsMatching("FORK!! block already in storage, timestamp mismatch")

		testSyncPetitionerConsensusVerifyNonBlocking(ctx, t, harness)
	})
}

func TestSyncPetitioner_ConsensusVerify_NonBlocking_Descending(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		harness := newBlockStorageHarness(parent).
			withSyncBroadcast(1).
			withSyncNoCommitTimeout(10 * time.Millisecond).
			withSyncCollectResponsesTimeout(10 * time.Millisecond).
			withSyncCollectChunksTimeout(50 * time.Millisecond).
			withBlockSyncDescendingEnabled(true). // => descending order
			allowingErrorsMatching("FORK!! block already in storage, timestamp mismatch")

		testSyncPetitionerConsensusVerifyNonBlocking(ctx, t, harness)
	})
}

func testSyncPetitionerConsensusVerifyNonBlocking(ctx context.Context, t *testing.T, harness *harness) {

	const NUM_BLOCKS = 4
	blocks := generateInMemoryBlockChain(NUM_BLOCKS)

	numOfStateRevisionsToRetain := 2
	virtualMachine := &services.MockVirtualMachine{}
	cfg := config.ForConsensusContextTests(false)
	management := &services.MockManagement{}
	management.When("GetGenesisReference", mock.Any, mock.Any).Return(&services.GetGenesisReferenceOutput{CurrentReference: 5000, GenesisReference: 0,}, nil)
	harness.management.When("GetCurrentReference", mock.Any, mock.Any).Return(&services.GetCurrentReferenceOutput{CurrentReference: primitives.TimestampSeconds(time.Now().Unix())}, nil)

	consensusContext := consensuscontext.NewConsensusContext(harness.txPool, virtualMachine, harness.stateStorage, management, cfg, harness.Logger, metric.NewRegistry())

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	committedBlockHeights := make(chan primitives.BlockHeight, 10)
	done := make(chan struct{})
	simulatedCommitsTarget := numOfStateRevisionsToRetain + 1

	harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Call(func(ctx context.Context, input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
		respondToBroadcastAvailabilityRequest(ctx, harness, input, NUM_BLOCKS, 1)
		return nil, nil
	})

	harness.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Call(func(ctx context.Context, input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
		respondToBlockSyncRequest(ctx, harness, input, blocks, harness.config.syncBatchSize)
		return nil, nil
	})

	harness.stateStorage.When("GetLastCommittedBlockInfo", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.GetLastCommittedBlockInfoInput) (*services.GetLastCommittedBlockInfoOutput, error) {
		output := harness.getLastBlockHeight(ctx, t)
		return &services.GetLastCommittedBlockInfoOutput{
			BlockHeight: output.LastCommittedBlockHeight,
		}, nil
	})

	virtualMachine.When("CallSystemContract", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.CallSystemContractInput) (*services.CallSystemContractOutput, error) {
		output, _ := harness.stateStorage.GetLastCommittedBlockInfo(ctx, &services.GetLastCommittedBlockInfoInput{})
		currentHeight := output.BlockHeight
		if currentHeight >= input.BlockHeight + primitives.BlockHeight(numOfStateRevisionsToRetain) {
			return nil, errors.New(fmt.Sprintf("unsupported block height: block %d too old. currently at %d. keeping %d back", input.BlockHeight, currentHeight, numOfStateRevisionsToRetain))
		}
		return &services.CallSystemContractOutput{
			OutputArgumentArray: &protocol.ArgumentArray{},
			CallResult:          protocol.EXECUTION_RESULT_SUCCESS,
		}, nil
	})

	harness.consensus.When("HandleBlockConsensus", mock.Any, mock.Any).Call(func(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
		if input.Mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE  {
			simulateConsensusCommits(ctx, harness, blocks, committedBlockHeights, simulatedCommitsTarget)
			simulateVerifyBlockConsensus(ctx, t, consensusContext, input.BlockPair.TransactionsBlock.Header.BlockHeight(), done)
		}
		return nil, nil
	})

	harness.start(ctx)

	select {
	case <-done:
		// test passed
	case <-timeoutCtx.Done():
		t.Fatalf("timed out waiting for sync flow to recover")
	}
}


func simulateConsensusCommits(ctx context.Context, harness *harness, blocks []*protocol.BlockPairContainer, committedBlockHeights chan primitives.BlockHeight, target int) {
	for i := 0; i < target; i++ {
		_, err := harness.commitBlock(ctx, blocks[i])
		if err == nil {
			committedBlockHeights <- blocks[i].ResultsBlock.Header.BlockHeight()
		}
	}
}

func simulateVerifyBlockConsensus(ctx context.Context, tb testing.TB, consensusContext services.ConsensusContext, currentBlockHeight primitives.BlockHeight, done chan struct{}) {
	go func() {
		consensusContext.RequestOrderingCommittee(ctx, &services.RequestCommitteeInput{
			CurrentBlockHeight: currentBlockHeight,
			RandomSeed:         0,
			MaxCommitteeSize:   4,
		})
		done <- struct{}{}
	}()
}
