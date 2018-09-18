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
	"math/rand"
	"net"
	"os"
	"testing"
	"time"
)

const NETWORK_SIZE = 3

type directHarness struct {
	config    Config
	transport *directTransport
	myPort    uint16

	peersListeners            []net.Listener
	peersListenersConnections []net.Conn
	peerTalkerConnection      net.Conn
	listenerMock              *transportListenerMock
}

func newDirectHarness() *directHarness {
	// randomize listen port between tests to reduce flakiness and chances of listening clashes
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	firstRandomPort := 20000 + r.Intn(40000)

	gossipPeers := make(map[string]config.GossipPeer)
	for i := 0; i < NETWORK_SIZE; i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		gossipPeers[publicKey.KeyForMap()] = config.NewHardCodedGossipPeer(uint16(firstRandomPort+i), "127.0.0.1")
	}

	cfg := config.EmptyConfig()
	cfg.SetNodePublicKey(keys.Ed25519KeyPairForTests(0).PublicKey())
	cfg.SetGossipPeers(gossipPeers)
	cfg.SetUint16(config.GOSSIP_LISTEN_PORT, uint16(firstRandomPort))
	cfg.SetDuration(config.GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 20*time.Millisecond)
	cfg.SetDuration(config.GOSSIP_NETWORK_TIMEOUT, 20*time.Millisecond)

	port := uint16(firstRandomPort)

	return &directHarness{
		config:       cfg,
		transport:    nil,
		myPort:       port,
		listenerMock: &transportListenerMock{},
	}
}

func (h *directHarness) start(ctx context.Context) *directHarness {
	log := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

	h.transport = NewDirectTransport(ctx, h.config, log).(*directTransport)

	// to synchronize tests, wait until server is ready
	test.Eventually(test.EVENTUALLY_ADAPTER_TIMEOUT, func() bool {
		return h.transport.isServerReady()
	})

	return h
}

func newDirectHarnessWithConnectedPeers(t *testing.T, ctx context.Context) *directHarness {
	h := newDirectHarness()

	var err error
	h.peersListeners = make([]net.Listener, NETWORK_SIZE-1)
	for i := 0; i < NETWORK_SIZE-1; i++ {
		h.peersListeners[i], err = net.Listen("tcp", fmt.Sprintf(":%d", h.portForPeer(i)))
		require.NoError(t, err, "test peer server could not listen")
	}

	h.start(ctx)

	h.peersListenersConnections = make([]net.Conn, NETWORK_SIZE-1)
	for i := 0; i < NETWORK_SIZE-1; i++ {
		h.peersListenersConnections[i], err = h.peersListeners[i].Accept()
		require.NoError(t, err, "test peer server could not accept connection from local transport")
	}

	h.peerTalkerConnection, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", h.myPort))
	require.NoError(t, err, "test should be able connect to local transport")

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
