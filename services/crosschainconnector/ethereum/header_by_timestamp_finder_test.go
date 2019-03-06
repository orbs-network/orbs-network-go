package ethereum

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGetEthBlockBeforeEthGenesis(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		bfh := NewFakeBlockAndTimestampGetter(logger)
		fetcher := NewTimestampFetcher(bfh, logger)
		// something before 2015/07/31
		_, err := fetcher.GetBlockByTimestamp(ctx, primitives.TimestampNano(1438300700000000000))
		require.Error(t, err, "expecting an error when trying to go too much into the past")
	})
}

func TestGetEthBlockByTimestampFromFutureFails(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		bfh := NewFakeBlockAndTimestampGetter(logger)
		fetcher := NewTimestampFetcher(bfh, logger)

		// something in the future (sometime in 2031), it works on a fake database - which will never advance in time
		_, err := fetcher.GetBlockByTimestamp(ctx, primitives.TimestampNano(1944035343000000000))
		require.Error(t, err, "expecting an error when trying to go to the future")
	})
}

func TestGetEthBlockByTimestampFromNearFutureReturnsLatest(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		bfh := NewFakeBlockAndTimestampGetter(logger)
		fetcher := NewTimestampFetcher(bfh, logger)

		blockBI, err := fetcher.GetBlockByTimestamp(ctx, primitives.TimestampNano(FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED*time.Second+NEAR_FUTURE_GRACE))
		require.NoError(t, err, "expecting no error when trying to go to the near future")
		require.EqualValues(t, FAKE_CLIENT_NUMBER_OF_BLOCKS, blockBI.Int64(), "expected to get last block when asking for a block from near future")
	})
}

func TestGetEthBlockByTimestampFromEth(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		bfh := NewFakeBlockAndTimestampGetter(logger)
		fetcher := NewTimestampFetcher(bfh, logger)

		// something recent
		blockBI, err := fetcher.GetBlockByTimestamp(ctx, primitives.TimestampNano(1505735343000000000))
		block := blockBI.Int64()
		require.NoError(t, err, "something went wrong while getting the block by timestamp of a recent block")
		require.EqualValues(t, 938874, block, "expected ts 1505735343 to return a specific block")

		// something not so recent
		blockBI, err = fetcher.GetBlockByTimestamp(ctx, primitives.TimestampNano(1500198628000000000))
		block = blockBI.Int64()
		require.NoError(t, err, "something went wrong while getting the block by timestamp of an older block")
		require.EqualValues(t, 32599, block, "expected ts 1500198628 to return a specific block")

		callsBefore := bfh.TimesCalled
		// "realtime" - 200 seconds
		blockBI, err = fetcher.GetBlockByTimestamp(ctx, primitives.TimestampNano(1506108583000000000))
		require.NoError(t, err, "something went wrong while getting the block by timestamp of a 'realtime' block")
		newBlock := blockBI.Int64()
		require.EqualValues(t, 999974, newBlock, "expected ts 1506108583 to return a specific block")

		t.Log(bfh.TimesCalled - callsBefore)
	})
}
