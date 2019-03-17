package tcp

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/pkg/errors"
	"sync"
)

type transportQueue struct {
	channel        chan *adapter.TransportData // replace this buffered channel with github.com/phf/go-queue if we don't want maxSizeMessages (and its pre allocation)
	networkAddress string
	maxBytes       int
	maxMessages    int
	disabled       bool // not under mutex on purpose

	protected struct {
		sync.Mutex
		bytesLeft int
	}
}

func NewTransportQueue(maxSizeBytes int, maxSizeMessages int) *transportQueue {
	q := &transportQueue{
		channel:     make(chan *adapter.TransportData, maxSizeMessages),
		maxBytes:    maxSizeBytes,
		maxMessages: maxSizeMessages,
	}
	q.protected.bytesLeft = maxSizeBytes
	return q
}

func (q *transportQueue) Push(data *adapter.TransportData) error {
	if q.disabled {
		return nil
	}

	err := q.consumeBytes(data)
	if err != nil {
		return err
	}

	select {
	case q.channel <- data:
		return nil
	default:
		return errors.Errorf("failed to push to queue - full with %d messages", q.maxMessages)
	}
}

func (q *transportQueue) Pop(ctx context.Context) *adapter.TransportData {
	select {
	case <-ctx.Done():
		return nil
	case res := <-q.channel:
		q.releaseBytes(res)
		return res
	}
}

func (q *transportQueue) Clear(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case res := <-q.channel:
			q.releaseBytes(res)
		default:
			return
		}
	}
}

func (q *transportQueue) Disable() {
	q.disabled = true
}

func (q *transportQueue) Enable() {
	q.disabled = false
}

func (q *transportQueue) consumeBytes(data *adapter.TransportData) error {
	q.protected.Lock()
	defer q.protected.Unlock()

	if data.TotalSize() > q.protected.bytesLeft {
		return errors.Errorf("failed to push %d bytes to queue - full with %d bytes out of %d bytes", data.TotalSize(), q.maxBytes-q.protected.bytesLeft, q.maxBytes)
	}

	q.protected.bytesLeft -= data.TotalSize()
	return nil
}

func (q *transportQueue) releaseBytes(data *adapter.TransportData) {
	q.protected.Lock()
	defer q.protected.Unlock()

	q.protected.bytesLeft += data.TotalSize()
}
