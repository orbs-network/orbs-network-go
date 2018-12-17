package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math"
	"math/big"
)

func (c *connectorCommon) GetBlockByTimestamp(ctx context.Context, nano primitives.TimestampNano) (*big.Int, error) {
	return c.getBlockByTimestamp(ctx, nano)
}

func (c *connectorCommon) findBlockByTimeStamp(ctx context.Context, eth ClientForGetBlockHeader, timestamp int64, currentBlockNumber, currentTimestamp, prevBlockNumber, prevTimestamp int64) (*big.Int, error) {
	c.logger.Info("searching for block in ethereum",
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
	c.logger.Info("eth block search delta", log.Int64("jump-backwards", blocksToJump))
	guessBlockNumber := currentBlockNumber - blocksToJump
	guess, err := eth.HeaderByNumber(ctx, big.NewInt(guessBlockNumber))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block by number")
	}

	guessTimestamp := guess.Time.Int64()

	return c.findBlockByTimeStamp(ctx, eth, timestamp, guessBlockNumber, guessTimestamp, currentBlockNumber, currentTimestamp)
}
