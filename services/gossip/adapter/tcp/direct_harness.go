// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/testkit"
	"github.com/orbs-network/orbs-network-go/test"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
	"time"
)

const NETWORK_SIZE = 3
const TEST_KEEP_ALIVE_INTERVAL = 20 * time.Millisecond
const TEST_NETWORK_TIMEOUT = 1 * time.Second

const HARNESS_PEER_READ_TIMEOUT = 1 * time.Second
const HARNESS_OUTGOING_CONNECTIONS_INIT_TIMEOUT = 3 * time.Second

type directHarness struct {
	config    config.GossipTransportConfig
	transport *DirectTransport

	peersListeners            []net.Listener
	peersListenersConnections []net.Conn
	peerTalkerConnection      net.Conn
	listenerMock              *testkit.MockTransportListener
}

func newDirectHarnessWithConnectedPeers(t *testing.T, ctx context.Context) *directHarness {
	keepAliveInterval := TEST_KEEP_ALIVE_INTERVAL
	networkTimeout := TEST_NETWORK_TIMEOUT
	return newDirectHarnessWithConnectedPeersWithTimeouts(t, ctx, keepAliveInterval, networkTimeout)
}

func newDirectHarnessWithConnectedPeersWithoutKeepAlives(t *testing.T, ctx context.Context) *directHarness {
	keepAliveInterval := 20 * time.Hour // High value to disable keep alive
	networkTimeout := TEST_NETWORK_TIMEOUT
	return newDirectHarnessWithConnectedPeersWithTimeouts(t, ctx, keepAliveInterval, networkTimeout)
}

func newDirectHarnessWithConnectedPeersWithTimeouts(t *testing.T, ctx context.Context, keepAliveInterval time.Duration, networkTimeout time.Duration) *directHarness {

	// order matters here
	gossipPeers, peersListeners := makePeers(t)                                           // step 1: create the peer server listeners to reserve random TCP ports
	cfg := config.ForDirectTransportTests(gossipPeers, keepAliveInterval, networkTimeout) // step 2: create the config given the peer pk/port pairs
	transport := makeTransport(ctx, t, cfg)                                               // step 3: create the transport; it will attempt to establish connections with the peer servers repeatedly until they start accepting connections
	// end of section where order matters

	peerTalkerConnection := establishPeerClient(t, transport.GetServerPort())      // establish connection from test to server port ( test harness ==> SUT )
	peersListenersConnections := establishPeerServerConnections(t, peersListeners) // establish connection from transport clients to peer servers ( SUT ==> test harness)

	h := &directHarness{
		config:                    cfg,
		transport:                 transport,
		listenerMock:              &testkit.MockTransportListener{},
		peerTalkerConnection:      peerTalkerConnection,
		peersListenersConnections: peersListenersConnections,
		peersListeners:            peersListeners,
	}

	// prevents race condition where client loop still did not flip the outgoing queue's `disabled` flag after successfully "dialing" to the harness
	require.True(t, test.Eventually(HARNESS_OUTGOING_CONNECTIONS_INIT_TIMEOUT, func() bool {
		return h.allOutgoingQueuesEnabled()
	}), "expected all outgoing queues to become enabled after successfully connecting to peersListeners")

	return h
}

func makeTransport(ctx context.Context, tb testing.TB, cfg config.GossipTransportConfig) *DirectTransport {
	log := log.DefaultTestingLogger(tb)
	registry := metric.NewRegistry()
	transport := NewDirectTransport(ctx, cfg, log, registry)
	// to synchronize tests, wait until server is ready
	test.Eventually(test.EVENTUALLY_ADAPTER_TIMEOUT, func() bool {
		return transport.IsServerListening()
	})
	return transport
}

func establishPeerClient(t *testing.T, serverPort int) net.Conn {
	peerTalkerConnection, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", serverPort))
	require.NoError(t, err, "test should be able connect to local transport")
	return peerTalkerConnection
}

func establishPeerServerConnections(t *testing.T, peersListeners []net.Listener) []net.Conn {
	peersListenersConnections := make([]net.Conn, NETWORK_SIZE-1)
	for i := 0; i < NETWORK_SIZE-1; i++ {
		conn, err := peersListeners[i].Accept()
		require.NoError(t, err, "test peer server could not accept connection from local transport")

		peersListenersConnections[i] = conn
	}
	return peersListenersConnections
}

func makePeers(t *testing.T) (map[string]config.GossipPeer, []net.Listener) {
	gossipPeers := make(map[string]config.GossipPeer)
	peersListeners := make([]net.Listener, NETWORK_SIZE-1)

	for i := 0; i < NETWORK_SIZE-1; i++ {
		nodeAddress := testKeys.EcdsaSecp256K1KeyPairForTests(i + 1).NodeAddress()

		conn, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err, "test peer server could not listen")

		peersListeners[i] = conn
		port := conn.Addr().(*net.TCPAddr).Port
		gossipPeers[nodeAddress.KeyForMap()] = config.NewHardCodedGossipPeer(port, "127.0.0.1")
	}
	return gossipPeers, peersListeners
}

func (h *directHarness) peerListenerReadTotal(peerIndex int, totalSize int) ([]byte, error) {
	buffer := make([]byte, totalSize)
	totalRead := 0
	for totalRead < totalSize {
		conn := h.peersListenersConnections[peerIndex]
		err := conn.SetReadDeadline(time.Now().Add(HARNESS_PEER_READ_TIMEOUT)) // apply an arbitrary read timeout
		read, err := conn.Read(buffer[totalRead:])
		if err != nil {
			return nil, err
		}
		totalRead += read
		if totalRead == totalSize {
			break
		}
	}
	return buffer, nil
}

func (h *directHarness) cleanupConnectedPeers() {
	h.peerTalkerConnection.Close()
	for i := 0; i < NETWORK_SIZE-1; i++ {
		h.peersListenersConnections[i].Close()
		h.peersListeners[i].Close()
	}
}

func (h *directHarness) reconnect(listenerIndex int) error {
	h.peersListenersConnections[listenerIndex].Close()    // disconnect transport forcefully
	conn, err := h.peersListeners[listenerIndex].Accept() // reconnect transport forcefully
	h.peersListenersConnections[listenerIndex] = conn

	return err
}

func (h *directHarness) nodeAddressForPeer(index int) primitives.NodeAddress {
	return testKeys.EcdsaSecp256K1KeyPairForTests(index + 1).NodeAddress()
}

func (h *directHarness) portForPeer(index int) int {
	peerPublicKey := h.nodeAddressForPeer(index)
	return h.config.GossipPeers()[peerPublicKey.KeyForMap()].GossipPort()
}

func (h *directHarness) expectTransportListenerCalled(payloads [][]byte) {
	h.listenerMock.When("OnTransportMessageReceived", mock.Any, payloads).Return().Times(1)
}

func (h *directHarness) verifyTransportListenerCalled(t *testing.T) {
	err := test.EventuallyVerify(test.EVENTUALLY_ADAPTER_TIMEOUT, h.listenerMock)
	require.NoError(t, err, "transport listener mock should be called as expected")
}

func (h *directHarness) expectTransportListenerNotCalled() {
	h.listenerMock.When("OnTransportMessageReceived", mock.Any, mock.Any).Return().Times(0)
}

func (h *directHarness) verifyTransportListenerNotCalled(t *testing.T) {
	err := test.ConsistentlyVerify(test.CONSISTENTLY_ADAPTER_TIMEOUT, h.listenerMock)
	require.NoError(t, err, "transport listener mock should be called as expected")
}

func (h *directHarness) allOutgoingQueuesEnabled() bool {
	return h.transport.allOutgoingQueuesEnabled()
}

func concatSlices(slices ...[]byte) []byte {
	var tmp []byte
	for _, s := range slices {
		tmp = append(tmp, s...)
	}
	return tmp
}

// encoded examples of the gossip wire protocol spec:
// https://github.com/orbs-network/orbs-spec/blob/master/encoding/gossip/membuffers-over-tcp.md

func exampleWireProtocolEncoding_Payloads_0x11_0x2233() []byte {
	// encoding payloads: [][]byte{{0x11}, {0x22, 0x33}}
	field_NumPayloads := []byte{0x02, 0x00, 0x00, 0x00}      // little endian
	field_FirstPayloadSize := []byte{0x01, 0x00, 0x00, 0x00} // little endian
	field_FirstPayloadData := []byte{0x11}
	field_FirstPayloadPadding := []byte{0x00, 0x00, 0x00}     // round payload data to 4 bytes
	field_SecondPayloadSize := []byte{0x02, 0x00, 0x00, 0x00} // little endian
	field_SecondPayloadData := []byte{0x22, 0x33}
	field_SecondPayloadPadding := []byte{0x00, 0x00} // round payload data to 4 bytes
	return concatSlices(field_NumPayloads, field_FirstPayloadSize, field_FirstPayloadData, field_FirstPayloadPadding, field_SecondPayloadSize, field_SecondPayloadData, field_SecondPayloadPadding)
}

func exampleWireProtocolEncoding_CorruptNumPayloads() []byte {
	field_NumPayloads := []byte{0x99, 0x99, 0x99, 0x99}      // corrupt value (too big)
	field_FirstPayloadSize := []byte{0x01, 0x00, 0x00, 0x00} // little endian
	field_FirstPayloadData := []byte{0x11}
	field_FirstPayloadPadding := []byte{0x00, 0x00, 0x00} // round payload data to 4 bytes
	return concatSlices(field_NumPayloads, field_FirstPayloadSize, field_FirstPayloadData, field_FirstPayloadPadding)
}

func exampleWireProtocolEncoding_CorruptPayloadSize() []byte {
	field_NumPayloads := []byte{0x01, 0x00, 0x00, 0x00}      // little endian
	field_FirstPayloadSize := []byte{0x99, 0x99, 0x99, 0x99} // corrupt value (too big)
	field_FirstPayloadData := []byte{0x11}
	field_FirstPayloadPadding := []byte{0x00, 0x00, 0x00} // round payload data to 4 bytes
	return concatSlices(field_NumPayloads, field_FirstPayloadSize, field_FirstPayloadData, field_FirstPayloadPadding)
}

func exampleWireProtocolEncoding_KeepAlive() []byte {
	// encoding payloads: [][]byte{} (this is how a keep alive looks like = zero payloads)
	field_NumPayloads := []byte{0x00, 0x00, 0x00, 0x00} // little endian
	return concatSlices(field_NumPayloads)
}
