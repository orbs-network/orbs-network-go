package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestReturnBlockPair(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness().
			withSyncBroadcast(1).
			withCommitStateDiff(1).
			withValidateConsensusAlgos(1).
			start(ctx)

		block := builders.BlockPair().Build()
		harness.commitBlock(ctx, block)

		output, err := harness.blockStorage.GetBlockPair(ctx, &services.GetBlockPairInput{BlockHeight: 1})

		require.NoError(t, err, "this is a happy flow test")
		require.EqualValues(t, block, output.BlockPair, "block data should be as committed")
	})
}
