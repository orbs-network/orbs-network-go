package statestorage

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"sync/atomic"
	"time"
)

type BlockTracker struct {
	currentHeight uint64
	graceDistance uint64
	timeout       time.Duration

	latch chan struct{}

	// following fields are for tests only
	enteredSelectSignalForTests chan int
	selectIterationsForTests    int
}

func NewBlockTracker(startingHeight uint64, graceDist uint16, timeout time.Duration) *BlockTracker {
	return &BlockTracker{
		currentHeight: startingHeight,
		graceDistance: uint64(graceDist),
		timeout:       timeout,
		latch:         make(chan struct{}),
	}
}

func (t *BlockTracker) IncrementHeight() {
	atomic.AddUint64(&t.currentHeight, 1) // increment atomically in case two goroutines commit concurrently
	prevLatch := t.latch
	t.latch = make(chan struct{})
	close(prevLatch)
}

func (t *BlockTracker) WaitForBlock(requestedHeight primitives.BlockHeight) error {

	rh := uint64(requestedHeight)
	if t.currentHeight >= rh { // requested block already committed
		return nil
	}

	if t.currentHeight < rh-t.graceDistance { // requested block too far ahead, no grace
		return errors.Errorf("requested future block outside of grace range")
	}

	timer := time.NewTimer(t.timeout)
	defer timer.Stop()

	for t.currentHeight < rh { // sit on latch until desired height or t.o.
		t.notifyEnterSelectForTests()
		select {
		case <-timer.C:
			return errors.Errorf("timed out waiting for block at height %v", requestedHeight)
		case <-t.latch:
		}
	}
	return nil
}

func (t *BlockTracker) notifyEnterSelectForTests() {
	if t.enteredSelectSignalForTests != nil {
		t.selectIterationsForTests++
		select {
		case t.enteredSelectSignalForTests <- t.selectIterationsForTests:
		default:
		}
	}
}
