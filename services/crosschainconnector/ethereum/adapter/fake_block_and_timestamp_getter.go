package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/pkg/errors"
	"math/big"
	"time"
)

const FAKE_CLIENT_NUMBER_OF_BLOCKS = 1000000
const FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED = 1506108783

var LastTimestampInFake = time.Unix(FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED, 0)

type FakeBlockAndTimestampGetter struct {
	data   map[int64]int64
	logger log.BasicLogger
	// block number -> timestamp
}

func (f *FakeBlockAndTimestampGetter) ApproximateBlockAt(ctx context.Context, blockNumber *big.Int) (*BlockHeightAndTime, error) {
	if blockNumber == nil {
		return &BlockHeightAndTime{
			Number: FAKE_CLIENT_NUMBER_OF_BLOCKS,
			Time:   f.data[FAKE_CLIENT_NUMBER_OF_BLOCKS-1],
		}, nil
	}

	h := &BlockHeightAndTime{
		Number: blockNumber.Int64(),
		Time:   f.data[blockNumber.Int64()],
	}

	if h.Time == 0 {
		return nil, errors.Errorf("search was done out of range, number: %d", blockNumber.Int64())
	}

	return h, nil
}

func NewFakeBlockAndTimestampGetter(logger log.BasicLogger) *FakeBlockAndTimestampGetter {
	f := &FakeBlockAndTimestampGetter{
		data: make(map[int64]int64),
	}

	f.logger = logger.WithTags(log.String("adapter", "ethereum-fake"))

	jitter := int64(1)
	spacer := int64(10)
	start := int64(1500000000) // 2017/07/14 @ 14:40 - it will always end at 1506108783, or 2017/09/22 @ 19:3303
	f.data[0] = start
	for i := int64(1); i < FAKE_CLIENT_NUMBER_OF_BLOCKS; i++ {
		// important that the numbers will be always increasing but always less than spacer
		if i%150 == 0 {
			jitter++
		}
		if i%1000 == 0 {
			jitter = 1
		}

		if i%3 == 0 { // use the jitter every 3, jitter is always less than spacer so this is okay
			f.data[i] = f.data[i-1] + spacer
		} else {
			f.data[i] = f.data[i-1] + jitter
		}
	}

	f.logger.Info("finished initializing 'ethdb'", log.Int64("last-ts", f.data[FAKE_CLIENT_NUMBER_OF_BLOCKS-1]))

	return f
}
