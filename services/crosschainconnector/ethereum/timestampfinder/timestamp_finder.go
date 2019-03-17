package timestampfinder

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math/big"
	"sync"
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
	lastResultCache struct {
		sync.RWMutex
		below *BlockNumberAndTime
		above *BlockNumberAndTime
	}
}

func NewTimestampFinder(btg BlockTimeGetter, logger log.BasicLogger) *finder {
	return &finder{btg: btg, logger: logger}
}

func (f *finder) FindBlockByTimestamp(ctx context.Context, referenceTimestampNano primitives.TimestampNano) (*big.Int, error) {
	// TODO: find a better way to handle this, the simulator has no concept of block number
	if f.isEthereumSimulator() {
		return nil, nil
	}

	var err error
	below, above := f.getLastResultCache()

	f.logger.Info("ethereum timestamp finder starting", log.Stringable("reference-timestamp", referenceTimestampNano), log.Int64("below-cache-number", below.BlockNumber), log.Stringable("below-cache-timestamp", below.BlockTimeNano), log.Int64("above-cache-number", above.BlockNumber), log.Stringable("above-cache-timestamp", above.BlockTimeNano))

	// attempt to return the last result immediately without any queries (for efficiency)
	if algoDidReachResult(referenceTimestampNano, below, above) {
		return f.returnConfirmedResult(referenceTimestampNano, below, above, 0)
	}

	// extend above if needed
	if !(referenceTimestampNano < above.BlockTimeNano) {
		above, err = algoExtendAbove(ctx, referenceTimestampNano, f.btg)
		if err != nil {
			return nil, err
		}
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

		f.logger.Info("ethereum timestamp finder step", log.Int("step", steps), log.Stringable("reference-timestamp", referenceTimestampNano), log.Int64("below-number", below.BlockNumber), log.Stringable("below-timestamp", below.BlockTimeNano), log.Int64("above-number", above.BlockNumber), log.Stringable("above-timestamp", above.BlockTimeNano))

		// did we finally reach the result?
		if algoDidReachResult(referenceTimestampNano, below, above) {
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
			f.logger.Error("ethereum timestamp finder is not converging (did not reduce range)", log.Int("step", steps), log.Stringable("reference-timestamp", referenceTimestampNano), log.Int64("new-below-number", below.BlockNumber), log.Stringable("new-below-timestamp", below.BlockTimeNano), log.Int64("new-above-number", above.BlockNumber), log.Stringable("new-above-timestamp", above.BlockTimeNano))
		}
	}

	return nil, errors.Errorf("ethereum timestamp finder went over maximum steps %d, reference timestamp %v", TIMESTAMP_FINDER_MAX_STEPS, referenceTimestampNano)
}

func (f *finder) returnConfirmedResult(referenceTimestampNano primitives.TimestampNano, below BlockNumberAndTime, above BlockNumberAndTime, steps int) (*big.Int, error) {
	f.setLastResultCache(below, above)
	// the block below is the one we actually return as result
	f.logger.Info("ethereum timestamp finder found result", log.Int("steps", steps), log.Stringable("reference-timestamp", referenceTimestampNano), log.Int64("result-number", below.BlockNumber), log.Stringable("result-timestamp", below.BlockTimeNano))
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
}

func (f *finder) isEthereumSimulator() bool {
	if ethBasedBlockTimeGetter, ok := f.btg.(*EthereumBasedBlockTimeGetter); ok {
		if _, ok := ethBasedBlockTimeGetter.ethereum.(*adapter.EthereumSimulator); ok {
			return true
		}
	}
	return false
}
