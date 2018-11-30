package synchronization

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"sync"
)

var LogTag = log.Service("block-tracker")

type BlockTracker struct {
	graceDistance uint16 // this is not primitives.BlockHeight on purpose, to indicate that grace distance should be small

	mutex         sync.RWMutex
	currentHeight uint64 // this is not primitives.BlockHeight so as to avoid unnecessary casts
	latch         chan struct{}
	logger        log.BasicLogger
	fireOnWait    func() // used by unit test
}

func NewBlockTracker(parent log.BasicLogger, startingHeight uint64, graceDist uint16) *BlockTracker {

	logger := parent.WithTags(LogTag)

	return &BlockTracker{
		logger:        logger,
		currentHeight: startingHeight,
		graceDistance: graceDist,
		latch:         make(chan struct{}),
	}
}

func (t *BlockTracker) IncrementHeight() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.currentHeight++
	prevLatch := t.latch
	t.latch = make(chan struct{})
	close(prevLatch)
}

func (t *BlockTracker) readAtomicHeightAndLatch() (uint64, chan struct{}) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.currentHeight, t.latch
}

// waits until we reach a block at the specified height, or until the context is closed
// to wait until some timeout, pass a child context with a deadline
func (t *BlockTracker) WaitForBlock(ctx context.Context, requestedHeight primitives.BlockHeight) error {
	requestedHeightUint := uint64(requestedHeight)
	currentHeight, currentLatch := t.readAtomicHeightAndLatch()
	t.logger.Info("WaitForBlock() start", log.Uint64("current-height", currentHeight), log.Uint64("requestedHeight", requestedHeightUint))

	if currentHeight >= requestedHeightUint { // requested block already committed
		return nil
	}

	if currentHeight+uint64(t.graceDistance) < requestedHeightUint { // requested block too far ahead, no grace
		return errors.Errorf("requested future block outside of grace range")
	}

	for currentHeight < requestedHeightUint {
		t.logger.Info("WaitForBlock() Before wait for block", log.Uint64("current-height", currentHeight), log.Uint64("requestedHeight", requestedHeightUint))
		if t.fireOnWait != nil {
			t.fireOnWait()
		}
		select {
		case <-ctx.Done():
			t.logger.Info("WaitForBlock() ctx.Done() called", log.Uint64("current-height", currentHeight), log.Uint64("requestedHeight", requestedHeightUint))
			return errors.Wrap(ctx.Err(), fmt.Sprintf("aborted while waiting for block at height %v", requestedHeight))
		case <-currentLatch:
			t.logger.Info("WaitForBlock() Latch released", log.Uint64("current-height", currentHeight), log.Uint64("requestedHeight", requestedHeightUint))
			currentHeight, currentLatch = t.readAtomicHeightAndLatch()
			t.logger.Info("WaitForBlock() read new height", log.Uint64("current-height", currentHeight), log.Uint64("requestedHeight", requestedHeightUint))
		}
	}
	t.logger.Info("WaitForBlock() return", log.Uint64("current-height", currentHeight), log.Uint64("requestedHeight", requestedHeightUint))
	return nil
}
