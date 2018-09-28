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
	"time"
)

const NETWORK_SIZE = 3

type directHarness struct {
	config    Config
	transport *directTransport

	peersListeners            []net.Listener
	peersListenersConnections []net.Conn
	peerTalkerConnection      net.Conn
	listenerMock              *transportListenerMock
}

func newDirectHarnessWithConnectedPeers(t *testing.T, ctx context.Context) *directHarness {
	// randomize listen port between tests to reduce flakiness and chances of listening clashes

	gossipPeers := make(map[string]config.GossipPeer)
	peersListeners := make([]net.Listener, NETWORK_SIZE-1)
	peersListenersConnections := make([]net.Conn, NETWORK_SIZE-1)

	for i := 0; i < NETWORK_SIZE-1; i++ {
		publicKey := keys.Ed25519KeyPairForTests(i + 1).PublicKey()
		conn, err := net.Listen("tcp", ":0")
		require.NoError(t, err, "test peer server could not listen")

		peersListeners[i] = conn
		gossipPeers[publicKey.KeyForMap()] = config.NewHardCodedGossipPeer(uint16(conn.Addr().(*net.TCPAddr).Port), "127.0.0.1")
	}

	log := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

	cfg := config.EmptyConfig()
	cfg.SetNodePublicKey(keys.Ed25519KeyPairForTests(0).PublicKey())
	cfg.SetGossipPeers(gossipPeers)
	cfg.SetUint32(config.GOSSIP_LISTEN_PORT, 0)
	cfg.SetDuration(config.GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 20*time.Millisecond)
	cfg.SetDuration(config.GOSSIP_NETWORK_TIMEOUT, 20*time.Millisecond)
	transport := NewDirectTransport(ctx, cfg, log).(*directTransport)

	// to synchronize tests, wait until server is ready
	test.Eventually(test.EVENTUALLY_ADAPTER_TIMEOUT, func() bool {
		return transport.isServerListening()
	})


	for i := 0; i < NETWORK_SIZE-1; i++ {
		conn, err := peersListeners[i].Accept()
		require.NoError(t, err, "test peer server could not accept connection from local transport")

		peersListenersConnections[i] = conn
	}

	peerTalkerConnection, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", transport.serverPort))
	require.NoError(t, err, "test should be able connect to local transport")

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
	h.listenerMock.When("OnTransportMessageReceived", payloads).Return().Times(1)
}

func (h *directHarness) verifyTransportListenerCalled(t *testing.T) {
	err := test.EventuallyVerify(test.EVENTUALLY_ADAPTER_TIMEOUT, h.listenerMock)
	require.NoError(t, err, "transport listener mock should be called as expected")
}

func (h *directHarness) expectTransportListenerNotCalled() {
	h.listenerMock.When("OnTransportMessageReceived", mock.Any).Return().Times(0)
}

func (h *directHarness) verifyTransportListenerNotCalled(t *testing.T) {
	err := test.ConsistentlyVerify(test.CONSISTENTLY_ADAPTER_TIMEOUT, h.listenerMock)
	require.NoError(t, err, "transport listener mock should be called as expected")
}
