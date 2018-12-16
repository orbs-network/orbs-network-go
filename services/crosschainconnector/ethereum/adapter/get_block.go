package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math/big"
)

func (c *connectorCommon) GetBlockByTimestamp(ctx context.Context, nano primitives.TimestampNano) (*big.Int, error) {
	return c.getBlockByTimestamp(ctx, nano)
}

func (c *connectorCommon) findBlockByTimeStamp(ctx context.Context, eth *ethclient.Client, timestamp int64, highBlockNumber, highTimestamp, lowBlockNumber, lowTimestamp int64) (*big.Int, error) {
	c.logger.Info("searching for block in ethereum",
		log.Int64("target-timestamp", timestamp),
		log.Int64("high-block-number", highBlockNumber),
		log.Int64("high-timestamp", highTimestamp),
		log.Int64("low-block-number", lowBlockNumber),
		log.Int64("low-timestamp", lowTimestamp))

	highLowTimeDiff := highTimestamp - lowTimestamp
	blockNumberDiff := highBlockNumber - lowBlockNumber

	// we stop when the range we are in-between is 1 or 0 (same block), it means we found a block with the exact timestamp or lowest from below
	if blockNumberDiff == 1 || blockNumberDiff == 0 {
		// if the block we are returning has a ts > target, it means we want one block before (so our ts is always bigger than block ts)
		if lowTimestamp > timestamp {
			return big.NewInt(lowBlockNumber - 1), nil
		} else {
			return big.NewInt(lowBlockNumber), nil
		}
	}

	secondsPerBlock := highLowTimeDiff / blockNumberDiff
	distanceToTargetFromHigh := highTimestamp - timestamp
	blocksToJump := distanceToTargetFromHigh / secondsPerBlock
	c.logger.Info("eth block search delta", log.Int64("jump-backwards", blocksToJump))
	guessBlockNumber := highBlockNumber - blocksToJump
	guess, err := eth.HeaderByNumber(ctx, big.NewInt(guessBlockNumber))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block by number")
	}

	guessTimestamp := guess.Time.Int64()

	// create new relative block diff from guess (educated guess according to known local average)
	distanceFromTarget := timestamp - guessTimestamp
	blocksToJumpForNewLocalGuess := distanceFromTarget / secondsPerBlock
	guessLocalTarget := guessBlockNumber + blocksToJumpForNewLocalGuess
	newLocalToGuess, err := eth.HeaderByNumber(ctx, big.NewInt(guessLocalTarget))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block by number")
	}

	if guessTimestamp > timestamp {
		// jumping backwards more, we did not jump far enough
		return c.findBlockByTimeStamp(ctx, eth, timestamp, guessBlockNumber, guessTimestamp, newLocalToGuess.Number.Int64(), newLocalToGuess.Time.Int64())
	} else {
		// need to jump forward as we skipped it
		return c.findBlockByTimeStamp(ctx, eth, timestamp, newLocalToGuess.Number.Int64(), newLocalToGuess.Time.Int64(), guessBlockNumber, guessTimestamp)
	}
}
