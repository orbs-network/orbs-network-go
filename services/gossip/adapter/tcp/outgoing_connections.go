package tcp

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"sync"
)

type outgoingConnectionMetrics struct {
	sendErrors      *metric.Gauge
	KeepaliveErrors *metric.Gauge
	sendQueueErrors *metric.Gauge
	activeCount     *metric.Gauge

	messageSize *metric.HistogramInt64
}

type outgoingConnections struct {
	sync.RWMutex
	activeConnections map[string]*outgoingConnection
	peerTopology      adapter.TransportPeers // this is important - we use own copy of peer configuration, otherwise nodes in e2e tests that run in-process can mutate each other's peerTopology
	logger            log.Logger
	metrics           *outgoingConnectionMetrics
	config            timingsConfig
	metricRegistry    metric.Registry
	nodeAddress       primitives.NodeAddress
}

func newOutgoingConnections(logger log.Logger, registry metric.Registry, config config.GossipTransportConfig) *outgoingConnections {
	c := &outgoingConnections{
		logger:            logger,
		activeConnections: make(map[string]*outgoingConnection),
		peerTopology:      make(adapter.TransportPeers),
		metrics:           createOutgoingConnectionMetrics(registry),
		metricRegistry:    registry,
		nodeAddress:       config.NodeAddress(),
		config:            config,
	}

	return c
}

func createOutgoingConnectionMetrics(registry metric.Registry) *outgoingConnectionMetrics {
	return &outgoingConnectionMetrics{
		sendErrors:      registry.NewGauge("Gossip.OutgoingConnection.SendErrors.Count"),
		KeepaliveErrors: registry.NewGauge("Gossip.OutgoingConnection.KeepaliveErrors.Count"),
		sendQueueErrors: registry.NewGauge("Gossip.OutgoingConnection.SendQueueErrors.Count"),
		activeCount:     registry.NewGauge("Gossip.OutgoingConnection.Active.Count"),
		messageSize:     registry.NewHistogramInt64("Gossip.OutgoingConnection.MessageSize.Bytes", MAX_PAYLOAD_SIZE_BYTES),
	}
}

func (c *outgoingConnections) GracefulShutdown(shutdownContext context.Context) {
	c.Lock()
	defer c.Unlock()
	for _, client := range c.activeConnections {
		client.disconnect()
	}
}

func (c *outgoingConnections) WaitUntilShutdown(shutdownContext context.Context) {
	c.Lock()
	defer c.Unlock()
	for _, client := range c.activeConnections {
		select {
		case <-client.closed:
		case <-shutdownContext.Done():
			c.logger.Error("failed shutting down within shutdown context")
		}
	}
}

func (c *outgoingConnections) updateTopology(bgCtx context.Context, newTopology adapter.TransportPeers) {
	c.Lock()
	defer c.Unlock()
	// If not in topology disconnect from outer world
	if _, isInNewTopology := newTopology[c.nodeAddress.KeyForMap()]; !isInNewTopology {
		c.disconnectAllUnderLock(bgCtx, c.peerTopology)
		return
	}

	peersToRemove, peersToAdd := adapter.PeerDiff(c.peerTopology, newTopology)

	c.disconnectAllUnderLock(bgCtx, peersToRemove)

	for peerNodeAddress, peer := range peersToAdd {
		c.connectForeverUnderLock(bgCtx, peerNodeAddress, peer)
	}
}

// note that bgCtx MUST be a long-running background context - if it's a short lived context, the new connection will die as soon as
// the context is done
func (c *outgoingConnections) connectForeverUnderLock(bgCtx context.Context, peerNodeAddress string, peer adapter.TransportPeer) {
	if c.nodeAddress.KeyForMap() != peerNodeAddress {
		c.peerTopology[peerNodeAddress] = peer
		client := newOutgoingConnection(peer, c.logger, c.metricRegistry, c.metrics, c.config)
		c.activeConnections[peerNodeAddress] = client
		client.connect(bgCtx)
	}
}

func (c *outgoingConnections) disconnectAllUnderLock(ctx context.Context, peersToDisconnect adapter.TransportPeers) {
	for key, peer := range peersToDisconnect {
		delete(c.peerTopology, key)
		if client, found := c.activeConnections[key]; found {
			select {
			case <-client.disconnect():
				delete(c.activeConnections, key)
			case <-ctx.Done():
				c.logger.Info("system shutdown while waiting for clients to disconnect")
			}
		} else {
			c.logger.Error("attempted to disconnect a client that was not connected", log.String("missing-peer", peer.HexOrbsAddress()))
		}

	}
}

var DataExceedsCapacityError = errors.Errorf("Data exceeds allowed size %d", SEND_QUEUE_MAX_BYTES)

// TODO(https://github.com/orbs-network/orbs-network-go/issues/182): we are not currently respecting any intents given in ctx (added in context refactor)
func (c *outgoingConnections) send(ctx context.Context, data *adapter.TransportData) error {
	c.RLock()
	defer c.RUnlock()

	if data.TotalSize() > SEND_QUEUE_MAX_BYTES {
		return DataExceedsCapacityError
	}

	switch data.RecipientMode {
	case gossipmessages.RECIPIENT_LIST_MODE_BROADCAST:
		for _, client := range c.activeConnections {
			client.addDataToOutgoingPeerQueue(ctx, data)
			c.metrics.messageSize.Record(int64(data.TotalSize()))
		}
		return nil
	case gossipmessages.RECIPIENT_LIST_MODE_LIST:
		for _, recipientPublicKey := range data.RecipientNodeAddresses {
			if client, found := c.activeConnections[recipientPublicKey.KeyForMap()]; found {
				client.addDataToOutgoingPeerQueue(ctx, data)
				c.metrics.messageSize.Record(int64(data.TotalSize()))
			} else {
				err := errors.Errorf("unknown recipient public key: %s", recipientPublicKey.String())
				c.logger.Error("failed sending gossip message", log.Error(err), log.Stringable("recipient-public-key", recipientPublicKey))
			}
		}
		return nil
	case gossipmessages.RECIPIENT_LIST_MODE_ALL_BUT_LIST:
		panic("Not implemented")
	}
	return errors.Errorf("unknown recipient mode: %s", data.RecipientMode.String())
}
