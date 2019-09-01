// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package gossip

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
)

type handlerFunc func(ctx context.Context, header *gossipmessages.Header, payloads [][]byte)

type gossipMessage struct {
	header   *gossipmessages.Header
	payloads [][]byte
}

type meteredTopicChannel struct {
	ch      chan gossipMessage
	size    *metric.Gauge
	inQueue *metric.Gauge
	logger  log.Logger
	name    string
}

func (c *meteredTopicChannel) send(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
	logger := c.logger.WithTags(trace.LogFieldFrom(ctx))
	logger.Info("transport message received", log.Stringable("header", header), log.Int("topic-size", len(c.ch)))
	c.ch <- gossipMessage{header: header, payloads: payloads} //TODO should the channel have *gossipMessage as type?
	c.updateMetrics()
}

func (c *meteredTopicChannel) updateMetrics() {
	c.inQueue.Update(int64(len(c.ch)))
}

func (c *meteredTopicChannel) run(ctx context.Context, logger log.Logger, handler handlerFunc) *govnr.ForeverHandle {
	return govnr.Forever(ctx, c.name, logfields.GovnrErrorer(logger), func() {
		for {
			select {
			case <-ctx.Done():
				c.drain()
				return
			case message := <-c.ch:
				handler(ctx, message.header, message.payloads)
				c.updateMetrics()
			}
		}
	})

}

func (c *meteredTopicChannel) drain() {
	for {
		select {
		case <-c.ch:
		default:
			return
		}
	}
}

func newMeteredTopicChannel(name string, registry metric.Registry, logger log.Logger) *meteredTopicChannel {
	capacity := 10
	sizeGauge := registry.NewGauge("Gossip.Topic." + name + ".QueueSize")
	sizeGauge.Update(int64(capacity))
	return &meteredTopicChannel{
		ch:      make(chan gossipMessage, capacity),
		size:    sizeGauge,
		inQueue: registry.NewGauge("Gossip.Topic." + name + ".MessagesInQueue"),
		name:    fmt.Sprintf("%s topic handler", name),
		logger:  logger.WithTags(log.String("gossip-topic", name)),
	}
}

type gossipMessageDispatcher struct {
	transactionRelay   *meteredTopicChannel
	blockSync          *meteredTopicChannel
	leanHelix          *meteredTopicChannel
	benchmarkConsensus *meteredTopicChannel
}

// These channels are buffered because we don't want to assume that the topic consumers behave nicely
// In fact, Block Sync should create a new one-off goroutine per "server request", Consensus should read messages immediately and store them in its own queue,
// and Transaction Relay shouldn't block for long anyway.
func newMessageDispatcher(registry metric.Registry, logger log.Logger) (d *gossipMessageDispatcher) {
	d = &gossipMessageDispatcher{
		transactionRelay:   newMeteredTopicChannel("TransactionRelay", registry, logger),
		blockSync:          newMeteredTopicChannel("BlockSync", registry, logger),
		leanHelix:          newMeteredTopicChannel("LeanHelixConsensus", registry, logger),
		benchmarkConsensus: newMeteredTopicChannel("BenchmarkConsensus", registry, logger),
	}
	return
}

func (d *gossipMessageDispatcher) dispatch(ctx context.Context, logger log.Logger, header *gossipmessages.Header, payloads [][]byte) {
	ch, err := d.get(header.Topic())
	if err != nil {
		logger.Error("no message channel found", log.Error(err))
		return
	}

	ch.send(ctx, header, payloads)

}

func (d *gossipMessageDispatcher) runHandler(ctx context.Context, logger log.Logger, topic gossipmessages.HeaderTopic, handler handlerFunc) *govnr.ForeverHandle {
	topicChannel, err := d.get(topic)
	if err != nil {
		logger.Error("no message channel found", log.Error(err))
		panic(err)
	} else {
		return topicChannel.run(ctx, logger, handler)
	}
}

func (d *gossipMessageDispatcher) get(topic gossipmessages.HeaderTopic) (*meteredTopicChannel, error) {
	switch topic {
	case gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY:
		return d.transactionRelay, nil
	case gossipmessages.HEADER_TOPIC_BLOCK_SYNC:
		return d.blockSync, nil
	case gossipmessages.HEADER_TOPIC_LEAN_HELIX:
		return d.leanHelix, nil
	case gossipmessages.HEADER_TOPIC_BENCHMARK_CONSENSUS:
		return d.benchmarkConsensus, nil
	default:
		return nil, errors.Errorf("no message channel for topic %d", topic)
	}
}
