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
	transport *directTransport
	port      uint16
}

func newHarness(ctx context.Context) *harness {
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

	log := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))

	transport := NewDirectTransport(ctx, cfg, log).(*directTransport)
	port := uint16(firstRandomPort)

	// to synchronize tests, wait until server is ready
	test.Eventually(func() bool {
		return transport.isServerReady()
	})

	return &harness{
		transport: transport,
		port:      port,
	}
}

func TestListeningForIncomingConnectionsWhileContextIsLive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	h := newHarness(ctx)

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", h.port))
	require.NoError(t, err, "should connect to local transport")
	conn.Close()

	cancel()

	_, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", h.port))
	require.Error(t, err, "should not connect to local transport")

	// wait until lingering goroutines shut down
	time.Sleep(1 * time.Millisecond)
}
