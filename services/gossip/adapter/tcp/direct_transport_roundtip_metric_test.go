//+build !race
// Histogram metrics are disabled in race mode

package tcp

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/stretchr/testify/require"
	"net"
	"regexp"
	"testing"
	"time"
)

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

type silentServer struct {
	listener net.Listener
}

func startSilentServer(port int) (*silentServer, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	return &silentServer{listener: listener}, err
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
