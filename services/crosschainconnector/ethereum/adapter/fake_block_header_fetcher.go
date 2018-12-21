package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/pkg/errors"
	"math/big"
)

const FAKE_CLIENT_NUMBER_OF_BLOCKS = 1000000
const FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED = 1506108783

type FakeBlockHeaderFetcher struct {
	data   map[int64]int64
	logger log.BasicLogger
	// block number -> timestamp
}

func (f *FakeBlockHeaderFetcher) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if number == nil {
		return &types.Header{
			Number: big.NewInt(FAKE_CLIENT_NUMBER_OF_BLOCKS),
			Time:   big.NewInt(f.data[FAKE_CLIENT_NUMBER_OF_BLOCKS-1]),
		}, nil
	}

	h := &types.Header{
		Number: number,
		Time:   big.NewInt(f.data[number.Int64()]),
	}

	if h.Time.Int64() == 0 {
		return nil, errors.Errorf("search was done out of range, number: %d", number.Int64())
	}

	return h, nil
}

func NewFakeBlockHeaderFetcher(logger log.BasicLogger) *FakeBlockHeaderFetcher {
	f := &FakeBlockHeaderFetcher{
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
