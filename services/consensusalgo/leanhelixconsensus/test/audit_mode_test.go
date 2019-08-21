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
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestHandleBlockConsensus_ExecutesBlocksYoungerThanThreshold_AndModeIsVerify(t *testing.T) {
	t.Skip("This test is skipped because we need to build a LeanHelixBlockProof that passes validateBlockConsensus, see issue: https://github.com/orbs-network/orbs-network-go/issues/1174 ")
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {

		h := newLeanHelixServiceHarness(5*time.Minute).start(parent, ctx)

		block := builders.BlockPair().WithHeight(1).WithEmptyLeanHelixBlockProof().Build()
		prevBlock := builders.BlockPair().WithHeight(0).WithEmptyLeanHelixBlockProof().Build()

		vrb := &services.ValidateResultsBlockInput{
			CurrentBlockHeight: block.TransactionsBlock.Header.BlockHeight(),
			ResultsBlock:       block.ResultsBlock,
			PrevBlockHash:      block.TransactionsBlock.Header.PrevBlockHashPtr(),
			TransactionsBlock:  block.TransactionsBlock,
			PrevBlockTimestamp: prevBlock.TransactionsBlock.Header.Timestamp()}

		h.consensusContext.When("ValidateResultsBlock", mock.Any, vrb).
			Return(&services.ValidateResultsBlockOutput{}, nil).Times(1)

		h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_ONLY,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              block,
			PrevCommittedBlockPair: prevBlock,
		})

		_, err := h.consensusContext.Verify()
		require.NoError(t, err, "Consensus Context not invoked as expected")
	})
}

func TestHandleBlockConsensus_DoesNotExecuteBlocksOlderThanThreshold_AndModeIsVerify(t *testing.T) {
	t.Skip("This test is skipped because we need to build a LeanHelixBlockProof that passes validateBlockConsensus, see issue: https://github.com/orbs-network/orbs-network-go/issues/1174 ")
	test.WithConcurrencyHarness(t, func(ctx context.Context, parent *test.ConcurrencyHarness) {
		h := newLeanHelixServiceHarness(0).start(parent, ctx)

		block := builders.BlockPair().WithTimestampAheadBy(-1 * time.Nanosecond).WithHeight(1).WithEmptyLeanHelixBlockProof().Build()

		h.consensusContext.Never("ValidateResultsBlock", mock.Any, mock.Any)

		h.consensus.HandleBlockConsensus(ctx, &handlers.HandleBlockConsensusInput{
			Mode:                   handlers.HANDLE_BLOCK_CONSENSUS_MODE_VERIFY_ONLY,
			BlockType:              protocol.BLOCK_TYPE_BLOCK_PAIR,
			BlockPair:              block,
			PrevCommittedBlockPair: nil,
		})

		_, err := h.consensusContext.Verify()
		require.NoError(t, err, "Consensus Context not invoked as expected")
	})
}
