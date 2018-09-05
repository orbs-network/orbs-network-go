package adapter

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net"
	"os"
	"testing"
	"time"
)

const networkSize = 3

type harness struct {
	config    Config
	transport *directTransport
	myPort    uint16
}

func newHarness() *harness {
	// randomize listen port between tests to reduce flakiness and chances of listening clashes
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	firstRandomPort := 20000 + r.Intn(40000)

	federationNodes := make(map[string]config.FederationNode)
	for i := 0; i < networkSize; i++ {
		publicKey := keys.Ed25519KeyPairForTests(i).PublicKey()
		federationNodes[publicKey.KeyForMap()] = config.NewHardCodedFederationNode(publicKey, uint16(firstRandomPort+i), "127.0.0.1")
	}

	cfg := config.EmptyConfig()
	cfg.SetNodePublicKey(keys.Ed25519KeyPairForTests(0).PublicKey())
	cfg.SetFederationNodes(federationNodes)
	cfg.SetDuration(config.GOSSIP_CONNECTION_KEEP_ALIVE_INTERVAL, 1*time.Millisecond)

	port := uint16(firstRandomPort)

	return &harness{
		config:    cfg,
		transport: nil,
		myPort:    port,
	}
}

func (h *harness) start(ctx context.Context) *harness {
	log := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

	h.transport = NewDirectTransport(ctx, h.config, log).(*directTransport)

	// to synchronize tests, wait until server is ready
	test.Eventually(func() bool {
		return h.transport.isServerReady()
	})

	return h
}

func (h *harness) portForPeer(index int) uint16 {
	peerPublicKey := keys.Ed25519KeyPairForTests(index + 1).PublicKey()
	return h.config.FederationNodes(0)[peerPublicKey.KeyForMap()].GossipPort()
}

func TestIncomingConnectionsAreListenedToWhileContextIsLive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := newHarness().start(ctx)

	connection, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", h.myPort))
	defer connection.Close()
	require.NoError(t, err, "should connect to local transport")

	cancel()

	buffer := []byte{0}
	connection.SetDeadline(time.Now().Add(1 * time.Minute))
	read, err := connection.Read(buffer)
	require.Equal(t, 0, read, "should disconnect from peer without reading anything")
	require.Error(t, err, "should disconnect from peer")
}

func TestOutgoingConnectionsToAllPeersOnInitWhileContextIsLive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	h := newHarness()

	var err error
	listeners := make([]net.Listener, networkSize-1)
	for i := 0; i < networkSize-1; i++ {
		listeners[i], err = net.Listen("tcp", fmt.Sprintf(":%d", h.portForPeer(i)))
		defer listeners[i].Close()
		require.NoError(t, err, "peer server could not listen")
	}

	h.start(ctx)

	connections := make([]net.Conn, networkSize-1)
	for i := 0; i < networkSize-1; i++ {
		connections[i], err = listeners[i].Accept()
		defer connections[i].Close()
		require.NoError(t, err, "peer server could not accept")
	}

	cancel()

	for i := 0; i < networkSize-1; i++ {
		buffer := []byte{0}
		read, err := connections[i].Read(buffer)
		require.Equal(t, 0, read, "should disconnect from peer without reading anything")
		require.Error(t, err, "should disconnect from peer")
	}
}

func TestOutgoingConnectionReconnectsOnFailure(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		h := newHarness().start(ctx)

		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", h.portForPeer(0)))
		defer listener.Close()
		require.NoError(t, err, "peer server could not listen")

		connection, err := listener.Accept()
		defer connection.Close()
		require.NoError(t, err, "peer server could not accept")

		for i := 0; i < 3; i++ {
			connection.Close()

			connection, err = listener.Accept()
			require.NoError(t, err, "peer server could not accept")
			fmt.Printf("i connection, err = listener.Accept()\n")
		}
	})
}
