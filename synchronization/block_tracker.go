package synchronization

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type BlockTracker struct {
	graceDistance primitives.BlockHeight
	timeout       time.Duration

	mutex         sync.RWMutex
	currentHeight primitives.BlockHeight
	latch         chan struct{}

	// following fields are for tests only
	enteredSelectSignalForTests chan int
	selectIterationsForTests    int
}

func NewBlockTracker(startingHeight primitives.BlockHeight, graceDistance primitives.BlockHeight, timeout time.Duration) *BlockTracker {
	return &BlockTracker{
		currentHeight: startingHeight,
		graceDistance: graceDistance,
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

func (t *BlockTracker) readAtomicHeightAndLatch() (primitives.BlockHeight, chan struct{}) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.currentHeight, t.latch
}

func (t *BlockTracker) WaitForBlock(requestedHeight primitives.BlockHeight) error {

	currentHeight, currentLatch := t.readAtomicHeightAndLatch()

	if currentHeight >= requestedHeight { // requested block already committed
		return nil
	}

	if currentHeight+t.graceDistance < requestedHeight { // requested block too far ahead, no grace
		return errors.Errorf("requested future block outside of grace range")
	}

	timer := time.NewTimer(t.timeout)
	defer timer.Stop()

	for currentHeight < requestedHeight {
		t.notifyEnterSelectForTests()
		select {
		case <-timer.C:
			return errors.Errorf("timed out waiting for block at height %v", requestedHeight)
		case <-currentLatch:
			currentHeight, currentLatch = t.readAtomicHeightAndLatch()
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
