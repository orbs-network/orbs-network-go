// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
)

const MAX_PAYLOADS_IN_MESSAGE = 100000
const MAX_PAYLOAD_SIZE_BYTES = 20 * 1024 * 1024
const SEND_QUEUE_MAX_MESSAGES = 1000
const SEND_QUEUE_MAX_BYTES = 20 * 1024 * 1024

var LogTag = log.String("adapter", "gossip")

type DirectTransport struct {
	govnr.TreeSupervisor

	logger log.Logger

	outgoingConnections *outgoingConnections
	server              *transportServer
}

func NewDirectTransport(parentCtx context.Context, config config.GossipTransportConfig, parentLogger log.Logger, registry metric.Registry) *DirectTransport {
	logger := parentLogger.WithTags(LogTag)
	t := &DirectTransport{
		logger:              logger,
		outgoingConnections: newOutgoingConnections(logger, registry, config),
		server:              newServer(config, parentLogger.WithTags(log.String("component", "tcp-transport-server")), registry),
	}

	t.Supervise(t.server)
	t.Supervise(t.outgoingConnections)

	t.outgoingConnections.connectAll(parentCtx, config.GossipPeers()) // client goroutines
	t.server.startSupervisedMainLoop(parentCtx)                       // server goroutine

	return t
}

func (t *DirectTransport) UpdateTopology(bgCtx context.Context, newPeers GossipPeers) {
	t.outgoingConnections.updateTopology(bgCtx, newPeers)
}

func (t *DirectTransport) RegisterListener(listener adapter.TransportListener, listenerNodeAddress primitives.NodeAddress) {
	t.server.Lock()
	defer t.server.Unlock()

	t.server.listener = listener
}

func (t *DirectTransport) Send(ctx context.Context, data *adapter.TransportData) error {
	return t.outgoingConnections.send(ctx, data)
}

func (t *DirectTransport) GetServerPort() int {
	return t.server.getPort()
}

func (t *DirectTransport) IsServerListening() bool {
	return t.server.IsListening()
}

func (t *DirectTransport) allOutgoingQueuesEnabled() bool {
	t.outgoingConnections.RLock()
	defer t.outgoingConnections.RUnlock()

	for _, client := range t.outgoingConnections.activeConnections {
		if client.queue.disabled() {
			return false
		}
	}
	return true
}

func (t *DirectTransport) GracefulShutdown(shutdownContext context.Context) {
	t.logger.Info("Shutting down")
	t.outgoingConnections.GracefulShutdown(shutdownContext)
	t.server.GracefulShutdown(shutdownContext)
}

func (t *DirectTransport) waitFor(shutdownContext context.Context, closed chan struct{}) {
	select {
	case <-closed:
	case <-shutdownContext.Done():
		t.logger.Error("failed shutting down within shutdown context")
	}
}

func calcPaddingSize(size uint32) uint32 {
	const contentAlignment = 4
	alignedSize := (size + contentAlignment - 1) / contentAlignment * contentAlignment
	return alignedSize - size
}
