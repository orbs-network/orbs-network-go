// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidateBlockWithValidProtocolVersion(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).
			withSyncBroadcast(1).
			withValidateConsensusAlgos(1).
			start(ctx)
		block := builders.BlockPair().Build()

		_, err := harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{block})
		require.NoError(t, err, "block should be valid")
	})
}

func TestValidateBlockWithInvalidProtocolVersion(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).allowingErrorsMatching("protocol version mismatch in.*").withSyncBroadcast(1).start(ctx)
		block := builders.BlockPair().Build()

		block.TransactionsBlock.Header.MutateProtocolVersion(998)

		_, err := harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{block})
		require.EqualError(t, err, "protocol version mismatch in transactions block header", "tx protocol was mutated, should fail")

		block = builders.BlockPair().Build()
		block.ResultsBlock.Header.MutateProtocolVersion(999)

		_, err = harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{block})
		require.EqualError(t, err, "protocol version mismatch in results block header", "rx protocol was mutated, should fail")

		block = builders.BlockPair().Build()
		block.TransactionsBlock.Header.MutateProtocolVersion(999)
		block.ResultsBlock.Header.MutateProtocolVersion(999)

		_, err = harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{block})
		require.EqualError(t, err, "protocol version mismatch in transactions block header", "tx and rx protocol was mutated, should fail")
	})
}

func TestValidateBlockWithValidHeight(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).
			withSyncBroadcast(1).
			withCommitStateDiff(1).
			withValidateConsensusAlgos(1).
			start(ctx)

		harness.commitBlock(ctx, builders.BlockPair().Build())

		block := builders.BlockPair().WithHeight(2).Build()

		_, err := harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{block})
		require.NoError(t, err, "happy flow")
	})
}

func TestValidateBlockWithInvalidHeight(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).
			withSyncBroadcast(1).
			withCommitStateDiff(1).
			withValidateConsensusAlgos(1).
			start(ctx)

		harness.commitBlock(ctx, builders.BlockPair().Build())

		block := builders.BlockPair().WithHeight(2).Build()

		block.TransactionsBlock.Header.MutateBlockHeight(998)

		_, err := harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{block})
		require.EqualError(t, err, "block pair height mismatch. transactions height is 998, results height is 2", "tx block height was mutate, expected an error")

		block.ResultsBlock.Header.MutateBlockHeight(999)

		_, err = harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{block})
		require.EqualError(t, err, "block pair height mismatch. transactions height is 998, results height is 999", "rx block height was mutate, expected an error")

		block.TransactionsBlock.Header.MutateBlockHeight(999)

		_, err = harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{block})
		require.EqualError(t, err, "block height is 999, expected 2", "tx & rx block height was mutate, expected an error")

		block.TransactionsBlock.Header.MutateProtocolVersion(1)

		_, err = harness.blockStorage.ValidateBlockForCommit(ctx, &services.ValidateBlockForCommitInput{block})
		require.EqualError(t, err, "block height is 999, expected 2", "only rx block height was mutate, expected an error")
	})
}

//TODO(v1) validate virtual chain
//TODO(v1) validate transactions root hash
//TODO(v1) validate metadata hash
//TODO(v1) validate receipts root hash
//TODO(v1) validate state diff hash
//TODO(v1) validate block consensus
