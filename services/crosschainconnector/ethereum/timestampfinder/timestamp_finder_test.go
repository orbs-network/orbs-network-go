// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package timestampfinder

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
	"time"
)

type harness struct {
	btg     BlockTimeGetter
	logger  log.BasicLogger
	metrics metric.Factory
	finder  *finder
}

func NewTestHarness(t testing.TB) *harness {
	logger := log.DefaultTestingLogger(t)
	btg := NewFakeBlockTimeGetter(logger)
	metricFactory := metric.NewRegistry()
	finder := NewTimestampFinder(btg, logger, metricFactory)

	h := &harness{
		btg:     btg,
		finder:  finder,
		logger:  logger,
		metrics: metricFactory,
	}

	return h
}

func (h *harness) GetBtgAsFake() *FakeBlockTimeGetter {
	btg := h.btg.(*FakeBlockTimeGetter) // will panic if not used with care
	return btg
}

func (h *harness) WithBtg(btg BlockTimeGetter) *harness {
	h.btg = btg
	finder := NewTimestampFinder(btg, h.logger, h.metrics)
	h.finder = finder
	return h
}

func TestGetEthBlockBeforeEthGenesis(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := NewTestHarness(t)
		// something before 2015/07/31
		_, err := h.finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1438300700000000000))
		require.Error(t, err, "expecting an error when trying to go too much into the past")
	})
}

func TestGetEthBlockByTimestampFromFutureFails(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := NewTestHarness(t)

		// something in the future (sometime in 2031), it works on a fake database - which will never advance in time
		_, err := h.finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1944035343000000000))
		require.Error(t, err, "expecting an error when trying to go to the future")
	})
}

func TestGetEthBlockByTimestampOfExactlyLatestBlockFails(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := NewTestHarness(t)

		_, err := h.finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS*time.Second))
		require.Error(t, err, "expecting error when trying to get exactly the latest time")
	})
}

func TestGetEthBlockByTimestampOfAlmostLatestBlockSucceeds(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := NewTestHarness(t)

		b, err := h.finder.FindBlockByTimestamp(ctx, primitives.TimestampNano((FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS-1)*time.Second))
		require.NoError(t, err, "expecting no error when trying to get latest time with some extra millis")
		// why -1 below? because the algorithm locks us to a block with time stamp **less** than what we requested, so it finds the latest but it is greater (ts-wise) so it will return -1
		require.EqualValues(t, FAKE_CLIENT_NUMBER_OF_BLOCKS-1, b.Int64(), "expecting block number to be of last value in fake db")
	})
}

func TestGetEthBlockByTimestampFromEth(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := NewTestHarness(t)

		// something recent
		blockBI, err := h.finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1505735343000000000))
		block := blockBI.Int64()
		require.NoError(t, err, "something went wrong while getting the block by timestamp of a recent block")
		require.EqualValues(t, 938874, block, "expected ts 1505735343 to return a specific block")

		// something not so recent
		blockBI, err = h.finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1500198628000000000))
		block = blockBI.Int64()
		require.NoError(t, err, "something went wrong while getting the block by timestamp of an older block")
		require.EqualValues(t, 32600, block, "expected ts 1500198628 to return a specific block")

		callsBefore := h.GetBtgAsFake().TimesCalled
		// "realtime" - 200 seconds
		blockBI, err = h.finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1506108583000000000))
		require.NoError(t, err, "something went wrong while getting the block by timestamp of a 'realtime' block")
		newBlock := blockBI.Int64()
		require.EqualValues(t, 999974, newBlock, "expected ts 1506108583 to return a specific block")

		t.Log(h.GetBtgAsFake().TimesCalled - callsBefore)
	})
}

func TestGetEthBlockByTimestampWorksWithIdenticalRequestsFromCache(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := NewTestHarness(t)

		// complex request
		blockBI, internalErr := h.finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1505735343000000000))
		block := blockBI.Int64()
		require.EqualValues(t, 0, h.finder.metrics.cacheHits.Value(), "shouldn't be a cache hit yet")
		require.NoError(t, internalErr, "something went wrong while getting the block by timestamp of a recent block")
		require.EqualValues(t, 938874, block, "expected ts 1505735343 to return a specific block")

		blockBI, internalErr = h.finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1505735343000000000))
		block = blockBI.Int64()
		require.EqualValues(t, 1, h.finder.metrics.cacheHits.Value(), "expected a cache hit from the metric")
		require.NoError(t, internalErr, "expected cache to hit to not throw an error")
		require.EqualValues(t, 938874, block, "expected ts 1505735343 to return a specific block")
	})
}

func TestGetEthBlockByTimestampWorksWithDifferentRequestsFromCache(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := NewTestHarness(t)

		desiredIterations := 20
		jump := (FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS - FAKE_CLIENT_FIRST_TIMESTAMP_SECONDS) / desiredIterations
		for seconds := FAKE_CLIENT_FIRST_TIMESTAMP_SECONDS + 10; seconds < FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS; seconds += jump {

			_, err := h.finder.FindBlockByTimestamp(ctx, secondsToNano(int64(seconds)))
			require.NoError(t, err)
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

				h := NewTestHarness(t).WithBtg(tt.btg)
				blockBI, err := h.finder.FindBlockByTimestamp(ctx, tt.referenceTs)
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
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	h := NewTestHarness(t)
	// should return block 938874, but we are going to cancel the context
	_, err := h.finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1505735343000000000))
	require.EqualError(t, err, "aborting search: context canceled")
}

func TestRunMultipleSearchesOnFakeGetter(t *testing.T) {
	h := NewTestHarness(t)
	test.WithContext(func(ctx context.Context) {
		searchRange := FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS - FAKE_CLIENT_FIRST_TIMESTAMP_SECONDS
		for i := 0; i < 500; i++ {
			// start searching in a random manner to avoid cache
			randBlockTime := rand.Intn(searchRange)
			_, err := h.finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(time.Duration(FAKE_CLIENT_FIRST_TIMESTAMP_SECONDS+randBlockTime)*time.Second))
			require.NoError(t, err)
		}
	})
}

func BenchmarkFullCycle(b *testing.B) {
	h := NewTestHarness(b)
	ctx := context.Background()
	// spin it
	h.finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1505735343000000000))
	searchRange := FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS - FAKE_CLIENT_FIRST_TIMESTAMP_SECONDS
	for i := 0; i < b.N; i++ {
		// start searching in a random manner to avoid cache
		randBlockTime := rand.Intn(searchRange)
		_, err := h.finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(time.Duration(FAKE_CLIENT_FIRST_TIMESTAMP_SECONDS+randBlockTime)*time.Second))
		if err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkFullCycleWithCache(b *testing.B) {
	h := NewTestHarness(b)
	ctx := context.Background()
	// spin it
	h.finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1505735343000000000))
	for i := 0; i < b.N; i++ {
		// start searching in a random manner to avoid cache
		_, err := h.finder.FindBlockByTimestamp(ctx, primitives.TimestampNano(1505735343000000000))
		if err != nil {
			b.Error(err)
		}
	}
}
