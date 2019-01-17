/*
Package memory provides an in-memory implementation of the Gossip Transport adapter, meant for usage in fast tests that
should not use the TCP-based adapter, such as acceptance tests or sociable unit tests, or in other in-process network use cases
*/
package memory

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"sync"
)

type message struct {
	payloads     [][]byte
	traceContext *trace.Context
}

type peer struct {
	socket   chan message
	listener chan adapter.TransportListener
}

type memoryTransport struct {
	sync.RWMutex
	peers map[string]*peer
}

func NewTransport(ctx context.Context, logger log.BasicLogger, federation map[string]config.FederationNode) *memoryTransport {
	transport := &memoryTransport{peers: make(map[string]*peer)}

	transport.Lock()
	defer transport.Unlock()
	for _, node := range federation {
		nodeAddress := node.NodeAddress().KeyForMap()
		transport.peers[nodeAddress] = newPeer(ctx, logger.WithTags(log.String("peer-listener", nodeAddress)))
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

func newPeer(bgCtx context.Context, logger log.BasicLogger) *peer {
	p := &peer{socket: make(chan message, 1000), listener: make(chan adapter.TransportListener)} // channel is buffered on purpose, otherwise the whole network is synced on transport

	supervised.GoForever(bgCtx, logger, func() {
		// wait till we have a listener attached
		select {
		case l := <-p.listener:
			p.acceptUsing(bgCtx, l)
		case <-bgCtx.Done():
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
	case <-ctx.Done():
		return
	}
}

func (p *peer) acceptUsing(bgCtx context.Context, listener adapter.TransportListener) {
	for {
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
