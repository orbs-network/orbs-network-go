// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package timestampfinder

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math/big"
	"sync"
	"time"
)

const TIMESTAMP_FINDER_PROBABLE_RANGE_EFFICIENT = 1000
const TIMESTAMP_FINDER_PROBABLE_RANGE_INEFFICIENT = 10000
const TIMESTAMP_FINDER_MAX_STEPS = 1000
const TIMESTAMP_FINDER_ALLOWED_HEURISTIC_STEPS = 10

type TimestampFinder interface {
	FindBlockByTimestamp(ctx context.Context, referenceTimestampNano primitives.TimestampNano) (*big.Int, error)
}

type finder struct {
	logger          log.BasicLogger
	btg             BlockTimeGetter
	metrics         *timestampBlockFinderMetrics
	lastResultCache struct {
		sync.RWMutex
		below *BlockNumberAndTime
		above *BlockNumberAndTime
	}
}

type timestampBlockFinderMetrics struct {
	timeToFindBlock     *metric.Histogram
	stepsRequired       *metric.Rate
	totalTimesCalled    *metric.Gauge
	cacheHits           *metric.Gauge
	lastBlockInEthereum *metric.Gauge
	lastBlockFound      *metric.Gauge
	lastBlockTimeStamp  *metric.Gauge
}

func newTimestampFinderMetrics(factory metric.Factory) *timestampBlockFinderMetrics {
	return &timestampBlockFinderMetrics{
		totalTimesCalled:    factory.NewGauge("Ethereum.TimestampBlockFinder.TotalTimesCalled.Count"),
		stepsRequired:       factory.NewRate("Ethereum.TimestampBlockFinder.StepsRequired.Rate"),
		cacheHits:           factory.NewGauge("Ethereum.TimestampBlockFinder.CacheHits.Count"),
		lastBlockFound:      factory.NewGauge("Ethereum.TimestampBlockFinder.LastBlockFound.Number"),
		lastBlockTimeStamp:  factory.NewGauge("Ethereum.TimestampBlockFinder.LastBlockFound.TimeStamp.UnixEpoch"),
		lastBlockInEthereum: factory.NewGauge("Ethereum.TimestampBlockFinder.LastBlockInEthereum.Number"),
		timeToFindBlock:     factory.NewLatency("Ethereum.TimestampBlockFinder.TimeToFindBlock.Duration.Millis", 30*time.Second),
	}
}

func NewTimestampFinder(btg BlockTimeGetter, logger log.BasicLogger, metrics metric.Factory) *finder {
	return &finder{btg: btg, logger: logger, metrics: newTimestampFinderMetrics(metrics)}
}

func (f *finder) FindBlockByTimestamp(ctx context.Context, referenceTimestampNano primitives.TimestampNano) (*big.Int, error) {
	start := time.Now()
	f.metrics.totalTimesCalled.Inc()

	// TODO: find a better way to handle this, the simulator has no concept of block number
	if f.isEthereumSimulator() {
		return nil, nil
	}

	var err error
	below, above := f.getLastResultCache()

	f.logger.Info("ethereum timestamp finder starting",
		log.Uint64("reference-timestamp", referenceTimestampNano.KeyForMap()),
		log.Int64("below-cache-number", below.BlockNumber),
		log.Uint64("below-cache-timestamp", below.BlockTimeNano.KeyForMap()),
		log.Int64("above-cache-number", above.BlockNumber),
		log.Uint64("above-cache-timestamp", above.BlockTimeNano.KeyForMap()))

	// attempt to return the last result immediately without any queries (for efficiency)
	if algoDidReachResult(referenceTimestampNano, below, above) {
		f.metrics.cacheHits.Inc()
		return f.returnConfirmedResult(referenceTimestampNano, below, above, 0)
	}

	// extend above if needed
	if !(referenceTimestampNano < above.BlockTimeNano) {
		above, err = algoExtendAbove(ctx, referenceTimestampNano, f.btg)
		if err != nil {
			return nil, err
		}
		f.metrics.lastBlockInEthereum.Update(above.BlockNumber)
	}

	// extend below if needed
	if !(1 <= below.BlockNumber && below.BlockTimeNano <= referenceTimestampNano) {
		below, err = algoExtendBelow(ctx, referenceTimestampNano, below.BlockNumber, above.BlockNumber, f.btg)
		if err != nil {
			return nil, err
		}
	}

	// try reducing further and further until finding the result
	for steps := 1; steps < TIMESTAMP_FINDER_MAX_STEPS; steps++ {
		if ctx.Err() == context.Canceled {
			return nil, errors.Wrap(ctx.Err(), "aborting search")
		}

		f.logger.Info("ethereum timestamp finder step",
			log.Int("step", steps),
			log.Uint64("reference-timestamp", referenceTimestampNano.KeyForMap()),
			log.Int64("below-number", below.BlockNumber),
			log.Uint64("below-timestamp", below.BlockTimeNano.KeyForMap()),
			log.Int64("above-number", above.BlockNumber),
			log.Uint64("above-timestamp", above.BlockTimeNano.KeyForMap()))

		// did we finally reach the result?
		if algoDidReachResult(referenceTimestampNano, below, above) {
			f.metrics.timeToFindBlock.RecordSince(start)
			return f.returnConfirmedResult(referenceTimestampNano, below, above, steps)
		}

		// make sure for sanity the result is still inside the range
		err = algoVerifyResultInsideRange(referenceTimestampNano, below, above)
		if err != nil {
			return nil, err
		}

		// make the range smaller
		distBefore := above.BlockNumber - below.BlockNumber
		below, above, err = algoReduceRange(ctx, referenceTimestampNano, below, above, f.btg, steps)
		if err != nil {
			return nil, err
		}
		distAfter := above.BlockNumber - below.BlockNumber

		// make sure we are converging
		if distAfter >= distBefore {
			f.logger.Error("ethereum timestamp finder is not converging (did not reduce range)",
				log.Int("step", steps),
				log.Uint64("reference-timestamp", referenceTimestampNano.KeyForMap()),
				log.Int64("new-below-number", below.BlockNumber),
				log.Uint64("new-below-timestamp", below.BlockTimeNano.KeyForMap()),
				log.Int64("new-above-number", above.BlockNumber),
				log.Uint64("new-above-timestamp", above.BlockTimeNano.KeyForMap()))
		}
	}

	return nil, errors.Errorf("ethereum timestamp finder went over maximum steps %d, reference timestamp %d", TIMESTAMP_FINDER_MAX_STEPS, referenceTimestampNano)
}

func (f *finder) returnConfirmedResult(referenceTimestampNano primitives.TimestampNano, below BlockNumberAndTime, above BlockNumberAndTime, steps int) (*big.Int, error) {
	f.setLastResultCache(below, above)
	f.metrics.stepsRequired.Measure(int64(steps))
	// the block below is the one we actually return as result
	f.logger.Info("ethereum timestamp finder found result",
		log.Int("steps", steps),
		log.Uint64("reference-timestamp", referenceTimestampNano.KeyForMap()),
		log.Int64("result-number", below.BlockNumber),
		log.Uint64("result-timestamp", below.BlockTimeNano.KeyForMap()))
	return big.NewInt(below.BlockNumber), nil
}

func (f *finder) getLastResultCache() (below BlockNumberAndTime, above BlockNumberAndTime) {
	f.lastResultCache.RLock()
	defer f.lastResultCache.RUnlock()
	if f.lastResultCache.below != nil {
		below = *f.lastResultCache.below
	}
	if f.lastResultCache.above != nil {
		above = *f.lastResultCache.above
	}
	return
}

func (f *finder) setLastResultCache(below BlockNumberAndTime, above BlockNumberAndTime) {
	f.lastResultCache.Lock()
	defer f.lastResultCache.Unlock()
	f.lastResultCache.below = &BlockNumberAndTime{BlockNumber: below.BlockNumber, BlockTimeNano: below.BlockTimeNano}
	f.lastResultCache.above = &BlockNumberAndTime{BlockNumber: above.BlockNumber, BlockTimeNano: above.BlockTimeNano}
	f.metrics.lastBlockFound.Update(below.BlockNumber)
	f.metrics.lastBlockTimeStamp.Update(int64(time.Duration(below.BlockTimeNano) / time.Second))
}

func (f *finder) isEthereumSimulator() bool {
	if ethBasedBlockTimeGetter, ok := f.btg.(*EthereumBasedBlockTimeGetter); ok {
		if _, ok := ethBasedBlockTimeGetter.ethereum.(*adapter.EthereumSimulator); ok {
			return true
		}
	}
	return false
}
