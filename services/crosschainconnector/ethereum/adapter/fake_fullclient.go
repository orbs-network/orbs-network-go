package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math/big"
	"time"
)

const NUMBER_OF_BLOCKS = 1000000

type FakeFullClient struct {
	connectorCommon

	data map[int64]int64 // block number -> timestamp
}

func (ffc *FakeFullClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	h := &types.Header{
		Number: number,
		Time:   big.NewInt(ffc.data[number.Int64()]),
	}

	if h.Time.Int64() == 0 {
		return nil, errors.Errorf("search was done out of range, number: %d", number.Int64())
	}

	return h, nil
}

func NewFakeFullClientConnection(logger log.BasicLogger) *FakeFullClient {
	ffc := &FakeFullClient{
		data: make(map[int64]int64),
	}

	ffc.logger = logger.WithTags(log.String("adapter", "ethereum-fake"))

	jitter := int64(1)
	spacer := int64(10)
	start := int64(1500000000) // 2017/07/14 @ 14:40 - it will always end at 1506108783, or 2017/09/22 @ 19:3303
	ffc.data[0] = start
	for i := int64(1); i < NUMBER_OF_BLOCKS; i++ {
		// important that the numbers will be always increasing but always less than spacer
		if i%150 == 0 {
			jitter++
		}
		if i%1000 == 0 {
			jitter = 1
		}

		if i%3 == 0 { // use the jitter every 3, jitter is always less than spacer so this is okay
			ffc.data[i] = ffc.data[i-1] + spacer
		} else {
			ffc.data[i] = ffc.data[i-1] + jitter
		}
	}

	ffc.getBlockByTimestamp = ffc.getFakeBlockByTimestamp

	ffc.logger.Info("finished initializing 'ethdb'", log.Int64("last-ts", ffc.data[NUMBER_OF_BLOCKS-1]))

	return ffc
}

func (ffc *FakeFullClient) getFakeBlockByTimestamp(ctx context.Context, nano primitives.TimestampNano) (*big.Int, error) {
	timestampInSeconds := int64(nano) / int64(time.Second)
	return ffc.findBlockByTimeStamp(ctx,
		ffc,
		timestampInSeconds,
		NUMBER_OF_BLOCKS-1000,
		ffc.data[NUMBER_OF_BLOCKS-1000],
		NUMBER_OF_BLOCKS,
		ffc.data[NUMBER_OF_BLOCKS-1])
}
