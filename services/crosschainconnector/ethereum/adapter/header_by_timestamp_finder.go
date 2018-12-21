package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math"
	"math/big"
	"sync"
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
	cache  *EthBlocksSearchCache
	logger log.BasicLogger
}

func NewTimestampFetcher(bfh BlockHeaderFetcher, logger log.BasicLogger) *finder {
	f := &finder{
		bfh: bfh,
		cache: &EthBlocksSearchCache{
			latest:  EthBlock{},
			back10k: EthBlock{},
		},
		logger: logger,
	}

	return f
}

func (f *finder) getBlocksCacheAndRefreshIfNeeded(ctx context.Context, unixEpoch int64) (*EthBlocksSearchCache, error) {
	if f.cache.latest.timestamp < unixEpoch-1000 {

		if err := f.cache.refreshUsing(ctx, f.bfh); err != nil {
			return nil, err
		}
	}

	return f.cache, nil
}

func (f *finder) GetBlockByTimestamp(ctx context.Context, nano primitives.TimestampNano) (*big.Int, error) {
	timestampInSeconds := int64(nano) / int64(time.Second)
	// ethereum started around 2015/07/31
	if timestampInSeconds < 1438300800 {
		return nil, errors.New("cannot query before ethereum genesis")
	}

	cache, err := f.getBlocksCacheAndRefreshIfNeeded(ctx, timestampInSeconds)
	if err != nil {
		return nil, err
	}

	// TODO: (v1) move cache refresh timeout to config
	if cache.latest.timestamp+1000 < timestampInSeconds {
		return nil, errors.New("invalid request to get block, trying to get a block in the future (sync issues?)")
	}

	theBlock, err := f.findBlockByTimeStamp(ctx, f.bfh, timestampInSeconds, cache.back10k.number, cache.back10k.timestamp, cache.latest.number, cache.latest.timestamp)
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

type EthBlock struct {
	number    int64
	timestamp int64
}

type EthBlocksSearchCache struct {
	sync.Mutex
	latest     EthBlock
	back10k    EthBlock
	lastUpdate time.Time
}

func (c *EthBlocksSearchCache) refreshUsing(ctx context.Context, headerFetcher BlockHeaderFetcher) error {
	c.Lock()
	defer c.Unlock()

	latest, err := headerFetcher.HeaderByNumber(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to get latest block")
	}

	c.latest.timestamp = latest.Time.Int64()
	c.latest.number = latest.Number.Int64()

	// this was added to support simulations and tests, should not be relevant for production
	latestNum := latest.Number.Int64()
	latestNum -= 10000
	if latestNum < 0 {
		latestNum = 0
	}
	older, err := headerFetcher.HeaderByNumber(ctx, big.NewInt(latestNum))
	if err != nil {
		return errors.Wrap(err, "failed to get past reference block")
	}

	c.back10k.timestamp = older.Time.Int64()
	c.back10k.number = older.Number.Int64()

	c.lastUpdate = time.Now()

	return nil
}
