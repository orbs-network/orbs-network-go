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
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestReturnBlockPair(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness, block := generateAndCommitOneBlock(ctx, t)

		output, err := harness.blockStorage.GetBlockPair(ctx, &services.GetBlockPairInput{BlockHeight: 1})

		require.NoError(t, err, "this is a happy flow test (ask a real block)")
		require.EqualValues(t, block, output.BlockPair, "block data should be as committed")
	})
}

func TestReturnNilWhenBlockHeight0(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness, _ := generateAndCommitOneBlock(ctx, t)

		output, err := harness.blockStorage.GetBlockPair(ctx, &services.GetBlockPairInput{BlockHeight: 0})

		require.Error(t, err, "ask 0 is not valid")
		require.Nil(t, output, "block data should nil")
	})
}

func TestReturnNilWhenBlockHeightInTheFuture(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness, _ := generateAndCommitOneBlock(ctx, t)
		output, err := harness.blockStorage.GetBlockPair(ctx, &services.GetBlockPairInput{BlockHeight: 1000})

		require.NoError(t, err, "far future is not found but valid")
		require.Nil(t, output.BlockPair, "block pair result should be nil")
	})
}

func TestReturnNilWhenBlockHeightInTrackerGraceButTimesOut(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness, _ := generateAndCommitOneBlock(ctx, t)

		childCtx, _ := context.WithTimeout(ctx, time.Millisecond)
		output, err := harness.blockStorage.GetBlockPair(childCtx, &services.GetBlockPairInput{BlockHeight: 2})

		require.NoError(t, err, "far future is not found but valid")
		require.Nil(t, output.BlockPair, "block pair result should be nil")
	})
}

func generateAndCommitOneBlock(ctx context.Context, t *testing.T) (*harness, *protocol.BlockPairContainer) {
	harness := newBlockStorageHarness(t).
		withSyncBroadcast(1).
		withCommitStateDiff(1).
		withValidateConsensusAlgos(1).
		start(ctx)

	block := builders.BlockPair().Build()
	harness.commitBlock(ctx, block)
	return harness, block
}
