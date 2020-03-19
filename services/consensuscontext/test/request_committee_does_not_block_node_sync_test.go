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


// audit mode execute
// sync on 2 blocks
// stuck on execution
// receive 10 commit blocks from consensus
// release execution
// fail on validate consensus - retrieve committee from old state
// robust - does not loop forever on request committee
// Recover FromOldStateQuery in consensusContext

const stateStorageHistorySnapshotNum = 5

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
				currentHeight := output.LastCommittedBlockHeight
				if currentHeight >= input.BlockHeight + stateStorageHistorySnapshotNum {
					return nil, errors.New(fmt.Sprintf("unsupported block height: block %d too old. currently at %d. keeping %d back", input.BlockHeight, currentHeight, stateStorageHistorySnapshotNum))
				}
				return &services.CallSystemContractOutput{
					OutputArgumentArray: &protocol.ArgumentArray{},
					CallResult: protocol.EXECUTION_RESULT_SUCCESS,
				}, nil
			})

			consensusAlgo.MockConsensusBlocksHandler.When("HandleBlockConsensus", mock.Any, mock.Any).Call(func(ctx context.Context, input *handlers.HandleBlockConsensusInput) (*handlers.HandleBlockConsensusOutput, error) {
				blockStorageHeight = 10
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

