package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math"
	"math/big"
	"time"
)

type BlockHeaderFetcher interface {
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
}

type TimestampFetcher interface {
	GetBlockByTimestamp(ctx context.Context, nano primitives.TimestampNano) (*big.Int, error)
}

type finder struct {
	bfh    BlockHeaderFetcher
	logger log.BasicLogger
}

func NewTimestampFetcher(bfh BlockHeaderFetcher, logger log.BasicLogger) *finder {
	f := &finder{
		bfh:    bfh,
		logger: logger,
	}

	return f
}

func (f *finder) GetBlockByTimestamp(ctx context.Context, nano primitives.TimestampNano) (*big.Int, error) {
	timestampInSeconds := int64(nano) / int64(time.Second)
	// ethereum started around 2015/07/31
	if timestampInSeconds < 1438300800 {
		return nil, errors.New("cannot query before ethereum genesis")
	}

	latest, err := f.bfh.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get latest block")
	}

	if latest == nil { // simulator always returns nil block number
		return nil, nil
	}

	// this was added to support simulations and tests, should not be relevant for production
	latestNum := latest.Number.Int64()
	latestNum -= 10000
	if latestNum < 0 {
		latestNum = 0
	}
	back10k, err := f.bfh.HeaderByNumber(ctx, big.NewInt(latestNum))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get past reference block")
	}

	theBlock, err := f.findBlockByTimeStamp(ctx, f.bfh, timestampInSeconds, back10k.Number.Int64(), back10k.Time.Int64(), latest.Number.Int64(), latest.Time.Int64())
	return theBlock, err
}

func (f *finder) findBlockByTimeStamp(ctx context.Context, eth BlockHeaderFetcher, timestamp int64, currentBlockNumber, currentTimestamp, prevBlockNumber, prevTimestamp int64) (*big.Int, error) {
	f.logger.Info("searching for block in ethereum",
		log.Int64("target-timestamp", timestamp),
		log.Int64("current-block-number", currentBlockNumber),
		log.Int64("current-timestamp", currentTimestamp),
		log.Int64("prev-block-number", prevBlockNumber),
		log.Int64("prev-timestamp", prevTimestamp))
	blockNumberDiff := currentBlockNumber - prevBlockNumber

	// we stop when the range we are in-between is 1 or 0 (same block), it means we found a block with the exact timestamp or lowest from below
	if blockNumberDiff == 1 || blockNumberDiff == 0 {
		// if the block we are returning has a ts > target, it means we want one block before (so our ts is always bigger than block ts)
		if currentTimestamp > timestamp {
			return big.NewInt(currentBlockNumber - 1), nil
		} else {
			return big.NewInt(currentBlockNumber), nil
		}
	}

	timeDiff := currentTimestamp - prevTimestamp
	secondsPerBlock := int64(math.Ceil(float64(timeDiff) / float64(blockNumberDiff)))
	distanceToTargetFromCurrent := currentTimestamp - timestamp
	blocksToJump := distanceToTargetFromCurrent / secondsPerBlock
	f.logger.Info("eth block search delta", log.Int64("jump-backwards", blocksToJump))
	guessBlockNumber := currentBlockNumber - blocksToJump
	guess, err := eth.HeaderByNumber(ctx, big.NewInt(guessBlockNumber))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block by number")
	}

	guessTimestamp := guess.Time.Int64()

	return f.findBlockByTimeStamp(ctx, eth, timestamp, guessBlockNumber, guessTimestamp, currentBlockNumber, currentTimestamp)
}
