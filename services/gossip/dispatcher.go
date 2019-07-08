// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gossip

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/scribe/log"
)

type gossipMessage struct {
	header   *gossipmessages.Header
	payloads [][]byte
}

type meteredTopicChannel struct {
	ch      chan gossipMessage
	size    *metric.Gauge
	inQueue *metric.Gauge
}

func (c *meteredTopicChannel) send(header *gossipmessages.Header, payloads [][]byte) {
	c.ch <- gossipMessage{header: header, payloads: payloads} //TODO should the channel have *gossipMessage as type?
	c.updateMetrics()
}

func (c *meteredTopicChannel) updateMetrics() {
	c.size.Update(int64(len(c.ch)))
}

func newMeteredTopicChannel(name string, registry metric.Registry) *meteredTopicChannel {
	capacity := 10
	sizeGauge := registry.NewGauge("Gossip.Topic." + name + ".QueueSize")
	sizeGauge.Update(int64(capacity))
	return &meteredTopicChannel{
		ch:      make(chan gossipMessage, capacity),
		size:    sizeGauge,
		inQueue: registry.NewGauge("Gossip.Topic." + name + ".MessagesInQueue"),
	}
}

type gossipMessageDispatcher map[gossipmessages.HeaderTopic]*meteredTopicChannel

// These channels are buffered because we don't want to assume that the topic consumers behave nicely
// In fact, Block Sync should create a new one-off goroutine per "server request", Consensus should read messages immediately and store them in its own queue,
// and Transaction Relay shouldn't block for long anyway.
func makeMessageDispatcher(registry metric.Registry) (d gossipMessageDispatcher) {
	d = make(gossipMessageDispatcher)
	d[gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY] = newMeteredTopicChannel("TransactionRelay", registry)
	d[gossipmessages.HEADER_TOPIC_BLOCK_SYNC] = newMeteredTopicChannel("BlockSync", registry)
	d[gossipmessages.HEADER_TOPIC_LEAN_HELIX] = newMeteredTopicChannel("LeanHelixConsensus", registry)
	d[gossipmessages.HEADER_TOPIC_BENCHMARK_CONSENSUS] = newMeteredTopicChannel("BenchmarkConsensus", registry)
	return
}

func (d gossipMessageDispatcher) dispatch(logger log.Logger, header *gossipmessages.Header, payloads [][]byte) {
	ch := d[header.Topic()]
	if ch == nil {
		logger.Error("no message channel for topic", log.Int("topic", int(header.Topic())))
		return
	}

	ch.send(header, payloads)
}

func (d gossipMessageDispatcher) runHandler(ctx context.Context, logger log.Logger, topic gossipmessages.HeaderTopic, handler func(ctx context.Context, header *gossipmessages.Header, payloads [][]byte)) {
	topicChannel := d[topic]
	if topicChannel == nil {
		panic(fmt.Sprintf("no message channel for topic %d", topic))
	} else {
		supervised.GoForever(ctx, logger, func() {
			for {
				select {
				case <-ctx.Done():
					return
				case message := <-topicChannel.ch:
					handler(ctx, message.header, message.payloads)
					topicChannel.updateMetrics()
				}
			}
		})
	}
}
