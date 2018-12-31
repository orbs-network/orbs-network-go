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

func (t *BlockTracker) incrementHeight() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.currentHeight++
	prevLatch := t.latch
	t.latch = make(chan struct{})
	close(prevLatch)
}

func (t *BlockTracker) ReachedHeight(height primitives.BlockHeight) {
	if uint64(height) != t.currentHeight+1 {
		panic(errors.Errorf("Block Tracker expected height %d but got height %d", t.currentHeight+1, height))
	}
	t.incrementHeight()
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

	if currentHeight >= requestedHeightUint { // requested block already committed
		return nil
	}

	if currentHeight+uint64(t.graceDistance) < requestedHeightUint { // requested block too far ahead, no grace
		return errors.Errorf("requested future block outside of grace range")
	}

	for currentHeight < requestedHeightUint {
		if t.fireOnWait != nil {
			t.fireOnWait()
		}
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), fmt.Sprintf("aborted while waiting for block at height %d", requestedHeight))
		case <-currentLatch:
			currentHeight, currentLatch = t.readAtomicHeightAndLatch()
		}
	}
	return nil
}

// shim for BlockWriter
type NopHeightReporter struct{}

func (_ NopHeightReporter) IncrementHeight()                            {}
func (_ NopHeightReporter) ReachedHeight(height primitives.BlockHeight) {}
