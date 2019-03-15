package timestampfinder

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"math/big"
	"time"
)

func secondsToNano(seconds int64) primitives.TimestampNano {
	return primitives.TimestampNano(seconds) * primitives.TimestampNano(time.Second)
}

func algoDidReachResult(referenceTimestampNano primitives.TimestampNano, below BlockNumberAndTime, above BlockNumberAndTime) bool {
	if below.BlockNumber+1 != above.BlockNumber {
		return false
	}
	if 1 <= below.BlockNumber && below.BlockTimeNano <= referenceTimestampNano && referenceTimestampNano < above.BlockTimeNano {
		return true
	}
	return false
}

func algoVerifyResultInsideRange(referenceTimestampNano primitives.TimestampNano, below BlockNumberAndTime, above BlockNumberAndTime) error {
	if below.BlockNumber < 1 {
		return errors.Errorf("ethereum timestamp finder range corrupt, below is %d (below 1)", below.BlockNumber)
	}
	if !(below.BlockNumber+1 < above.BlockNumber) {
		return errors.Errorf("ethereum timestamp finder range corrupt, below %d is too close to above %d", below.BlockNumber, above.BlockNumber)
	}
	if referenceTimestampNano < below.BlockTimeNano {
		return errors.Errorf("ethereum timestamp finder range corrupt, below timestamp %s is above reference %s", below.BlockTimeNano, referenceTimestampNano)
	}
	if above.BlockTimeNano <= referenceTimestampNano {
		return errors.Errorf("ethereum timestamp finder range corrupt, above timestamp %s is below reference %s", above.BlockTimeNano, referenceTimestampNano)
	}
	return nil
}

func algoExtendAbove(ctx context.Context, referenceTimestampNano primitives.TimestampNano, btg BlockTimeGetter) (BlockNumberAndTime, error) {
	latest, err := btg.GetTimestampForLatestBlock(ctx)
	if err != nil {
		return BlockNumberAndTime{}, err
	}
	if latest == nil {
		return BlockNumberAndTime{}, errors.New("ethereum timestamp finder received nil as latest block from getter")
	}
	if !(referenceTimestampNano < latest.BlockTimeNano) {
		return BlockNumberAndTime{}, errors.Errorf("the latest ethereum block %d at %v is not newer than the reference timestamp %v, must wait for newer blocks to be mined", latest.BlockNumber, latest.BlockTimeNano, referenceTimestampNano)
	}
	return *latest, nil
}

func algoExtendBelow(ctx context.Context, referenceTimestampNano primitives.TimestampNano, belowBlockNumber int64, aboveBlockNumber int64, btg BlockTimeGetter) (BlockNumberAndTime, error) {
	startBlockNumber := belowBlockNumber
	if startBlockNumber < 1 {
		startBlockNumber = aboveBlockNumber
	}

	cursorBlockNumberAttempts := []int64{
		startBlockNumber - TIMESTAMP_FINDER_PROBABLE_RANGE_EFFICIENT,
		startBlockNumber - TIMESTAMP_FINDER_PROBABLE_RANGE_INEFFICIENT,
		1,
	}

	for _, cursorBlockNumber := range cursorBlockNumberAttempts {
		if cursorBlockNumber >= 1 {

			cursor, err := btg.GetTimestampForBlockNumber(ctx, big.NewInt(cursorBlockNumber))
			if err != nil {
				return BlockNumberAndTime{}, err
			}
			if cursor == nil {
				return BlockNumberAndTime{}, errors.New("ethereum timestamp finder received nil as cursor block from getter")
			}
			if 1 <= cursor.BlockNumber && cursor.BlockTimeNano <= referenceTimestampNano {
				return *cursor, nil
			}
			if cursor.BlockNumber == 1 {
				return BlockNumberAndTime{}, errors.Errorf("the first ethereum block %d at %v is newer than the reference timestamp %v", cursor.BlockNumber, cursor.BlockTimeNano, referenceTimestampNano)
			}

		}
	}

	// not supposed to be able to get here
	return BlockNumberAndTime{}, errors.Errorf("unable to extend below, reference timestamp is %v, above is %d, below is %d", referenceTimestampNano, aboveBlockNumber, belowBlockNumber)
}

func algoReduceRange(ctx context.Context, referenceTimestampNano primitives.TimestampNano, below BlockNumberAndTime, above BlockNumberAndTime, btg BlockTimeGetter, step int) (BlockNumberAndTime, BlockNumberAndTime, error) {
	allowedHeuristics := false
	if step <= TIMESTAMP_FINDER_ALLOWED_HEURISTIC_STEPS {
		allowedHeuristics = true
	}

	// get the proposed cursor (somewhere in the middle)
	var cursorBlockNumber int64
	if allowedHeuristics {
		cursorBlockNumber = algoGetCursorWithHeuristics(referenceTimestampNano, below, above)
	} else {
		cursorBlockNumber = algoGetCursorWithBinarySearch(below, above)
	}

	// get the block at the cursor
	cursor, err := btg.GetTimestampForBlockNumber(ctx, big.NewInt(cursorBlockNumber))
	if err != nil {
		return BlockNumberAndTime{}, BlockNumberAndTime{}, err
	}
	if cursor == nil {
		return BlockNumberAndTime{}, BlockNumberAndTime{}, errors.New("ethereum timestamp finder received nil as cursor block from getter")
	}

	// make the range smaller
	if referenceTimestampNano < cursor.BlockTimeNano {
		return below, *cursor, nil
	} else {
		return *cursor, above, nil
	}
}

func algoGetCursorWithHeuristics(referenceTimestampNano primitives.TimestampNano, below BlockNumberAndTime, above BlockNumberAndTime) int64 {
	distInBlocks := above.BlockNumber - below.BlockNumber
	distInNano := above.BlockTimeNano - below.BlockTimeNano
	if distInNano == 0 {
		// not supposed to happen according to algoVerifyResultInsideRange
		return above.BlockNumber - 1
	}
	res := below.BlockNumber + distInBlocks*int64(referenceTimestampNano-below.BlockTimeNano)/int64(distInNano)
	if res <= below.BlockNumber {
		return below.BlockNumber + 1
	}
	if res >= above.BlockNumber {
		return above.BlockNumber - 1
	}
	return res
}

func algoGetCursorWithBinarySearch(below BlockNumberAndTime, above BlockNumberAndTime) int64 {
	distInBlocks := above.BlockNumber - below.BlockNumber
	// distInBlocks must be >= 2 according to algoVerifyResultInsideRange
	return below.BlockNumber + distInBlocks/2
}
