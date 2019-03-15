package timestampfinder

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/pkg/errors"
	"math/big"
	"time"
)

const FAKE_CLIENT_NUMBER_OF_BLOCKS = 1000000
const FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED = 1506108784

var LastTimestampInFake = time.Unix(FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED, 0)

type FakeBlockTimeGetter struct {
	data        map[int64]int64 // block number -> timestamp in seconds
	logger      log.BasicLogger
	TimesCalled int
}

func (f *FakeBlockTimeGetter) GetTimestampForBlockNumber(ctx context.Context, blockNumber *big.Int) (*BlockNumberAndTime, error) {
	if blockNumber == nil {
		return &BlockNumberAndTime{
			BlockNumber: FAKE_CLIENT_NUMBER_OF_BLOCKS,
			TimeSeconds: f.data[FAKE_CLIENT_NUMBER_OF_BLOCKS],
		}, nil
	}

	h := &BlockNumberAndTime{
		BlockNumber: blockNumber.Int64(),
		TimeSeconds: f.data[blockNumber.Int64()],
	}

	if h.TimeSeconds == 0 {
		return nil, errors.Errorf("search was done out of range, number: %d", blockNumber.Int64())
	}

	f.TimesCalled++

	return h, nil
}

func NewFakeBlockTimeGetter(logger log.BasicLogger) *FakeBlockTimeGetter {
	f := &FakeBlockTimeGetter{
		data: make(map[int64]int64),
	}

	f.logger = logger.WithTags(log.String("adapter", "ethereum-fake"))

	secondsJitter := int64(1)
	secondsSpacer := int64(10)
	timestampStart := int64(1500000000) // 2017/07/14 @ 14:40 - it will always end at 1506108783, or 2017/09/22 @ 19:3303
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

	f.logger.Info("finished initializing 'ethdb'", log.Int64("last-ts", f.data[FAKE_CLIENT_NUMBER_OF_BLOCKS-1]))

	return f
}
