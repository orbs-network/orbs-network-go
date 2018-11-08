package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
)

type peer struct {
	socket   chan [][]byte
	listener chan adapter.TransportListener
}

func newPeer(bgCtx context.Context, logger log.BasicLogger) *peer {
	p := &peer{socket:make(chan [][]byte), listener: make(chan adapter.TransportListener)}

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

func (p *peer) send(payloads [][]byte) {
	p.socket <- payloads
}

func (p *peer) acceptUsing(ctx context.Context, listener adapter.TransportListener) {
	for {
		select {
		case payloads := <-p.socket:
			listener.OnTransportMessageReceived(ctx, payloads)
		case <-ctx.Done():
			return
		}
	}
}

