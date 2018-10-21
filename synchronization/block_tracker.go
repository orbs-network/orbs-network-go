package synchronization

import (
	"context"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type BlockTracker struct {
	graceDistance uint16 // this is not primitives.BlockHeight on purpose, to indicate that grace distance should be small
	timeout       time.Duration

	mutex         sync.RWMutex
	currentHeight uint64 // this is not primitives.BlockHeight so as to avoid unnecessary casts
	latch         chan struct{}

	fireOnWait func() // used by unit test
}

func NewBlockTracker(startingHeight uint64, graceDist uint16, timeout time.Duration) *BlockTracker {
	return &BlockTracker{
		currentHeight: startingHeight,
		graceDistance: graceDist,
		timeout:       timeout,
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

func (t *BlockTracker) WaitForBlock(ctx context.Context, requestedHeight primitives.BlockHeight) error {

	requestedHeightUint := uint64(requestedHeight)
	currentHeight, currentLatch := t.readAtomicHeightAndLatch()

	if currentHeight >= requestedHeightUint { // requested block already committed
		return nil
	}

	if currentHeight+uint64(t.graceDistance) < requestedHeightUint { // requested block too far ahead, no grace
		return errors.Errorf("requested future block outside of grace range")
	}

	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	
	for currentHeight < requestedHeightUint {
		if t.fireOnWait != nil {
			t.fireOnWait()
		}
		select {
		case <-ctx.Done():
			return errors.Errorf("timed out waiting for block at height %v", requestedHeight)
		case <-currentLatch:
			currentHeight, currentLatch = t.readAtomicHeightAndLatch()
		}
	}
	return nil
}
