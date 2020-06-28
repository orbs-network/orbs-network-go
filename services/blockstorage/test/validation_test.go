// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidateBlockWithValidProtocolVersion(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		harness := newBlockStorageHarness(parent).
			withSyncBroadcast(1).
			withValidateConsensusAlgos(1).
			start(ctx)
		block := builders.BlockPair().Build()
		var prevBlock *protocol.BlockPairContainer
		_, err := harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{BlockPair: block, PrevBlockPair: prevBlock})
		require.NoError(t, err, "block should be valid")
	})
}

func TestValidateBlockWithInvalidProtocolVersion(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		harness := newBlockStorageHarness(parent).
			allowingErrorsMatching("protocol version mismatch in.*").
			withSyncBroadcast(1).
			expectValidateConsensusAlgos().
			start(ctx)
		block := builders.BlockPair().Build()
		var prevBlock *protocol.BlockPairContainer

		errorProtocolVersion := config.MAXIMAL_PROTOCOL_VERSION_SUPPORTED_VALUE+100
		expectedTxErrMsg := fmt.Sprintf("protocol version (%d) higher than maximal supported (%d) in transactions block header", errorProtocolVersion, config.MAXIMAL_PROTOCOL_VERSION_SUPPORTED_VALUE)
		expectedRxErrMsg := fmt.Sprintf("protocol version (%d) higher than maximal supported (%d) in results block header", errorProtocolVersion, config.MAXIMAL_PROTOCOL_VERSION_SUPPORTED_VALUE)

		block.TransactionsBlock.Header.MutateProtocolVersion(errorProtocolVersion)
		_, err := harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{BlockPair: block, PrevBlockPair: prevBlock})
		require.EqualError(t, err, expectedTxErrMsg, "tx protocol was mutated, should fail")

		block = builders.BlockPair().Build()
		block.ResultsBlock.Header.MutateProtocolVersion(errorProtocolVersion)
		_, err = harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{BlockPair: block, PrevBlockPair: prevBlock})
		require.EqualError(t, err, expectedRxErrMsg, "rx protocol was mutated, should fail")

		block = builders.BlockPair().Build()
		block.TransactionsBlock.Header.MutateProtocolVersion(errorProtocolVersion)
		block.ResultsBlock.Header.MutateProtocolVersion(errorProtocolVersion)
		_, err = harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{BlockPair: block, PrevBlockPair: prevBlock})
		require.EqualError(t, err, expectedTxErrMsg, "tx and rx protocol was mutated, should fail")
	})
}

func TestValidateBlockWithValidHeight(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		harness := newBlockStorageHarness(parent).
			withSyncBroadcast(1).
			withCommitStateDiff(1).
			withValidateConsensusAlgos(1).
			start(ctx)

		harness.commitBlock(ctx, builders.BlockPair().Build())

		block := builders.BlockPair().WithHeight(2).Build()
		prevBlockHeight := int(block.ResultsBlock.Header.BlockHeight() - 1)
		prevBlock := harness.getBlock(prevBlockHeight)
		_, err := harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{BlockPair: block, PrevBlockPair: prevBlock})
		require.NoError(t, err, "happy flow")
	})
}

func TestValidateBlockWithInvalidHeight(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, parent *with.ConcurrencyHarness) {
		harness := newBlockStorageHarness(parent).
			withSyncBroadcast(1).
			withCommitStateDiff(1).
			withValidateConsensusAlgos(1).
			start(ctx)

		harness.commitBlock(ctx, builders.BlockPair().Build())

		block := builders.BlockPair().WithHeight(2).Build()
		prevBlockHeight := int(block.ResultsBlock.Header.BlockHeight() - 1)
		prevBlock := harness.getBlock(prevBlockHeight)

		block.TransactionsBlock.Header.MutateBlockHeight(998)
		_, err := harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{BlockPair: block, PrevBlockPair: prevBlock})
		require.EqualError(t, err, "block pair height mismatch. transactions height is 998, results height is 2", "tx block height was mutate, expected an error")

		block.ResultsBlock.Header.MutateBlockHeight(999)
		_, err = harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{BlockPair: block, PrevBlockPair: prevBlock})
		require.EqualError(t, err, "block pair height mismatch. transactions height is 998, results height is 999", "rx block height was mutate, expected an error")

		block.TransactionsBlock.Header.MutateBlockHeight(999)
		_, err = harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{BlockPair: block, PrevBlockPair: prevBlock})
		require.EqualError(t, err, "block height is 999, expected 2", "tx & rx block height was mutate, expected an error")
	})
}
