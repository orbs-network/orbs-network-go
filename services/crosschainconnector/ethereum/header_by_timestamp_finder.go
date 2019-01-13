package ethereum

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math"
	"math/big"
	"time"
)

type TimestampFetcher interface {
	GetBlockByTimestamp(ctx context.Context, nano primitives.TimestampNano) (*big.Int, error)
}

type finder struct {
	logger log.BasicLogger
	btg    BlockAndTimestampGetter
}

func NewTimestampFetcher(btg BlockAndTimestampGetter, logger log.BasicLogger) *finder {
	f := &finder{
		btg:    btg,
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

	latest, err := f.btg.ApproximateBlockAt(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get latest block")
	}

	if latest == nil { // simulator always returns nil block number
		return nil, nil
	}

	// this was added to support simulations and tests, should not be relevant for production
	latestNum := latest.Number
	latestNum -= 10000
	if latestNum < 0 {
		latestNum = 0
	}
	back10k, err := f.btg.ApproximateBlockAt(ctx, big.NewInt(latestNum))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get past reference block")
	}

	theBlock, err := f.findBlockByTimeStamp(ctx, timestampInSeconds, back10k.Number, back10k.Time, latest.Number, latest.Time)
	return theBlock, err
}

func (f *finder) findBlockByTimeStamp(ctx context.Context, timestamp int64, currentBlockNumber, currentTimestamp, prevBlockNumber, prevTimestamp int64) (*big.Int, error) {
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
	guess, err := f.btg.ApproximateBlockAt(ctx, big.NewInt(guessBlockNumber))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block by number")
	}

	return f.findBlockByTimeStamp(ctx, timestamp, guess.Number, guess.Time, currentBlockNumber, currentTimestamp)
}
