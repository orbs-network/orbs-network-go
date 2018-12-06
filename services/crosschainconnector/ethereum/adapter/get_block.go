package adapter

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"math/big"
	"time"
)

func (c *connectorCommon) GetBlockByTimestamp(ctx context.Context, nano primitives.TimestampNano) (int64, error) {
	client, err := c.getFullClient()
	if err != nil {
		return -1, err
	}

	timestampInSeconds := int64(nano) / int64(time.Second)

	latest, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return -1, err
	}

	latestTimestamp := latest.Time.Int64()
	if latest.Time.Int64() < timestampInSeconds {
		return -1, errors.New("invalid request to get block, trying to get a block in the future (sync issues?)")
	}

	latestNumber := latest.Number.Int64()
	// a possible improvement can be instead of going back 10k blocks, assume secs/block to begin with and guess the block ts/number, but that may cause invalid calculation for older blocks
	tenKblocksAgoNumber := big.NewInt(latestNumber - 10000)
	older, err := client.HeaderByNumber(ctx, tenKblocksAgoNumber)
	if err != nil {
		return -1, err
	}

	theBlock, err := c.findBlockByTimeStamp(ctx, client, timestampInSeconds, latestNumber, latestTimestamp, older.Number.Int64(), older.Time.Int64())
	return theBlock, err
}

func (c *connectorCommon) findBlockByTimeStamp(ctx context.Context, eth *ethclient.Client, timestamp int64, highBlockNumber, highTimestamp, lowBlockNumber, lowTimestamp int64) (int64, error) {
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
			return lowBlockNumber - 1, nil
		} else {
			return lowBlockNumber, nil
		}
	}

	secondsPerBlock := highLowTimeDiff / blockNumberDiff
	distanceToTargetFromHigh := highTimestamp - timestamp
	blocksToJump := distanceToTargetFromHigh / secondsPerBlock
	c.logger.Info("eth block search delta", log.Int64("jump-backwards", blocksToJump))
	guess, err := eth.HeaderByNumber(ctx, big.NewInt(highBlockNumber-blocksToJump))
	if err != nil {
		return -1, err
	}

	guessTimestamp := guess.Time.Int64()
	guessBlockNumber := guess.Number.Int64()

	// create new relative block diff from guess (educated guess according to known local average)
	distanceFromTarget := timestamp - guessTimestamp
	blocksToJumpForNewLocalGuess := distanceFromTarget / secondsPerBlock
	guessLocalTarget := guessBlockNumber + blocksToJumpForNewLocalGuess
	newLocalToGuess, err := eth.HeaderByNumber(ctx, big.NewInt(guessLocalTarget))
	if err != nil {
		return -1, err
	}

	if guessTimestamp > timestamp {
		// jumping backwards more, we did not jump far enough
		return c.findBlockByTimeStamp(ctx, eth, timestamp, guessBlockNumber, guessTimestamp, newLocalToGuess.Number.Int64(), newLocalToGuess.Time.Int64())
	} else {
		// need to jump forward as we skipped it
		return c.findBlockByTimeStamp(ctx, eth, timestamp, newLocalToGuess.Number.Int64(), newLocalToGuess.Time.Int64(), guessBlockNumber, guessTimestamp)
	}
}
