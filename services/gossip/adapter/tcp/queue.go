package tcp

import (
	"context"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/pkg/errors"
	"sync"
)

type transportQueue struct {
	channel     chan *adapter.TransportData // replace this buffered channel with github.com/phf/go-queue if we don't want maxSizeMessages (and its pre allocation)
	maxBytes    int
	maxMessages int

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

func (q *transportQueue) consumeBytes(data *adapter.TransportData) error {
	dataBytes := totalBytesInData(data)

	q.protected.Lock()
	defer q.protected.Unlock()

	if dataBytes > q.protected.bytesLeft {
		return errors.Errorf("failed to push %d bytes to queue - full with %d bytes", dataBytes, q.maxBytes-q.protected.bytesLeft)
	}

	q.protected.bytesLeft -= dataBytes
	return nil
}

func (q *transportQueue) releaseBytes(data *adapter.TransportData) {
	dataBytes := totalBytesInData(data)

	q.protected.Lock()
	defer q.protected.Unlock()

	q.protected.bytesLeft += dataBytes
}

func totalBytesInData(data *adapter.TransportData) (res int) {
	for _, payload := range data.Payloads {
		res += len(payload)
	}
	return
}
