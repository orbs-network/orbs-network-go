package timestampfinder

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
	"time"
)

func TestGetEthBlockBeforeEthGenesis(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		btg := NewFakeBlockTimeGetter(logger)
		finder := NewTimestampFinder(btg, logger)
		// something before 2015/07/31
		_, err := finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1438300700000000000))
		require.Error(t, err, "expecting an error when trying to go too much into the past")
	})
}

func TestGetEthBlockByTimestampFromFutureFails(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		btg := NewFakeBlockTimeGetter(logger)
		finder := NewTimestampFinder(btg, logger)

		// something in the future (sometime in 2031), it works on a fake database - which will never advance in time
		_, err := finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1944035343000000000))
		require.Error(t, err, "expecting an error when trying to go to the future")
	})
}

func TestGetEthBlockByTimestampOfExactlyLatestBlockFails(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		btg := NewFakeBlockTimeGetter(logger)
		finder := NewTimestampFinder(btg, logger)

		_, err := finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS*time.Second))
		require.Error(t, err, "expecting error when trying to get exactly the latest time")
	})
}

func TestGetEthBlockByTimestampOfAlmostLatestBlockSucceeds(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		btg := NewFakeBlockTimeGetter(logger)
		finder := NewTimestampFinder(btg, logger)

		b, err := finder.FindBlockByTimestamp(ctx, primitives.TimestampNano((FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS-1)*time.Second))
		require.NoError(t, err, "expecting no error when trying to get latest time with some extra millis")
		// why -1 below? because the algorithm locks us to a block with time stamp **less** than what we requested, so it finds the latest but it is greater (ts-wise) so it will return -1
		require.EqualValues(t, FAKE_CLIENT_NUMBER_OF_BLOCKS-1, b.Int64(), "expecting block number to be of last value in fake db")
	})
}

func TestGetEthBlockByTimestampFromEth(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		btg := NewFakeBlockTimeGetter(logger)
		finder := NewTimestampFinder(btg, logger)

		// something recent
		blockBI, err := finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1505735343000000000))
		block := blockBI.Int64()
		require.NoError(t, err, "something went wrong while getting the block by timestamp of a recent block")
		require.EqualValues(t, 938874, block, "expected ts 1505735343 to return a specific block")

		// something not so recent
		blockBI, err = finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1500198628000000000))
		block = blockBI.Int64()
		require.NoError(t, err, "something went wrong while getting the block by timestamp of an older block")
		require.EqualValues(t, 32600, block, "expected ts 1500198628 to return a specific block")

		callsBefore := btg.TimesCalled
		// "realtime" - 200 seconds
		blockBI, err = finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1506108583000000000))
		require.NoError(t, err, "something went wrong while getting the block by timestamp of a 'realtime' block")
		newBlock := blockBI.Int64()
		require.EqualValues(t, 999974, newBlock, "expected ts 1506108583 to return a specific block")

		t.Log(btg.TimesCalled - callsBefore)
	})
}

func TestGetEthBlockByTimestampWorksWithIdenticalRequestsFromCache(t *testing.T) {
	// this test relies on the context cancellation - without cache this will return an error
	var (
		externalErr   error
		externalBlock *big.Int
	)

	latch := make(chan struct{})
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		btg := NewFakeBlockTimeGetter(logger)
		finder := NewTimestampFinder(btg, logger)

		// complex request
		blockBI, internalErr := finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1505735343000000000))
		block := blockBI.Int64()
		require.NoError(t, internalErr, "something went wrong while getting the block by timestamp of a recent block")
		require.EqualValues(t, 938874, block, "expected ts 1505735343 to return a specific block")

		// same exact request again, async so we can check if it works when context was canceled, latency ensures we cannot really search before context will be done
		btg.Latency = 200 * time.Millisecond
		go func() {
			externalBlock, externalErr = finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1505735343000000000))
			latch <- struct{}{}
		}()
	})

	<-latch

	block := externalBlock.Int64()
	require.NoError(t, externalErr, "expected cache to hit even though context is done already")
	require.EqualValues(t, 938874, block, "expected ts 1505735343 to return a specific block")
}

func TestGetEthBlockByTimestampWorksWithDifferentRequestsFromCache(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		btg := NewFakeBlockTimeGetter(logger)
		finder := NewTimestampFinder(btg, logger)

		desiredIterations := 20
		jump := (FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS - FAKE_CLIENT_FIRST_TIMESTAMP_SECONDS) / desiredIterations
		for seconds := FAKE_CLIENT_FIRST_TIMESTAMP_SECONDS + 10; seconds < FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS; seconds += jump {

			_, err := finder.FindBlockByTimestamp(ctx, secondsToNano(int64(seconds)))
			require.NoError(t, err)
			t.Log("")

		}
	})
}

func TestGetEthBlockByTimestampWhenSmallNumOfBlocks(t *testing.T) {
	tests := []struct {
		name          string
		referenceTs   primitives.TimestampNano
		btg           BlockTimeGetter
		expectedError bool
		expectedNum   int64
	}{
		{
			name:          "NoBlocks",
			referenceTs:   1022,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{}),
			expectedError: true,
			expectedNum:   0,
		},
		{
			name:          "OneBlock_Equals",
			referenceTs:   1022,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1022}}),
			expectedError: true,
			expectedNum:   0,
		},
		{
			name:          "OneBlock_Below",
			referenceTs:   1022,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1011}}),
			expectedError: true,
			expectedNum:   0,
		},
		{
			name:          "OneBlock_Above",
			referenceTs:   1022,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1033}}),
			expectedError: true,
			expectedNum:   0,
		},
		{
			name:          "TwoBlocks_Middle",
			referenceTs:   1500,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1000}, {2, 2000}}),
			expectedError: false,
			expectedNum:   1,
		},
		{
			name:          "JustIdenticalBlocks",
			referenceTs:   1000,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1000}, {2, 1000}, {3, 1000}}),
			expectedError: true,
			expectedNum:   0,
		},
		{
			name:          "SeveralIdenticalBlocks_Middle",
			referenceTs:   1500,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1000}, {2, 1000}, {3, 1000}, {4, 2000}}),
			expectedError: false,
			expectedNum:   3,
		},
		{
			name:          "SeveralIdenticalBlocks_Equal",
			referenceTs:   1000,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1000}, {2, 1000}, {3, 1000}, {4, 2000}}),
			expectedError: false,
			expectedNum:   3,
		},
		{
			name:          "SlowBlocks_ThenFast_Below",
			referenceTs:   3000000000000,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1000000000000}, {2, 2000000000000}, {3, 3000000000000}, {4, 3000000000001}, {5, 3000000000002}}),
			expectedError: false,
			expectedNum:   3,
		},
		{
			name:          "SlowBlocks_ThenFast_Above",
			referenceTs:   3000000000001,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1000000000000}, {2, 2000000000000}, {3, 3000000000000}, {4, 3000000000001}, {5, 3000000000002}}),
			expectedError: false,
			expectedNum:   4,
		},
		{
			name:          "FastBlocks_ThenSlow_Below",
			referenceTs:   1000000000002,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1000000000000}, {2, 1000000000001}, {3, 1000000000002}, {4, 2000000000001}, {5, 3000000000002}}),
			expectedError: false,
			expectedNum:   3,
		},
		{
			name:          "FastBlocks_ThenSlow_Above",
			referenceTs:   2000000000001,
			btg:           newBlockTimeGetterStub([]BlockNumberAndTime{{1, 1000000000000}, {2, 1000000000001}, {3, 1000000000002}, {4, 2000000000001}, {5, 3000000000002}}),
			expectedError: false,
			expectedNum:   4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test.WithContext(func(ctx context.Context) {

				logger := log.DefaultTestingLogger(t)
				finder := NewTimestampFinder(tt.btg, logger)
				blockBI, err := finder.FindBlockByTimestamp(ctx, tt.referenceTs)
				if !tt.expectedError {
					require.NoError(t, err)
					require.Equal(t, tt.expectedNum, blockBI.Int64())
				} else {
					require.Error(t, err)
				}

			})
		})
	}
}

func TestTimestampFinderTerminatesOnContextCancel(t *testing.T) {
	var err error
	latch := make(chan struct{})
	test.WithContext(func(ctx context.Context) {
		logger := log.DefaultTestingLogger(t)
		btg := NewFakeBlockTimeGetter(logger).WithLatency(20 * time.Millisecond)
		finder := NewTimestampFinder(btg, logger)

		go func() {
			// should return block 938874, but we are going to cancel the context
			_, err = finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1505735343000000000))
			latch <- struct{}{}
		}()
	})

	<-latch
	require.EqualError(t, err, "aborting search - context canceled")
}
