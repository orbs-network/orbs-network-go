// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package timestampfinder

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/pkg/errors"
	"math/big"
	"time"
)

const FAKE_CLIENT_NUMBER_OF_BLOCKS = 1000000
const FAKE_CLIENT_FIRST_TIMESTAMP_SECONDS = 1500000000
const FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS = 1506108784

var LastTimestampInFake = time.Unix(FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS, 0)

type FakeBlockTimeGetter struct {
	data        map[int64]int64 // block number -> timestamp in seconds
	logger      log.BasicLogger
	TimesCalled int
	Latency     time.Duration
}

func NewFakeBlockTimeGetter(logger log.BasicLogger) *FakeBlockTimeGetter {
	f := &FakeBlockTimeGetter{
		data:    make(map[int64]int64),
		Latency: 0,
	}

	f.logger = logger.WithTags(log.String("adapter", "ethereum-fake"))

	secondsJitter := int64(1)
	secondsSpacer := int64(10)
	timestampStart := int64(FAKE_CLIENT_FIRST_TIMESTAMP_SECONDS) // 2017/07/14 @ 14:40 - it will always end at 1506108783, or 2017/09/22 @ 19:3303
	f.data[0] = timestampStart
	for blockNumber := int64(1); blockNumber <= FAKE_CLIENT_NUMBER_OF_BLOCKS; blockNumber++ {
		// important that the numbers will be always increasing but always less than spacer
		if blockNumber%150 == 0 {
			secondsJitter++
		}
		if blockNumber%1000 == 0 {
			secondsJitter = 1
		}

		if blockNumber%3 == 0 { // use the jitter every 3, jitter is always less than spacer so this is okay
			f.data[blockNumber] = f.data[blockNumber-1] + secondsSpacer
		} else {
			f.data[blockNumber] = f.data[blockNumber-1] + secondsJitter
		}
	}

	f.logger.Info("finished initializing 'ethdb'", log.Int64("last-ts", f.data[FAKE_CLIENT_NUMBER_OF_BLOCKS]))

	return f
}

func (f *FakeBlockTimeGetter) WithLatency(d time.Duration) *FakeBlockTimeGetter {
	f.Latency = d
	return f
}

func (f *FakeBlockTimeGetter) GetTimestampForBlockNumber(ctx context.Context, blockNumber *big.Int) (*BlockNumberAndTime, error) {
	if blockNumber == nil {
		panic("requested nil block number which is not allowed")
	}

	h := &BlockNumberAndTime{
		BlockNumber:   blockNumber.Int64(),
		BlockTimeNano: secondsToNano(f.data[blockNumber.Int64()]),
	}

	if h.BlockTimeNano == 0 {
		return nil, errors.Errorf("search was done out of range, number: %d", blockNumber.Int64())
	}

	f.TimesCalled++
	if f.Latency > 0 {
		time.Sleep(f.Latency)
	}

	return h, nil
}

func (f *FakeBlockTimeGetter) GetTimestampForLatestBlock(ctx context.Context) (*BlockNumberAndTime, error) {
	return &BlockNumberAndTime{
		BlockNumber:   FAKE_CLIENT_NUMBER_OF_BLOCKS,
		BlockTimeNano: secondsToNano(f.data[FAKE_CLIENT_NUMBER_OF_BLOCKS]),
	}, nil
}
