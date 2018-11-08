package adapter

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
)

type peer struct {
	listener adapter.TransportListener
	socket   chan [][]byte
}

func newPeer(bgCtx context.Context, logger log.BasicLogger) *peer {
	p := &peer{socket:make(chan [][]byte)}

	ctx, cancel := context.WithCancel(bgCtx)
	defer cancel()
	supervised.GoForever(ctx, logger, func() {
		select {
		case payloads := <-p.socket:
			p.listener.OnTransportMessageReceived(ctx, payloads)
		case <-ctx.Done():
			return
		}
	})

	return p
}

func (p *peer) send(payloads [][]byte) {
	//select {
	//case p.socket <- payloads:
	//default:
	//	// maybe listener is not yet ready
	//}
}

