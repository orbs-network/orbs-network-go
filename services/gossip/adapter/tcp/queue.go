// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
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
	usagePercentageMetric *metric.Gauge
}

func NewTransportQueue(maxSizeBytes int, maxSizeMessages int, metricFactory metric.Factory) *transportQueue {
	q := &transportQueue{
		channel:     make(chan *adapter.TransportData, maxSizeMessages),
		maxBytes:    maxSizeBytes,
		maxMessages: maxSizeMessages,
	}
	q.protected.bytesLeft = maxSizeBytes

	q.usagePercentageMetric = metricFactory.NewGauge("Gossip.OutgoingConnection.Queue.Usage.Percent")

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

	dataSize := data.TotalSize()
	if dataSize > q.protected.bytesLeft {
		return errors.Errorf("failed to push %d bytes to queue - full with %d bytes out of %d bytes", dataSize, q.maxBytes-q.protected.bytesLeft, q.maxBytes)
	}

	q.protected.bytesLeft -= dataSize
	q.updateUsageMetric()
	return nil
}

func (q *transportQueue) releaseBytes(data *adapter.TransportData) {
	q.protected.Lock()
	defer q.protected.Unlock()

	q.protected.bytesLeft += data.TotalSize()
	q.updateUsageMetric()
}

func (q *transportQueue) updateUsageMetric() {
	bytesUsed := q.maxBytes - q.protected.bytesLeft
	q.usagePercentageMetric.Update(int64(bytesUsed * 100 / q.maxBytes))
}
