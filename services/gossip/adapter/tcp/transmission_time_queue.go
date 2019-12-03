package tcp

import (
	"sync"
	"time"
)

const MaxTransmissionTimeQueueSize = 100

type TransmissionTimeQueue struct {
	times        chan time.Time
	droppedCount uint64
	lock         sync.Mutex
}

func newTransmissionTimeQueue() *TransmissionTimeQueue {
	return &TransmissionTimeQueue{
		times:        make(chan time.Time, MaxTransmissionTimeQueueSize),
		droppedCount: 0,
		lock:         sync.Mutex{},
	}
}

func (c *TransmissionTimeQueue) push(sentAt time.Time) {
	c.lock.Lock()
	defer c.lock.Unlock()

	select {
	case c.times <- sentAt:
		return
	default:
		<-c.times
		c.droppedCount++
		c.times <- sentAt
	}
}

func (c *TransmissionTimeQueue) pop() (time.Time, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.droppedCount > 0 {
		c.droppedCount--
		return time.Time{}, false
	}

	select {
	case t := <-c.times:
		return t, true
	default:
		return time.Time{}, false // called pop before push
	}
}
