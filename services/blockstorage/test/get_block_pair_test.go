package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
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

		require.NoError(t, err, "this is a happy flow test (ask 0)")
		require.Nil(t, output.BlockPair, "block data should nil")
	})
}

func TestReturnNilWhenBlockHeightInTheFuture(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness, _ := generateAndCommitOneBlock(ctx, t)
		output, err := harness.blockStorage.GetBlockPair(ctx, &services.GetBlockPairInput{BlockHeight: 10})

		require.NoError(t, err, "this is a happy flow test (ask in future)")
		require.Nil(t, output.BlockPair, "block data should be nil")
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
