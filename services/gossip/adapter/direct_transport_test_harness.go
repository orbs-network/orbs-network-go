package adapter

import (
	"context"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"net"
	"os"
	"testing"
)

const NETWORK_SIZE = 3

type directHarness struct {
	config    config.GossipTransportConfig
	transport *directTransport

	peersListeners            []net.Listener
	peersListenersConnections []net.Conn
	peerTalkerConnection      net.Conn
	listenerMock              *transportListenerMock
}

func newDirectHarnessWithConnectedPeers(t *testing.T, ctx context.Context) *directHarness {

	// order matters here
	gossipPeers, peersListeners := makePeers(t)        // step 1: create the peer server listeners to reserve random TCP ports
	cfg := config.ForDirectTransportTests(gossipPeers) // step 2: create the config given the peer pk/port pairs
	transport := makeTransport(ctx, cfg)               // step 3: create the transport; it will attempt to establish connections with the peer servers repeatedly until they start accepting connections
	// end of section where order matters

	peerTalkerConnection := establishPeerClient(t, transport.serverPort)           // establish connection from test to server port ( test harness ==> SUT )
	peersListenersConnections := establishPeerServerConnections(t, peersListeners) // establish connection from transport clients to peer servers ( SUT ==> test harness)

	h := &directHarness{
		config:                    cfg,
		transport:                 transport,
		listenerMock:              &transportListenerMock{},
		peerTalkerConnection:      peerTalkerConnection,
		peersListenersConnections: peersListenersConnections,
		peersListeners:            peersListeners,
	}

	return h
}

func makeTransport(ctx context.Context, cfg config.GossipTransportConfig) *directTransport {
	log := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))
	transport := NewDirectTransport(ctx, cfg, log).(*directTransport)
	// to synchronize tests, wait until server is ready
	test.Eventually(test.EVENTUALLY_ADAPTER_TIMEOUT, func() bool {
		return transport.isServerListening()
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
		publicKey := keys.Ed25519KeyPairForTests(i + 1).PublicKey()
		conn, err := net.Listen("tcp", ":0")
		require.NoError(t, err, "test peer server could not listen")

		peersListeners[i] = conn
		gossipPeers[publicKey.KeyForMap()] = config.NewHardCodedGossipPeer(uint16(conn.Addr().(*net.TCPAddr).Port), "127.0.0.1")
	}
	return gossipPeers, peersListeners
}

func (h *directHarness) peerListenerReadTotal(peerIndex int, totalSize int) ([]byte, error) {
	buffer := make([]byte, totalSize)
	totalRead := 0
	for totalRead < totalSize {
		read, err := h.peersListenersConnections[peerIndex].Read(buffer[totalRead:])
		totalRead += read
		if totalRead == totalSize {
			break
		}
		if err != nil {
			return nil, err
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

func (h *directHarness) publicKeyForPeer(index int) primitives.Ed25519PublicKey {
	return keys.Ed25519KeyPairForTests(index + 1).PublicKey()
}

func (h *directHarness) portForPeer(index int) uint16 {
	peerPublicKey := h.publicKeyForPeer(index)
	return h.config.GossipPeers(0)[peerPublicKey.KeyForMap()].GossipPort()
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
