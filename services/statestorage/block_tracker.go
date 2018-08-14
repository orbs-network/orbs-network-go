package statestorage

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/pkg/errors"
	"sync"
	"time"
)

type BlockTracker struct {
	graceDistance uint64
	timeout       time.Duration

	mutex         sync.RWMutex
	currentHeight uint64
	latch         chan struct{}

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
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.currentHeight++
	prevLatch := t.latch
	t.latch = make(chan struct{})
	close(prevLatch)
}

func (t *BlockTracker) getCurrentHeightAndLatchAtomic() (uint64, chan struct{}) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.currentHeight, t.latch
}

func (t *BlockTracker) WaitForBlock(requestedHeight primitives.BlockHeight) error {

	rh := uint64(requestedHeight)
	ch, cl := t.getCurrentHeightAndLatchAtomic()

	if ch >= rh { // requested block already committed
		return nil
	}

	if ch < rh-t.graceDistance { // requested block too far ahead, no grace
		return errors.Errorf("requested future block outside of grace range")
	}

	timer := time.NewTimer(t.timeout)
	defer timer.Stop()

	for ; ch < rh; ch, cl = t.getCurrentHeightAndLatchAtomic() { // sit on latch until desired height or t.o.
		t.notifyEnterSelectForTests()
		select {
		case <-timer.C:
			return errors.Errorf("timed out waiting for block at height %v", requestedHeight)
		case <-cl:
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
