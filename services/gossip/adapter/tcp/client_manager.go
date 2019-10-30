package tcp

import (
	"context"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"sync"
	"sync/atomic"
)

type GossipPeers map[string]config.GossipPeer

type clientMetrics struct {
	outgoingConnectionSendErrors      *metric.Gauge
	outgoingConnectionKeepaliveErrors *metric.Gauge
	outgoingConnectionSendQueueErrors *metric.Gauge
	activeOutgoingConnections         *metric.Gauge

	outgoingMessageSize *metric.Histogram
}

type clientManager struct {
	sync.RWMutex
	peers          map[string]*clientConnection
	gossipPeers    GossipPeers // this is important - use own copy of peers, otherwise nodes in e2e tests that run in process can mutate each other's gossipPeers
	logger         log.Logger
	metrics        *clientMetrics
	atomicConfig   atomic.Value
	metricRegistry metric.Registry
}

func newClientManager(logger log.Logger, registry metric.Registry, config config.GossipTransportConfig) *clientManager {
	c := &clientManager{
		logger:         logger,
		peers:          make(map[string]*clientConnection),
		gossipPeers:    make(GossipPeers),
		metrics:        createClientMetrics(registry),
		metricRegistry: registry,
	}
	c.atomicConfig.Store(config)

	return c
}

func createClientMetrics(registry metric.Registry) *clientMetrics {
	return &clientMetrics{
		outgoingConnectionSendErrors:      registry.NewGauge("Gossip.OutgoingConnection.SendErrors.Count"),
		outgoingConnectionKeepaliveErrors: registry.NewGauge("Gossip.OutgoingConnection.KeepaliveErrors.Count"),
		outgoingConnectionSendQueueErrors: registry.NewGauge("Gossip.OutgoingConnection.SendQueueErrors.Count"),
		activeOutgoingConnections:         registry.NewGauge("Gossip.OutgoingConnection.Active.Count"),
		outgoingMessageSize:               registry.NewHistogram("Gossip.OutgoingConnection.MessageSize.Bytes", MAX_PAYLOAD_SIZE_BYTES),
	}
}

func (c *clientManager) GracefulShutdown(shutdownContext context.Context) {
	c.Lock()
	c.Unlock()
	for _, client := range c.peers {
		client.disconnect()
	}
}

func (c *clientManager) WaitUntilShutdown(shutdownContext context.Context) {
	c.Lock()
	defer c.Unlock()
	for _, client := range c.peers {
		select {
		case <-client.closed:
		case <-shutdownContext.Done():
			c.logger.Error("failed shutting down within shutdown context")
		}
	}
}

func (c *clientManager) config() config.GossipTransportConfig {
	if c, ok := c.atomicConfig.Load().(config.GossipTransportConfig); ok {
		return c
	}

	return nil
}

// note that bgCtx MUST be a long-running background context - if it's a short lived context, the new connection will die as soon as
// the context is done
func (c *clientManager) connectForever(bgCtx context.Context, peerNodeAddress string, peer config.GossipPeer) {
	c.Lock()
	defer c.Unlock()

	if c.config().NodeAddress().KeyForMap() != peerNodeAddress {
		c.gossipPeers[peerNodeAddress] = peer
		client := newClientConnection(peer, c.logger, c.metricRegistry, c.metrics, c.config())
		c.peers[peerNodeAddress] = client
		client.connect(bgCtx)
	}
}

func (c *clientManager) updateTopology(bgCtx context.Context, newPeers GossipPeers) {
	oldPeers := c.readOldPeerConfig()
	peersToRemove, peersToAdd := peerDiff(oldPeers, newPeers)

	c.disconnectAll(bgCtx, peersToRemove)

	for peerNodeAddress, peer := range peersToAdd {
		c.connectForever(bgCtx, peerNodeAddress, peer)
	}
}

func (c *clientManager) connectAll(parent context.Context) {
	for peerNodeAddress, peer := range c.gossipPeers {
		c.connectForever(parent, peerNodeAddress, peer)
	}
}

func (c *clientManager) disconnectAll(ctx context.Context, peersToDisconnect GossipPeers) {
	c.Lock()
	defer c.Unlock()
	for key, peer := range peersToDisconnect {
		delete(c.gossipPeers, key)
		if client, found := c.peers[key]; found {
			select {
			case <-client.disconnect():
				delete(c.peers, key)
			case <-ctx.Done():
				c.logger.Info("system shutdown while waiting for clients to disconnect")
			}
		} else {
			c.logger.Error("attempted to disconnect a client that was not connected", log.String("missing-peer", peer.HexOrbsAddress()))
		}

	}
}

func (c *clientManager) readOldPeerConfig() GossipPeers {
	c.RLock()
	defer c.RUnlock()
	return c.gossipPeers
}

// TODO(https://github.com/orbs-network/orbs-network-go/issues/182): we are not currently respecting any intents given in ctx (added in context refactor)
func (c *clientManager) send(ctx context.Context, data *adapter.TransportData) error {
	c.RLock()
	defer c.RUnlock()

	switch data.RecipientMode {
	case gossipmessages.RECIPIENT_LIST_MODE_BROADCAST:
		for _, client := range c.peers {
			client.addDataToOutgoingPeerQueue(ctx, data)
			c.metrics.outgoingMessageSize.Record(int64(data.TotalSize()))
		}
		return nil
	case gossipmessages.RECIPIENT_LIST_MODE_LIST:
		for _, recipientPublicKey := range data.RecipientNodeAddresses {
			if client, found := c.peers[recipientPublicKey.KeyForMap()]; found {
				client.addDataToOutgoingPeerQueue(ctx, data)
				c.metrics.outgoingMessageSize.Record(int64(data.TotalSize()))
			} else {
				return errors.Errorf("unknown recipient public key: %s", recipientPublicKey.String())
			}
		}
		return nil
	case gossipmessages.RECIPIENT_LIST_MODE_ALL_BUT_LIST:
		panic("Not implemented")
	}
	return errors.Errorf("unknown recipient mode: %s", data.RecipientMode.String())
}
