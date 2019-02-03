package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInitSetsLastCommittedBlockHeightToZero(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).withSyncBroadcast(1).start(ctx)

		val, err := harness.blockStorage.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
		require.NoError(t, err)

		require.EqualValues(t, 0, val.LastCommittedBlockHeight)
		require.EqualValues(t, 0, val.LastCommittedBlockTimestamp)

		harness.verifyMocks(t, 1)
	})
}

func TestInitSetsLastCommittedBlockHeightFromPersistence(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).withSyncBroadcast(1)
		now := harness.setupCustomBlocksForInit()
		harness = harness.start(ctx)

		val, err := harness.blockStorage.GetLastCommittedBlockHeight(ctx, &services.GetLastCommittedBlockHeightInput{})
		require.NoError(t, err)

		require.EqualValues(t, 10, val.LastCommittedBlockHeight)
		require.EqualValues(t, now.UnixNano(), val.LastCommittedBlockTimestamp)

		harness.verifyMocks(t, 1)
	})
}
