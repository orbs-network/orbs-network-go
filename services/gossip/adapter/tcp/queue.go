// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"sync"
)

type transportQueue struct {
	channel        chan *adapter.TransportData // replace this buffered channel with github.com/phf/go-queue if we don't want maxSizeMessages (and its pre allocation)
	networkAddress string
	maxBytes       int
	maxMessages    int

	protected struct {
		sync.Mutex
		bytesLeft int
		disabled  bool // not under mutex on purpose
	}
	usagePercentageMetric *metric.Gauge
}

func NewTransportQueue(maxSizeBytes int, maxSizeMessages int, metricFactory metric.Registry, peerNodeAddress string, logger log.Logger) *transportQueue {
	q := &transportQueue{
		channel:     make(chan *adapter.TransportData, maxSizeMessages),
		maxBytes:    maxSizeBytes,
		maxMessages: maxSizeMessages,
	}
	q.protected.bytesLeft = maxSizeBytes

	// round-about way to remove old queue metric if exists
	queueUsageName := fmt.Sprintf("Gossip.OutgoingConnection.QueueUsage.%s.Percent", peerNodeAddress)
	queueUsageMetric := metricFactory.Get(queueUsageName)
	if queueUsageMetric != nil {
		logger.Info("TransportQueue ctor issue", log.Error(errors.Errorf("Metric %s still existed when new connection created", queueUsageName)))
	}
	metricFactory.Remove(queueUsageMetric)
	q.usagePercentageMetric = metricFactory.NewGaugeWithPrometheusName(queueUsageName, fmt.Sprintf("Gossip.OutgoingConnection.Queue.Usage.%s.Percent", peerNodeAddress))

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
	q.protected.Lock()
	defer q.protected.Unlock()
	q.protected.disabled = true
}

func (q *transportQueue) Enable() {
	q.protected.Lock()
	defer q.protected.Unlock()
	q.protected.disabled = false
}

func (q *transportQueue) disabled() bool {
	q.protected.Lock()
	defer q.protected.Unlock()
	return q.protected.disabled
}

func (q *transportQueue) OnNewConnection(ctx context.Context) {
	q.Clear(ctx)
	q.Enable()
}

func NewQueueFullError(bytesAttempted int, bytesInQueue int, queueSize int) error {
	return errors.Errorf("failed to push %d bytes to queue - full with %d bytes out of %d bytes", bytesAttempted, bytesInQueue, queueSize)
}

func (q *transportQueue) consumeBytes(data *adapter.TransportData) error {
	q.protected.Lock()
	defer q.protected.Unlock()

	if q.protected.disabled {
		return errors.Errorf("attempted to push to a disabled queue")
	}

	dataSize := data.TotalSize()
	if dataSize > q.protected.bytesLeft {
		return NewQueueFullError(dataSize, q.maxBytes-q.protected.bytesLeft, q.maxBytes)
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
