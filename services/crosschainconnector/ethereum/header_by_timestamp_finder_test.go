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
		require.EqualError(t, err, "requested future block at time 2031-08-09 09:49:03 +0000 UTC, latest block time is 2017-09-22 19:33:04 +0000 UTC", "expecting an error when trying to go to the future")
	})
}

func TestGetEthBlockByTimestampOfExactlyLatestBlockSucceeds(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		bfh := NewFakeBlockAndTimestampGetter(logger)
		fetcher := NewTimestampFetcher(bfh, logger)

		b, err := fetcher.GetBlockByTimestamp(ctx, primitives.TimestampNano(FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED*time.Second))
		require.NoError(t, err, "expecting no error when trying to get latest time with some extra millis")
		require.EqualValues(t, FAKE_CLIENT_NUMBER_OF_BLOCKS, b.Int64(), "expecting block number to be of last value in fake db")
	})
}

func TestGetEthBlockByTimestampOfAlmostLatestBlockSucceeds(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		bfh := NewFakeBlockAndTimestampGetter(logger)
		fetcher := NewTimestampFetcher(bfh, logger)

		b, err := fetcher.GetBlockByTimestamp(ctx, primitives.TimestampNano((FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED-1)*time.Second))
		require.NoError(t, err, "expecting no error when trying to get latest time with some extra millis")
		// why -1 below? because the algorithm locks us to a block with time stamp **less** than what we requested, so it finds the latest but it is greater (ts-wise) so it will return -1
		require.EqualValues(t, FAKE_CLIENT_NUMBER_OF_BLOCKS-1, b.Int64(), "expecting block number to be of last value in fake db")
	})
}

func TestGetEthBlockByTimestampDoesNotPanicOnSameTimeBlocks(t *testing.T) {
	// this test is only valid for when testing with ganache, where we might get this kind of scenario, cannot (?) happen on real ethereum
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		bfh := NewFakeBlockAndTimestampGetter(logger).WithSameTimeBlocks(0.95)
		fetcher := NewTimestampFetcher(bfh, logger)

		_, err := fetcher.GetBlockByTimestamp(ctx, primitives.TimestampNano(1500198628000000000)) // should be block 32599, but this fake db was reset to same ts
		require.Error(t, err, "expecting no error when trying to get latest time with some extra millis")
		t.Log(err)
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
