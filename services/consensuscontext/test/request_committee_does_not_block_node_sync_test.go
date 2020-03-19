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
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"

	//"github.com/orbs-network/orbs-spec/types/go/protocol"

	//"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/with"
	//"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"testing"
)

func (h *harness) expectStateStorageNotRead() {
	h.stateStorage.When("ReadKeys", mock.Any, mock.Any).Return(&services.ReadKeysOutput{
		StateRecords: []*protocol.StateRecord{
			(&protocol.StateRecordBuilder{
				Key:   []byte{0x01},
				Value: []byte{0x02},
			}).Build(),
		},
	}, nil).Times(0)
}

// audit mode execute
// sync on 2 blocks
// stuck on execution
// receive 10 commit blocks from consensus
// release execution
// fail on validate consensus - retrieve committee from old state
// robust - does not loop forever on request committee
// Recover FromOldStateQuery in consensusContext

const STATE_STORAGE_HISTORY_SNAPSHOT_NUM = 5

func TestRequestCommittee_NonBlocking_NodeSync(t *testing.T)  {
		with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
			harness := newHarness(parent.Logger, true)
			done := make(chan struct{})
			blockStorage := &services.MockBlockStorage{}
			consensusAlgo := &services.MockConsensusAlgoLeanHelix{}
			blockStorageHeight := 0

			harness.stateStorage.When("GetLastCommittedBlockInfo", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.GetLastCommittedBlockInfoInput) (*services.GetLastCommittedBlockInfoOutput, error) {
				output, _ := blockStorage.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
				return &services.GetLastCommittedBlockInfoOutput{
					LastCommittedBlockHeight: output.LastCommittedBlockHeight,
				}, nil
			})

			harness.virtualMachine.When("CallSystemContract", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.CallSystemContractInput) (*services.CallSystemContractOutput, error) {
				output, _ := harness.stateStorage.GetLastCommittedBlockInfo(ctx, &services.GetLastCommittedBlockInfoInput{})
				fmt.Println(input.BlockHeight)
				currentHeight := output.LastCommittedBlockHeight
				if currentHeight >= input.BlockHeight + STATE_STORAGE_HISTORY_SNAPSHOT_NUM {
					return nil, errors.New(fmt.Sprintf("unsupported block height: block %d too old. currently at %d. keeping %d back", input.BlockHeight, currentHeight, STATE_STORAGE_HISTORY_SNAPSHOT_NUM))
				}
				return &services.CallSystemContractOutput{
					OutputArgumentArray: &protocol.ArgumentArray{},
					CallResult: protocol.EXECUTION_RESULT_SUCCESS,
				}, nil
			})

			consensusAlgo.MockConsensusBlocksHandler.When("HandleBlockConsensus", mock.Any, mock.Any).Call(func(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
				go func() {
					harness.service.RequestOrderingCommittee(ctx, &services.RequestCommitteeInput{
						CurrentBlockHeight: 1,
						RandomSeed:         0,
						MaxCommitteeSize:   22,
					})
					done <- struct{}{}
				} ()
				return nil, nil
			})

			blockStorage.Mock.When("GetLastCommittedBlockHeight", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.GetLastCommittedBlockHeightInput) (*services.GetLastCommittedBlockHeightOutput, error) {
				return &services.GetLastCommittedBlockHeightOutput{
					LastCommittedBlockHeight: primitives.BlockHeight(blockStorageHeight),
				}, nil
			})
			block := builders.BlockPair().WithHeight(1).WithEmptyLeanHelixBlockProof().Build()
			prevBlock := builders.BlockPair().WithHeight(0).WithEmptyLeanHelixBlockProof().Build()
			blockStorage.When("ValidateBlockForCommit", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.ValidateBlockForCommitInput) (*services.ValidateBlockForCommitOutput, error) {
				consensusAlgo.HandleBlockConsensus(ctx,	&handlers.HandleBlockConsensusInput{
					Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_ONLY,
					BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
					BlockPair:              block,
					PrevCommittedBlockPair: prevBlock,
				})

				return nil, nil
			})

			// start of flow "after receiving blocks chunk from peer"
			blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{BlockPair: block})

			select {
			case <-done:
				// test passed
			case <-ctx.Done():
				t.Fatalf("timed out waiting for sync flow to complete")
			}
		})
}

	//_, err := s.storage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{BlockPair: blockPair})

	//.MockBlockSyncHandler.HandleBlockSyncResponse().ValidateBlockForCommit()

	//consensusAlgo.When("HandleBlockConsensus", mock.Any, mock.Any).Call(func(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
	//
	//
	//})
	//
	//txPool := &services.MockTransactionPool{}
	//machine := &services.MockVirtualMachine{}
//	harness := newHarness()
//	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
//
//	})
//
//
//harness := test2.newSingleLhcNodeHarness().
//			withSyncNoCommitTimeout(10 * time.Millisecond).
//			withSyncCollectResponsesTimeout(10 * time.Millisecond).
//			withSyncCollectChunksTimeout(50 * time.Millisecond)



//}

//
////newSingleLhcNodeHarness
//func TestSyncPetitioner_Stress_CommitsDuringSync(t *testing.T) {
//	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
//		harness := test.newBlockStorageHarness(parent).
//			withSyncNoCommitTimeout(10 * time.Millisecond).
//			withSyncCollectResponsesTimeout(10 * time.Millisecond).
//			withSyncCollectChunksTimeout(50 * time.Millisecond)
//
//		const NUM_BLOCKS = 50
//		done := make(chan struct{})
//
//		harness.gossip.When("BroadcastBlockAvailabilityRequest", mock.Any, mock.Any).Call(func(ctx context.Context, input *gossiptopics.BlockAvailabilityRequestInput) (*gossiptopics.EmptyOutput, error) {
//			test.respondToBroadcastAvailabilityRequest(ctx, harness, input, NUM_BLOCKS, 7)
//			return nil, nil
//		})
//
//		harness.gossip.When("SendBlockSyncRequest", mock.Any, mock.Any).Call(func(ctx context.Context, input *gossiptopics.BlockSyncRequestInput) (*gossiptopics.EmptyOutput, error) {
//			if input.Message.SignedChunkRange.LastBlockHeight() >= NUM_BLOCKS {
//				done <- struct{}{}
//			}
//			respondToBlockSyncRequestWithConcurrentCommit(t, ctx, harness, input, NUM_BLOCKS)
//			return nil, nil
//		})
//
//		machine := &services.MockVirtualMachine{}
//		machine.When("CallSystemContract", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.CallSystemContractInput) (*services.CallSystemContractOutput, error) {
//			fmt.Println("unsupported block height")
//			return &services.CallSystemContractOutput{
//			}, errors.New("unsupported block height")
//		})
//		consensusContextService := consensuscontext.NewConsensusContext(harness.txPool, machine, harness.stateStorage, config.ForConsensusContextTests(false), harness.Logger, metric.NewRegistry())
//
//		harness.consensus.When("HandleBlockConsensus", mock.Any, mock.Any).Call(func(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
//
//			if input.Mode == handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_AND_UPDATE && input.PrevCommittedBlockPair != nil {
//				fmt.Println("ADFDSFSDF")
//				currHeight := input.BlockPair.TransactionsBlock.Header.BlockHeight()
//				prevHeight := input.PrevCommittedBlockPair.TransactionsBlock.Header.BlockHeight()
//				// audit mode + long execution ->
//				// consensus algo concurrently commits multiple blocks > state storage cache threshold
//				// no support for
//
//				consensusContextService.RequestOrderingCommittee(ctx, &services.RequestCommitteeInput{
//					CurrentBlockHeight: currHeight,
//					RandomSeed:         0,
//					MaxCommitteeSize:   22,
//				})
//				fmt.Println("123123")
//				if currHeight != prevHeight+1 {
//					done <- struct{}{}
//					require.Failf(t, "HandleBlockConsensus given invalid args", "called with height %d and prev height %d", currHeight, prevHeight)
//				}
//			}
//			return nil, nil
//		})
//
//		harness.start(ctx)
//
//		select {
//		case <-done:
//			// test passed
//		case <-ctx.Done():
//			t.Fatalf("timed out waiting for sync flow to complete")
//		}
//	})
//}
//
//// this would attempt to commit the same blocks at the same time from the sync flow and directly (simulating blocks arriving from consensus)
//func respondToBlockSyncRequestWithConcurrentCommit(t testing.TB, ctx context.Context, harness *test.harness, input *gossiptopics.BlockSyncRequestInput, availableBlocks int) {
//	response := builders.BlockSyncResponseInput().
//		WithFirstBlockHeight(input.Message.SignedChunkRange.FirstBlockHeight()).
//		WithLastBlockHeight(input.Message.SignedChunkRange.LastBlockHeight()).
//		WithLastCommittedBlockHeight(primitives.BlockHeight(availableBlocks)).
//		WithSenderNodeAddress(input.RecipientNodeAddress).Build()
//
//	go func() {
//		time.Sleep(time.Duration(rand.Intn(1000)) * time.Nanosecond)
//		_, err := harness.blockStorage.HandleBlockSyncResponse(ctx, response)
//		require.NoError(t, err, "failed handling block sync response")
//
//	}()
//
//	go func() {
//		time.Sleep(time.Duration(rand.Intn(1000)) * time.Nanosecond)
//		_, err := harness.blockStorage.CommitBlock(ctx, &services.CommitBlockInput{
//			BlockPair: response.Message.BlockPairs[0],
//		})
//		require.NoError(t, err, "failed committing first block in parallel to sync")
//		_, err = harness.blockStorage.CommitBlock(ctx, &services.CommitBlockInput{
//			BlockPair: response.Message.BlockPairs[1],
//		})
//		require.NoError(t, err, "failed committing second block in parallel to sync")
//
//	}()
//}
