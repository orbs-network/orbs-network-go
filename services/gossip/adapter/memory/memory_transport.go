// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

/*
Package memory provides an in-memory implementation of the Gossip Transport adapter, meant for usage in fast tests that
should not use the TCP-based adapter, such as acceptance tests or sociable unit tests, or in other in-process network use cases
*/
package memory

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/scribe/log"
	"sync"
)

const SEND_QUEUE_MAX_MESSAGES = 1000

var LogTag = log.String("adapter", "gossip")

type message struct {
	payloads     [][]byte
	traceContext *trace.Context
}

type peer struct {
	socket   chan message
	listener chan adapter.TransportListener
	logger   log.Logger
}

type memoryTransport struct {
	sync.RWMutex
	peers map[string]*peer
}

func NewTransport(ctx context.Context, logger log.Logger, validators map[string]config.ValidatorNode) *memoryTransport {
	transport := &memoryTransport{peers: make(map[string]*peer)}

	transport.Lock()
	defer transport.Unlock()
	for _, node := range validators {
		nodeAddress := node.NodeAddress().KeyForMap()
		transport.peers[nodeAddress] = newPeer(ctx, logger.WithTags(LogTag, log.Stringable("node", node.NodeAddress())), len(validators))
	}

	return transport
}

func (p *memoryTransport) RegisterListener(listener adapter.TransportListener, nodeAddress primitives.NodeAddress) {
	p.Lock()
	defer p.Unlock()
	p.peers[string(nodeAddress)].attach(listener)
}

func (p *memoryTransport) Send(ctx context.Context, data *adapter.TransportData) error {
	switch data.RecipientMode {

	case gossipmessages.RECIPIENT_LIST_MODE_BROADCAST:
		for key, peer := range p.peers {
			if key != data.SenderNodeAddress.KeyForMap() {
				peer.send(ctx, data)
			}
		}

	case gossipmessages.RECIPIENT_LIST_MODE_LIST:
		for _, k := range data.RecipientNodeAddresses {
			p.peers[k.KeyForMap()].send(ctx, data)
		}

	case gossipmessages.RECIPIENT_LIST_MODE_ALL_BUT_LIST:
		panic("Not implemented")
	}

	return nil
}

func newPeer(ctx context.Context, logger log.Logger, totalPeers int) *peer {
	p := &peer{
		// channel is buffered on purpose, otherwise the whole network is synced on transport
		// we also multiply by number of peers because we have one logical "socket" for combined traffic from all peers together
		// we decided not to separate sockets between every 2 peers (like tcp transport) because:
		//  1) nodes in production tend to broadcast messages, so traffic is usually combined anyways
		//  2) the implementation complexity to mimic tcp transport isn't justified
		socket:   make(chan message, SEND_QUEUE_MAX_MESSAGES*totalPeers),
		listener: make(chan adapter.TransportListener),
		logger:   logger,
	}

	supervised.GoForever(ctx, logger, func() {
		// wait till we have a listener attached
		select {
		case l := <-p.listener:
			p.acceptUsing(ctx, l)
		case <-ctx.Done():
			return
		}
	})

	return p
}

func (p *peer) attach(listener adapter.TransportListener) {
	p.listener <- listener
}

func (p *peer) send(ctx context.Context, data *adapter.TransportData) {
	tracingContext, _ := trace.FromContext(ctx)
	select {
	case p.socket <- message{payloads: data.Payloads, traceContext: tracingContext}:
		return
	case <-ctx.Done():
		return
	default:
		p.logger.Error("memory transport send buffer is full")
		return
	}
}

func (p *peer) acceptUsing(bgCtx context.Context, listener adapter.TransportListener) {
	for {
		p.logger.Info("reading a message from socket", log.Int("socket-size", len(p.socket)))
		select {
		case message := <-p.socket:
			receive(bgCtx, listener, message)
		case <-bgCtx.Done():
			return
		}
	}
}

func receive(bgCtx context.Context, listener adapter.TransportListener, message message) {
	ctx, cancel := context.WithCancel(bgCtx)
	defer cancel()
	traceContext := contextFrom(ctx, message)
	listener.OnTransportMessageReceived(traceContext, message.payloads)
}

func contextFrom(ctx context.Context, message message) context.Context {
	if message.traceContext == nil {
		return trace.NewContext(ctx, "memory-transport")
	} else {
		return trace.PropagateContext(ctx, message.traceContext)
	}
}
