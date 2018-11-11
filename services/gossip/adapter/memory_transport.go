package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"sync"
)

type peer struct {
	socket   chan [][]byte
	listener chan TransportListener
}

type memoryTransport struct {
	sync.RWMutex
	byPublicKey map[string]*peer
}

/*
Package adapter provides an in-memory implementation of the Gossip Transport adapter, meant for usage in fast tests that
should not use the TCP-based adapter, such as acceptance tests or sociable unit tests, or in other in-process network use cases
*/
func NewMemoryTransport(ctx context.Context, logger log.BasicLogger, federation map[string]config.FederationNode) *memoryTransport {
	peers := &memoryTransport{byPublicKey: make(map[string]*peer)}

	peers.Lock()
	defer peers.Unlock()
	for _, node := range federation {
		key := node.NodePublicKey().KeyForMap()
		peers.byPublicKey[key] = newPeer(ctx, logger.WithTags(log.String("peer-listener", key)))
	}

	return peers
}

func (p *memoryTransport) RegisterListener(listener TransportListener, key primitives.Ed25519PublicKey) {
	p.Lock()
	defer p.Unlock()
	p.byPublicKey[string(key)].attach(listener)
}

func (p *memoryTransport) Send(ctx context.Context, data *TransportData) error {
	switch data.RecipientMode {

	case gossipmessages.RECIPIENT_LIST_MODE_BROADCAST:
		for key, peer := range p.byPublicKey {
			if key != data.SenderPublicKey.KeyForMap() {
				peer.send(ctx, data)
			}
		}

	case gossipmessages.RECIPIENT_LIST_MODE_LIST:
		for _, k := range data.RecipientPublicKeys {
			p.byPublicKey[k.KeyForMap()].send(ctx, data)
		}

	case gossipmessages.RECIPIENT_LIST_MODE_ALL_BUT_LIST:
		panic("Not implemented")
	}

	return nil
}

func newPeer(bgCtx context.Context, logger log.BasicLogger) *peer {
	p := &peer{socket: make(chan [][]byte), listener: make(chan TransportListener)}

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

func (p *peer) attach(listener TransportListener) {
	p.listener <- listener
}

func (p *peer) send(ctx context.Context, data *TransportData) {
	select {
	case p.socket <- data.Payloads:
	case <- ctx.Done():
		return
	}
}

func (p *peer) acceptUsing(ctx context.Context, listener TransportListener) {
	for {
		select {
		case payloads := <-p.socket:
			listener.OnTransportMessageReceived(ctx, payloads)
		case <-ctx.Done():
			return
		}
	}
}
