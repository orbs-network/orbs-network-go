// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"encoding/hex"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/memory"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestDirectTransport_HandlesStartupWithEmptyPeerList(t *testing.T) {
	address := keys.EcdsaSecp256K1KeyPairForTests(0).NodeAddress()
	cfg := config.ForDirectTransportTests(address, make(adapter.GossipPeers), 20*time.Hour /*disable keep alive*/, 1*time.Second)
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		topology := memory.NewTopologyProvider(cfg, harness.Logger)
		transport := NewDirectTransport(ctx, topology, cfg, harness.Logger, metric.NewRegistry())
		harness.Supervise(transport)
		defer transport.GracefulShutdown(ctx)

		require.True(t, test.Eventually(test.EVENTUALLY_ADAPTER_TIMEOUT, func() bool {
			return transport.IsServerListening()
		}), "server did not start")
	})
}

func TestDirectTransport_SupportsTopologyChangeInRuntime(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		harness.AllowErrorsMatching("failed sending gossip message") // because the test will send to node3 which is not in topology

		node1 := aNode(ctx, harness.Logger)
		node2 := aNode(ctx, harness.Logger)
		node3 := aNode(ctx, harness.Logger)
		node4 := aNode(ctx, harness.Logger)
		superviseAll(harness, node1, node2, node3, node4)
		defer shutdownAll(ctx, node1, node2, node3, node4)

		waitForAllNodesToSatisfy(t, "server did not start", func(node *nodeHarness) bool { return node.transport.IsServerListening() }, node1, node2, node3, node4)

		firstTopology := aTopologyContaining(node1, node2, node3)
		node1.updateTopology(ctx, firstTopology)
		node2.updateTopology(ctx, firstTopology)
		node3.updateTopology(ctx, firstTopology)

		waitForAllNodesToSatisfy(t,
			"expected all nodes to have peers added",
			func(node *nodeHarness) bool { return len(node.transport.outgoingConnections.activeConnections) > 0 },
			node1, node2, node3)

		waitForAllNodesToSatisfy(t,
			"expected all outgoing queues to become enabled after topology change",
			func(node *nodeHarness) bool { return node.transport.allOutgoingQueuesEnabled() },
			node1, node2, node3)

		node1.requireSendsSuccessfullyTo(t, ctx, node2)
		node2.requireSendsSuccessfullyTo(t, ctx, node1)
		node2.requireSendsSuccessfullyTo(t, ctx, node3)

		secondTopology := aTopologyContaining(node1, node2, node4)
		node1.updateTopology(ctx, secondTopology)
		node2.updateTopology(ctx, secondTopology)
		node4.updateTopology(ctx, secondTopology)

		waitForAllNodesToSatisfy(t,
			"expected all nodes to have peers added",
			func(node *nodeHarness) bool { return len(node.transport.outgoingConnections.activeConnections) > 0 },
			node1, node2, node4)

		waitForAllNodesToSatisfy(t,
			"expected all outgoing queues to become enabled after topology change",
			func(node *nodeHarness) bool { return node.transport.allOutgoingQueuesEnabled() },
			node1, node2, node4)

		node1.requireSendsSuccessfullyTo(t, ctx, node4)
		node1.requireSendsSuccessfullyTo(t, ctx, node2)
		node2.listener.ExpectNotReceive()

		node2.transport.Send(ctx, &adapter.TransportData{
			SenderNodeAddress:      node2.address,
			RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
			RecipientNodeAddresses: []primitives.NodeAddress{node3.address},
			Payloads:               aMessage(),
		})

		require.NoError(t, test.ConsistentlyVerify(test.EVENTUALLY_ADAPTER_TIMEOUT, node1.listener, node2.listener, node3.listener), "node 2 was able to send a message to node 3 which is no longer a part of its topology")
	})
}

func TestDirectTransport_SupportsBroadcastTransmissions(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		node1 := aNode(ctx, harness.Logger)
		node2 := aNode(ctx, harness.Logger)
		node3 := aNode(ctx, harness.Logger)
		superviseAll(harness, node1, node2, node3)
		defer shutdownAll(ctx, node1, node2, node3)

		waitForAllNodesToSatisfy(t, "server did not start", func(node *nodeHarness) bool { return node.transport.IsServerListening() }, node1, node2, node3)

		firstTopology := aTopologyContaining(node1, node2, node3)
		node1.updateTopology(ctx, firstTopology)
		node2.updateTopology(ctx, firstTopology)
		node3.updateTopology(ctx, firstTopology)

		waitForAllNodesToSatisfy(t,
			"expected all nodes to have peers added",
			func(node *nodeHarness) bool { return len(node.transport.outgoingConnections.activeConnections) > 0 },
			node1, node2, node3)

		waitForAllNodesToSatisfy(t,
			"expected all outgoing queues to become enabled after topology change",
			func(node *nodeHarness) bool { return node.transport.allOutgoingQueuesEnabled() },
			node1, node2, node3)

		payloads := aMessage()

		node1.listener.ExpectNotReceive()
		node2.listener.ExpectReceive(payloads)
		node3.listener.ExpectReceive(payloads)
		require.NoError(t, node1.transport.Send(ctx, &adapter.TransportData{
			SenderNodeAddress: node1.address,
			RecipientMode:     gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
			Payloads:          payloads,
		}))

		require.NoError(t, test.ConsistentlyVerify(test.EVENTUALLY_ADAPTER_TIMEOUT, node1.listener), "message was sent to self node")
		require.NoError(t, test.EventuallyVerify(test.EVENTUALLY_ADAPTER_TIMEOUT, node2.listener, node3.listener), "message was not sent to target node")
	})
}

func TestDirectTransport_FailsGracefullyIfMulticastFailedToSendToASingleRecipient(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		harness.AllowErrorsMatching("failed sending gossip message") // because the test will send to an arbitrary recipient which is not in topology

		node1 := aNode(ctx, harness.Logger)
		node2 := aNode(ctx, harness.Logger)
		superviseAll(harness, node1, node2)
		defer shutdownAll(ctx, node1, node2)

		waitForAllNodesToSatisfy(t, "server did not start", func(node *nodeHarness) bool { return node.transport.IsServerListening() }, node1, node2)

		firstTopology := aTopologyContaining(node1, node2)
		node1.updateTopology(ctx, firstTopology)
		node2.updateTopology(ctx, firstTopology)

		waitForAllNodesToSatisfy(t,
			"expected all nodes to have peers added",
			func(node *nodeHarness) bool { return len(node.transport.outgoingConnections.activeConnections) > 0 },
			node1, node2)

		waitForAllNodesToSatisfy(t,
			"expected all outgoing queues to become enabled after topology change",
			func(node *nodeHarness) bool { return node.transport.allOutgoingQueuesEnabled() },
			node1, node2)

		payloads := aMessage()

		node2.listener.ExpectReceive(payloads)
		require.NoError(t, node1.transport.Send(ctx, &adapter.TransportData{
			SenderNodeAddress:      node1.address,
			RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
			RecipientNodeAddresses: []primitives.NodeAddress{{0x1}, node2.address},
			Payloads:               payloads,
		}))

		require.NoError(t, test.EventuallyVerify(test.EVENTUALLY_ADAPTER_TIMEOUT, node2.listener), "message was not sent to target node")
	})
}

func TestDirectTransport_TestAutoUpdate(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		harness.AllowErrorsMatching("failed sending gossip message") // because the test will send to an arbitrary recipient which is not in topology

		node1 := aNode(ctx, harness.Logger)
		node2 := aNode(ctx, harness.Logger)
		superviseAll(harness, node1, node2)
		defer shutdownAll(ctx, node1, node2)

		waitForAllNodesToSatisfy(t, "server did not start", func(node *nodeHarness) bool { return node.transport.IsServerListening() }, node1, node2)

		firstTopology := aTopologyContaining(node1, node2)
		node1.topologyProvider.UpdateTopologyFromPeers(firstTopology) // update only the internal topology provider
		node2.topologyProvider.UpdateTopologyFromPeers(firstTopology) // update only the internal topology provider

		waitForAllNodesToSatisfy(t,
			"expected all nodes to have peers added",
			func(node *nodeHarness) bool { return len(node.transport.outgoingConnections.activeConnections) > 0 },
			node1, node2)

		waitForAllNodesToSatisfy(t,
			"expected all outgoing queues to become enabled after topology change",
			func(node *nodeHarness) bool { return node.transport.allOutgoingQueuesEnabled() },
			node1, node2)
	})
}

type nodeHarness struct {
	topologyProvider *memory.TopologyProvider
	transport        *DirectTransport
	address          primitives.NodeAddress
	listener         *testkit.MockTransportListener
}

func (n *nodeHarness) requireSendsSuccessfullyTo(t *testing.T, ctx context.Context, other *nodeHarness) {
	payloads := aMessage()

	other.listener.ExpectReceive(payloads)
	require.NoError(t, n.transport.Send(ctx, &adapter.TransportData{
		SenderNodeAddress:      n.address,
		RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
		RecipientNodeAddresses: []primitives.NodeAddress{other.address},
		Payloads:               payloads,
	}))

	require.NoError(t, test.EventuallyVerify(test.EVENTUALLY_ADAPTER_TIMEOUT, other.listener), "message was not sent to target node")
}

func (n *nodeHarness) toGossipPeer() adapter.GossipPeer {
	return adapter.NewGossipPeer(n.transport.GetServerPort(), "127.0.0.1", hex.EncodeToString(n.address))
}

func (n *nodeHarness) updateTopology(ctx context.Context, peers adapter.GossipPeers)  {
	n.topologyProvider.UpdateTopologyFromPeers(peers)
	n.transport.UpdateTopology(ctx)
}

func waitForAllNodesToSatisfy(t *testing.T, message string, predicate func(node *nodeHarness) bool, nodes ...*nodeHarness) {
	require.True(t, test.Eventually(1*time.Second, func() bool {
		ok := true
		for _, node := range nodes {
			ok = ok && predicate(node)
		}
		return ok
	}), message)
}

func aMessage() [][]byte {
	header := (&gossipmessages.HeaderBuilder{
		Topic:         gossipmessages.HEADER_TOPIC_LEAN_HELIX,
		RecipientMode: gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
	}).Build()
	message := &gossipmessages.LeanHelixMessage{
		Content: []byte{},
	}
	payloads := [][]byte{header.Raw(), message.Content}
	return payloads
}

func aNode(ctx context.Context, logger log.Logger) *nodeHarness {
	address := aKey()
	peers := aTopologyContaining()
	cfg := config.ForDirectTransportTests(address, peers, 20*time.Hour /*disable keep alive*/, 1*time.Second)
	topology := memory.NewTopologyProvider(cfg, logger)
	transport := NewDirectTransport(ctx, topology, cfg, logger, metric.NewRegistry())
	listener := &testkit.MockTransportListener{}
	transport.RegisterListener(listener, address)
	return &nodeHarness{topology, transport, address, listener}
}

var currentNodeIndex = 1

func aKey() primitives.NodeAddress {
	address := keys.EcdsaSecp256K1KeyPairForTests(currentNodeIndex).NodeAddress()
	currentNodeIndex++
	return address
}

func aTopologyContaining(nodes ...*nodeHarness) adapter.GossipPeers {
	peers := make(adapter.GossipPeers)
	for _, node := range nodes {
		peers[node.address.KeyForMap()] = node.toGossipPeer()
	}
	return peers
}

func shutdownAll(ctx context.Context, nodes ...*nodeHarness) {
	for _, node := range nodes {
		node.transport.GracefulShutdown(ctx)
	}
}

func superviseAll(s govnr.Supervisor, nodes ...*nodeHarness) {
	for _, node := range nodes {
		s.Supervise(node.transport)
	}
}
