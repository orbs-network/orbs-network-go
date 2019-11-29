// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"net"
	"regexp"
	"testing"
	"time"
)

func TestDirectTransport_HandlesStartupWithEmptyPeerList(t *testing.T) {
	address := keys.EcdsaSecp256K1KeyPairForTests(0).NodeAddress()
	cfg := config.ForDirectTransportTests(address, make(GossipPeers), 20*time.Hour /*disable keep alive*/, 1*time.Second)
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		transport := NewDirectTransport(ctx, cfg, harness.Logger, metric.NewRegistry())
		harness.Supervise(transport)
		defer transport.GracefulShutdown(ctx)

		require.True(t, test.Eventually(test.EVENTUALLY_ADAPTER_TIMEOUT, func() bool {
			return transport.IsServerListening()
		}), "server did not start")
	})
}

func TestDirectTransport_SupportsTopologyChangeInRuntime(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		node1 := aNode(ctx, harness.Logger)
		node2 := aNode(ctx, harness.Logger)
		node3 := aNode(ctx, harness.Logger)
		node4 := aNode(ctx, harness.Logger)
		superviseAll(harness, node1, node2, node3, node4)
		defer shutdownAll(ctx, node1, node2, node3, node4)

		waitForAllNodesToSatisfy(t, "server did not start", func(node *nodeHarness) bool { return node.transport.IsServerListening() }, node1, node2, node3, node4)

		firstTopology := aTopologyContaining(node1, node2, node3)
		node1.transport.UpdateTopology(ctx, firstTopology)
		node2.transport.UpdateTopology(ctx, firstTopology)
		node3.transport.UpdateTopology(ctx, firstTopology)

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
		node1.transport.UpdateTopology(ctx, secondTopology)
		node2.transport.UpdateTopology(ctx, secondTopology)
		node4.transport.UpdateTopology(ctx, secondTopology)

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
		require.Error(t, node2.transport.Send(ctx, &adapter.TransportData{
			SenderNodeAddress:      node2.address,
			RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
			RecipientNodeAddresses: []primitives.NodeAddress{node3.address},
			Payloads:               aMessage(),
		}), "node 2 was able to send a message to node 3 which is no longer a part of its topology")
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
		node1.transport.UpdateTopology(ctx, firstTopology)
		node2.transport.UpdateTopology(ctx, firstTopology)
		node3.transport.UpdateTopology(ctx, firstTopology)

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

func TestDirectTransport_UpdatesRoundtripMetric(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		node1 := aNode(ctx, harness.Logger)
		node2 := aNode(ctx, harness.Logger)
		superviseAll(harness, node1, node2)
		defer shutdownAll(ctx, node1, node2)

		topology := aTopologyContaining(node1, node2)
		node1.transport.UpdateTopology(ctx, topology)
		node2.transport.UpdateTopology(ctx, topology)

		waitForAllNodesToSatisfy(t,
			"expected all nodes to have peers added",
			func(node *nodeHarness) bool { return len(node.transport.outgoingConnections.activeConnections) > 0 },
			node1, node2)

		waitForAllNodesToSatisfy(t,
			"expected all outgoing queues to become enabled after topology change",
			func(node *nodeHarness) bool { return node.transport.allOutgoingQueuesEnabled() },
			node1, node2)

		roundtrip1to2 := node1.metricRegistry.Get(fmt.Sprintf("Gossip.OutgoingConnection.Roundtrip.%s", node2.address.String()[:6]))
		roundtrip2to1 := node2.metricRegistry.Get(fmt.Sprintf("Gossip.OutgoingConnection.Roundtrip.%s", node1.address.String()[:6]))

		for i := 1; i <= 5; i++ {
			node1.requireSendsSuccessfullyTo(t, ctx, node2)
			node2.requireSendsSuccessfullyTo(t, ctx, node1)

			require.True(t, test.Eventually(1*time.Second, func() bool {
				matched, err := regexp.MatchString(fmt.Sprintf("samples=%d", i), roundtrip1to2.String())
				require.NoError(t, err)
				return matched
			}), "expected 1->2 roundtrip metric to update")

			require.True(t, test.Eventually(1*time.Second, func() bool {
				matched, err := regexp.MatchString(fmt.Sprintf("samples=%d", i), roundtrip2to1.String())
				require.NoError(t, err)
				return matched
			}), "expected 2->1 roundtrip metric to update")
		}
	})
}

func replaceNodeWithSilentServer(t *testing.T, ctx context.Context, node *nodeHarness) *silentServer {
	node.transport.GracefulShutdown(ctx)

	var server *silentServer
	var err error
	test.Eventually(time.Second, func() bool {
		server, err = startSilentServer(node.transport.GetServerPort())
		return err == nil
	})
	require.NoError(t, err, "silent server failed to start")
	return server
}

func TestDirectTransport_RoundtripMetricReflectsLatency(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		node1 := aNode(ctx, harness.Logger)
		node2 := aNode(ctx, harness.Logger)
		superviseAll(harness, node1, node2)
		defer shutdownAll(ctx, node1, node2)

		server := replaceNodeWithSilentServer(t, ctx, node2)

		topology := aTopologyContaining(node1, node2)
		node1.transport.UpdateTopology(ctx, topology)

		inConn, outConn := server.connectTo(t, node1)
		defer inConn.Close()
		defer outConn.Close()

		waitForAllNodesToSatisfy(t,
			"expected all nodes to have peers added",
			func(node *nodeHarness) bool { return len(node.transport.outgoingConnections.activeConnections) > 0 },
			node1)

		waitForAllNodesToSatisfy(t,
			"expected all outgoing queues to become enabled after topology change",
			func(node *nodeHarness) bool { return node.transport.allOutgoingQueuesEnabled() },
			node1)

		payloads := aMessage()

		require.NoError(t, node1.transport.Send(ctx, &adapter.TransportData{
			SenderNodeAddress:      node1.address,
			RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
			RecipientNodeAddresses: []primitives.NodeAddress{node2.address},
			Payloads:               payloads,
		}))

		time.Sleep(200 * time.Millisecond) // create an artificial latency

		roundtripMetric := node1.metricRegistry.Get(fmt.Sprintf("Gossip.OutgoingConnection.Roundtrip.%s", node2.address.String()[:6]))

		// No acks
		matched, err := regexp.MatchString(fmt.Sprintf("samples=0"), roundtripMetric.String())
		require.NoError(t, err)
		require.True(t, matched, "expected 0 samples in the roundtrip metric")

		// Send ack
		_, err = inConn.Write(ACK_BUFFER)
		require.NoError(t, err, "couldn't write ack")

		// see that the metric has updated
		require.True(t, test.Eventually(1*time.Second, func() bool {
			matched, err := regexp.MatchString(fmt.Sprintf("samples=1"), roundtripMetric.String())
			require.NoError(t, err)
			return matched
		}), "expected roundtrip metric to update")
		require.True(t, test.Eventually(1*time.Second, func() bool {
			matched, err := regexp.MatchString(fmt.Sprintf("max=2\\d\\d\\."), roundtripMetric.String())
			require.NoError(t, err)
			return matched
		}), "expected roundtrip metric to be in the range of 2XX ms")

	})
}

func TestDirectTransport_SendsMessagesWhenAcksAreDelayed(t *testing.T) {
	with.Concurrency(t, func(ctx context.Context, harness *with.ConcurrencyHarness) {
		node1 := aNode(ctx, harness.Logger)
		node2 := aNode(ctx, harness.Logger)
		superviseAll(harness, node1, node2)
		defer shutdownAll(ctx, node1, node2)

		server := replaceNodeWithSilentServer(t, ctx, node2)

		topology := aTopologyContaining(node1, node2)
		node1.transport.UpdateTopology(ctx, topology)

		inConn, outConn := server.connectTo(t, node1)
		defer inConn.Close()
		defer outConn.Close()

		waitForAllNodesToSatisfy(t,
			"expected all nodes to have peers added",
			func(node *nodeHarness) bool { return len(node.transport.outgoingConnections.activeConnections) > 0 },
			node1)

		waitForAllNodesToSatisfy(t,
			"expected all outgoing queues to become enabled after topology change",
			func(node *nodeHarness) bool { return node.transport.allOutgoingQueuesEnabled() },
			node1)

		payloads := aMessage()

		for i := 0; i < 10*MaxQueueSize; i++ {
			require.NoError(t, node1.transport.Send(ctx, &adapter.TransportData{
				SenderNodeAddress:      node1.address,
				RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
				RecipientNodeAddresses: []primitives.NodeAddress{node2.address},
				Payloads:               payloads,
			}))
		}

		readUntilTimeout(t, inConn) // make sure processing of the outgoing messages is finished

		roundtripMetric := node1.metricRegistry.Get(fmt.Sprintf("Gossip.OutgoingConnection.Roundtrip.%s", node2.address.String()[:6]))

		// No acks
		matched, err := regexp.MatchString(fmt.Sprintf("samples=0"), roundtripMetric.String())
		require.NoError(t, err)
		require.True(t, matched, "expected 0 samples in the roundtrip metric")

		// send back 10*MaxQueueSize acks
		for i := 0; i < 10*MaxQueueSize; i++ {
			_, err := inConn.Write(ACK_BUFFER)
			require.NoError(t, err, "couldn't write ack")
		}

		// see that the metric has updated
		require.True(t, test.Eventually(1*time.Second, func() bool {
			matched, err := regexp.MatchString(fmt.Sprintf("samples=%d", MaxQueueSize), roundtripMetric.String())
			require.NoError(t, err)
			return matched
		}), "expected roundtrip metric to update exactly MaxQueueSize times")

	})
}

type nodeHarness struct {
	transport      *DirectTransport
	address        primitives.NodeAddress
	listener       *testkit.MockTransportListener
	metricRegistry metric.Registry
}

type silentServer struct {
	listener net.Listener
}

func (server *silentServer) connectTo(t *testing.T, node *nodeHarness) (net.Conn, net.Conn) {
	inConn, err := server.listener.Accept()
	require.NoError(t, err, "could not receive connection from node")

	var outConn net.Conn
	test.Eventually(time.Second, func() bool {
		outConn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", node.toGossipPeer().GossipEndpoint(), node.toGossipPeer().GossipPort()))
		return err == nil
	})
	require.NoError(t, err, "could not connect to node1")
	return inConn, outConn
}

func startSilentServer(port int) (*silentServer, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	return &silentServer{listener: listener}, err
}

func readUntilTimeout(t *testing.T, conn net.Conn) {
	buffer := make([]byte, 1024, 1024)
	test.Eventually(5*time.Second, func() bool {
		err := conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		require.NoError(t, err)
		_, err = conn.Read(buffer)
		if isTimeoutError(err) {
			return true
		}
		require.NoError(t, err)
		return false
	})
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

func (n *nodeHarness) toGossipPeer() config.GossipPeer {
	return config.NewHardCodedGossipPeer(n.transport.GetServerPort(), "127.0.0.1", hex.EncodeToString(n.address))
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
	metricRegistry := metric.NewRegistry()
	transport := NewDirectTransport(ctx, cfg, logger, metricRegistry)
	listener := &testkit.MockTransportListener{}
	transport.RegisterListener(listener, address)
	return &nodeHarness{transport, address, listener, metricRegistry}
}

var currentNodeIndex = 1

func aKey() primitives.NodeAddress {
	address := keys.EcdsaSecp256K1KeyPairForTests(currentNodeIndex).NodeAddress()
	currentNodeIndex++
	return address
}

func aTopologyContaining(nodes ...*nodeHarness) GossipPeers {
	peers := make(map[string]config.GossipPeer)
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
